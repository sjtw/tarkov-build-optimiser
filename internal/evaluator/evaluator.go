package evaluator

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/weapon_tree"
)

type ItemEvaluation struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Slots []*SlotEvaluation `json:"slots"`
}

type SlotEvaluation struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Item    *ItemEvaluation `json:"item"`
	IsEmpty bool            `json:"empty"`
}

func (s *SlotEvaluation) findSlotById(slotID string) *SlotEvaluation {
	if s.ID == slotID {
		return s
	}
	if s.Item == nil {
		return nil
	}
	for _, subSlot := range s.Item.Slots {
		if foundSlot := subSlot.findSlotById(slotID); foundSlot != nil {
			return foundSlot
		}
	}
	return nil
}

func (se *SlotEvaluation) MarshalJSON() ([]byte, error) {
	if se.IsEmpty {
		return json.Marshal(map[string]interface{}{
			"id":   se.ID,
			"name": se.Name,
			"item": nil,
		})
	}

	return json.Marshal(map[string]interface{}{
		"id":   se.ID,
		"name": se.Name,
		"item": se.Item,
	})
}

type EvaluatedWeapon struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	EvaluationType string            `json:"evaluation_type"`
	Slots          []*SlotEvaluation `json:"slots"`
}

func (ew *EvaluatedWeapon) GetSlotById(slotID string) *SlotEvaluation {
	for _, slot := range ew.Slots {
		if foundSlot := slot.findSlotById(slotID); foundSlot != nil {
			return foundSlot
		}
	}
	return nil
}

type OptimalItem struct {
	Name   string
	ID     string
	SlotID string
}

// Build represents a complete weapon configuration.
type Build struct {
	WeaponTree     weapon_tree.WeaponTree
	OptimalItems   []OptimalItem
	FocusedStatSum int
	EvaluationType string
	ExcludedItems  []string
}

func (b *Build) ToEvaluatedWeapon() EvaluatedWeapon {
	result := EvaluatedWeapon{
		ID:             b.WeaponTree.Item.ID,
		Name:           b.WeaponTree.Item.Name,
		EvaluationType: b.EvaluationType,
		Slots:          make([]*SlotEvaluation, 0, len(b.WeaponTree.Item.Slots)),
	}

	for _, slot := range b.WeaponTree.Item.Slots {
		slot := &SlotEvaluation{
			ID:      slot.ID,
			Name:    slot.Name,
			IsEmpty: true,
		}
		result.Slots = append(result.Slots, slot)
	}

	b.WeaponTree.UpdateAllowedItems()
	b.WeaponTree.UpdateAllowedItemSlots()
	items := make([]OptimalItem, len(b.OptimalItems))
	copy(items, b.OptimalItems)

	misses := 0
	failed := false
	for len(items) > 0 {
		misses = 0
		for i := 0; i < len(items); i++ {
			destinationSlot := result.GetSlotById(items[i].SlotID)
			if destinationSlot == nil {
				misses++
				continue
			}

			source := b.WeaponTree.GetAllowedItem(items[i].ID)

			evaluated := &ItemEvaluation{
				ID:    source.ID,
				Name:  source.Name,
				Slots: make([]*SlotEvaluation, len(source.Slots)),
			}

			for j := 0; j < len(source.Slots); j++ {
				evaluated.Slots[j] = &SlotEvaluation{
					ID:      source.Slots[j].ID,
					Name:    source.Slots[j].Name,
					IsEmpty: true,
				}
			}

			destinationSlot.Item = evaluated
			destinationSlot.IsEmpty = false
			items = append(items[:i], items[i+1:]...)
		}

		if len(items) > 0 && misses == len(items) {
			failed = true
			break
		}
	}

	if failed {
		log.Error().Msgf("Failed to convert optimal items to evaluated weapon for %s", b.WeaponTree.Item.Name)
	}

	return result
}

func findBestBuild(weapon *weapon_tree.WeaponTree, chosenItems []OptimalItem, focusedStat string, focusedStatSum int,
	excludedItems map[string]bool) *Build {
	build := processSlots(weapon.Item.Slots, chosenItems, focusedStat, focusedStatSum, excludedItems)
	build.WeaponTree = *weapon

	return build
}

// findBestBuild recursively traverses the build tree to find the optimal build.
// It returns the best Build found in the current recursion path.
func processSlots(slotsToProcess []*weapon_tree.ItemSlot, chosenItems []OptimalItem, focusedStat string, focusedStatSum int,
	excludedItems map[string]bool) *Build {

	// Base Case: No more slots to process
	if len(slotsToProcess) == 0 {
		exclusions := make([]string, 0)
		for excluded, is := range excludedItems {
			if is {
				exclusions = append(exclusions, excluded)
			}
		}
		return &Build{
			OptimalItems:   append([]OptimalItem{}, chosenItems...), // Make a copy
			FocusedStatSum: focusedStatSum,
			EvaluationType: focusedStat,
			ExcludedItems:  exclusions,
		}
	}

	// Take the first slot to process
	currentSlot := slotsToProcess[0]
	remainingSlots := slotsToProcess[1:]

	var best *Build = nil

	// Iterate through each allowed item in the current slot
	for _, item := range currentSlot.AllowedItems {
		// Skip if the item is excluded due to conflicts
		if excludedItems[item.ID] {
			continue
		}

		// Check for conflicts with already chosen items
		conflict := false
		for _, chosen := range chosenItems {
			if conflictsWith(item, chosen) {
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}

		// Include this item in the build
		newChosen := append(chosenItems, OptimalItem{
			Name:   item.Name,
			ID:     item.ID,
			SlotID: currentSlot.ID,
		})
		newRecoil := 0

		if focusedStat == "recoil" {
			newRecoil = focusedStatSum + item.RecoilModifier
		} else {
			newRecoil = focusedStatSum + item.ErgonomicsModifier
		}

		// Create a new exclusion set
		newExcluded := helpers.CloneMap(excludedItems)
		for _, conflictItem := range item.ConflictingItems {
			newExcluded[conflictItem] = true
		}

		// Append subslots opened by this item to the remainingSlots
		newSlotsToProcess := append([]*weapon_tree.ItemSlot{}, remainingSlots...) // Copy to avoid mutation
		if len(item.Slots) > 0 {
			newSlotsToProcess = append(newSlotsToProcess, item.Slots...)
		}

		// Recurse with the updated build state and slots
		candidate := processSlots(newSlotsToProcess, newChosen, focusedStat, newRecoil, newExcluded)

		// Update the best build if necessary
		if candidate != nil {
			if best == nil || candidate.FocusedStatSum > best.FocusedStatSum {
				best = candidate
			}
		}
	}

	// Proceed without choosing any item for this slot
	candidate := processSlots(remainingSlots, chosenItems, focusedStat, focusedStatSum, excludedItems)
	if candidate != nil {
		if best == nil || candidate.FocusedStatSum > best.FocusedStatSum {
			best = candidate
		}
	}

	return best
}

// conflictsWith checks if two items conflict based on the conflict map.
func conflictsWith(item *weapon_tree.Item, chosen OptimalItem) bool {
	for _, conflict := range item.ConflictingItems {
		if conflict == chosen.ID {
			return true
		}
	}
	return false
}
