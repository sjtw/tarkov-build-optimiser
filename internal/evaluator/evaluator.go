package evaluator

import (
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"sort"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/weapon_tree"
)

type ItemEvaluation struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Slots              []*SlotEvaluation `json:"slots"`
	RecoilModifier     int               `json:"recoil_modifier"`
	ErgonomicsModifier int               `json:"ergonomics_modifier"`
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
	RecoilSum      int `json:"recoil_sum"`
	ErgonomicsSum  int `json:"ergonomics_sum"`
	EvaluationType string
	ExcludedItems  []string
}

func (b *Build) ToEvaluatedWeapon() (EvaluatedWeapon, error) {
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

	// ensure the item & slot maps are up to date with the current state of the weapon tree
	b.WeaponTree.UpdateAllowedItems()
	b.WeaponTree.UpdateAllowedItemSlots()
	remainingItems := make([]OptimalItem, len(b.OptimalItems))
	copy(remainingItems, b.OptimalItems)

	// keep looping through remainingItems until all have been added to the evaluated weapon
	// or until no furter progress can be made.
	misses := 0
	for len(remainingItems) > 0 {
		misses = 0
		for i := 0; i < len(remainingItems); i++ {
			destinationSlot := result.GetSlotById(remainingItems[i].SlotID)
			// item comtaining this slot likely hasn't been added to the evaluated weapon yet
			if destinationSlot == nil {
				misses++
				continue
			}

			source := b.WeaponTree.GetAllowedItem(remainingItems[i].ID)

			evaluated := &ItemEvaluation{
				ID:                 source.ID,
				Name:               source.Name,
				RecoilModifier:     source.RecoilModifier,
				ErgonomicsModifier: source.ErgonomicsModifier,
				Slots:              make([]*SlotEvaluation, len(source.Slots)),
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
			newItems := make([]OptimalItem, 0, len(remainingItems)-1)
			for _, item := range remainingItems {
				if item.ID != evaluated.ID {
					newItems = append(newItems, item)
				}
			}
			remainingItems = newItems
		}

		if len(remainingItems) > 0 && misses == len(remainingItems) {
			log.Error().Msgf("Failed to convert optimal items to evaluated weapon for %s", b.WeaponTree.Item.Name)
			return EvaluatedWeapon{}, errors.New("failed to convert optimal items to evaluated weapon")
		}
	}
	return result, nil
}

func FindBestBuild(weapon *weapon_tree.WeaponTree, focusedStat string,
	excludedItems map[string]bool) *Build {
	uselessItems := map[string]bool{}
	itemHits := map[string]int{}
	memo := map[string]*Build{}
	build := processSlots(weapon.Item.Slots, []OptimalItem{}, focusedStat, 0, 0, excludedItems, []string{}, uselessItems, memo, itemHits)
	build.WeaponTree = *weapon

	//for key, value := range pathTracker {
	//	if value > 1 {
	//		log.Info().Msgf("PathTracker Key: %s, Value: %d", key, value)
	//	}
	//}

	return build
}

// findBestBuild recursively traverses the build tree to find the optimal build.
// It returns the best Build found in the current recursion path.
func processSlots(slotsToProcess []*weapon_tree.ItemSlot, chosenItems []OptimalItem, focusedStat string, recoilStatSum int, ergoStatSum int, excludedItems map[string]bool, currentPath []string, uselessItems map[string]bool, memo map[string]*Build, itemHits map[string]int) *Build {
	//key := createMemoKey(slotsToProcess, focusedStat, recoilStatSum, ergoStatSum, excludedItems)
	//if build, exists := memo[key]; exists {
	//	return build
	//}

	clonedExcludedItems := helpers.CloneMap(excludedItems)

	//for key, value := range pathTracker {
	//	if value > 1 {
	//		log.Info().Msgf("path visited > 1x: %s, Value: %d", key, value)
	//	}
	//}

	if len(slotsToProcess) == 0 {
		exclusions := make([]string, 0)
		for excludedID, isExcluded := range clonedExcludedItems {
			if isExcluded {
				exclusions = append(exclusions, excludedID)
			}
		}
		return &Build{
			OptimalItems:   append([]OptimalItem{}, chosenItems...), // Make a copy
			RecoilSum:      recoilStatSum,
			EvaluationType: focusedStat,
			ExcludedItems:  exclusions,
		}
	}

	// Take the first slot to process
	currentSlot := slotsToProcess[0]
	//currentPath = append(currentPath, currentSlot.ID)
	////visitCount := pathTracker[currentSlot.ID]
	////pathTracker[strings.Join(currentPath, ",")] = visitCount + 1
	//
	remainingSlots := slotsToProcess[1:]
	//for index, slot := range currentPath {
	//	if index != len(currentPath)-1 && slot == currentSlot.ID {
	//		log.Info().Msgf("Recursion detected: %v", currentPath)
	//	}
	//}

	var best *Build = nil

	for _, item := range currentSlot.AllowedItems {
		if excludedItems[item.ID] {
			continue
		}

		if uselessItems[item.ID] {
			continue
		}

		if !canDescendantsImproveStat(item, focusedStat) {
			uselessItems[item.ID] = true
			continue
		}

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

		count := itemHits[item.ID]
		itemHits[item.ID] = count + 1
		itemHitsArray := make([]struct {
			ID    string
			Count int
		}, 0, len(itemHits))

		for id, count := range itemHits {
			itemHitsArray = append(itemHitsArray, struct {
				ID    string
				Count int
			}{ID: id, Count: count})
		}

		sort.Slice(itemHitsArray, func(i, j int) bool {
			return itemHitsArray[i].Count > itemHitsArray[j].Count
		})

		newChosen := append(chosenItems, OptimalItem{
			Name:   item.Name,
			ID:     item.ID,
			SlotID: currentSlot.ID,
		})

		newRecoil := recoilStatSum + item.RecoilModifier
		newErgo := ergoStatSum + item.ErgonomicsModifier

		newSlotsToProcess := append([]*weapon_tree.ItemSlot{}, remainingSlots...)
		if len(item.Slots) > 0 {
			newSlotsToProcess = append(newSlotsToProcess, item.Slots...)

			//newSlotsToProcess = filterSlots(newSlotsToProcess, []string{"Scope", "Tactical", "Ubgl", "Mount", "Foregrip"})
			newSlotsToProcess = filterSlots(newSlotsToProcess, []string{"Scope", "Tactical", "Ubgl"})
		}

		newExcluded := helpers.CloneMap(excludedItems)
		for _, conflictItem := range item.ConflictingItems {
			newExcluded[conflictItem] = true
		}
		candidate := processSlots(newSlotsToProcess, newChosen, focusedStat, newRecoil, newErgo, newExcluded, currentPath, uselessItems, memo, itemHits)

		if candidate != nil {
			if best == nil {
				best = candidate
			}

			if doesImproveStats(candidate, best, focusedStat) {
				best = candidate
			}
		}
	}

	if best == nil {
		newExcluded := helpers.CloneMap(excludedItems)
		candidate := processSlots(remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, newExcluded, currentPath, uselessItems, memo, itemHits)
		if candidate != nil {
			best = candidate
		}
	}

	if best == nil {
		// Return a build even if best is nil
		exclusions := make([]string, 0)
		for excluded, is := range excludedItems {
			if is {
				exclusions = append(exclusions, excluded)
			}
		}
		return &Build{
			OptimalItems:   append([]OptimalItem{}, chosenItems...), // Make a copy
			RecoilSum:      recoilStatSum,
			EvaluationType: focusedStat,
			ExcludedItems:  exclusions,
		}
	}

	//memo[key] = best

	return best
}

func filterSlots(slots []*weapon_tree.ItemSlot, excludedSlotNames []string) []*weapon_tree.ItemSlot {
	filteredSlots := make([]*weapon_tree.ItemSlot, 0)

	if len(slots) == 0 {
		return filteredSlots
	}

	for _, slot := range slots {
		excluded := false
		for _, excludedName := range excludedSlotNames {
			if slot.Name == excludedName {
				excluded = true
			}
		}
		if !excluded {
			filteredSlots = append(filteredSlots, slot)
		}
	}

	return filteredSlots
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

func canDescendantsImproveStat(item *weapon_tree.Item, focusedStat string) bool {
	if canImproveStat(item, focusedStat) {
		return true
	}

	for _, subslot := range item.Slots {
		for _, subitem := range subslot.AllowedItems {
			if canDescendantsImproveStat(subitem, focusedStat) {
				return true
			}
		}
	}

	return false
}

func canImproveStat(item *weapon_tree.Item, focusedStat string) bool {
	if focusedStat == "recoil" && item.RecoilModifier < 0 {
		return true
	} else if focusedStat == "ergonomics" && item.ErgonomicsModifier > 0 {
		return true
	}
	//} else if item.ErgonomicsModifier > 0 || item.RecoilModifier < 0 {
	//	return true
	//}

	return false
}

func doesImproveStats(candidate *Build, best *Build, focusedStat string) bool {
	if focusedStat == "recoil" {
		if candidate.RecoilSum < best.RecoilSum {
			return true
		}
		//} else if candidate.ErgonomicsSum > best.ErgonomicsSum {
		//	return true
		//}
	} else if focusedStat == "ergonomics" {
		if candidate.ErgonomicsSum > best.ErgonomicsSum {
			return true
		} else if candidate.RecoilSum < best.RecoilSum {
			return true
		}
	}

	return false
}

func createMemoKey(slots []*weapon_tree.ItemSlot, focusedStat string, recoilStatSum int, ergoStatSum int, excludedItems map[string]bool) string {
	return focusedStat + "|" +
		serializeSlots(slots) + "|" +
		serializeExcludedItems(excludedItems) + "|" +
		strconv.Itoa(recoilStatSum) + "|" +
		strconv.Itoa(ergoStatSum)
}

func serializeSlots(slots []*weapon_tree.ItemSlot) string {
	slotIDs := make([]string, len(slots))
	for i, slot := range slots {
		slotIDs[i] = slot.ID
	}
	sort.Strings(slotIDs)
	return strings.Join(slotIDs, ",")
}

func serializeExcludedItems(excludedItems map[string]bool) string {
	excludedIDs := make([]string, 0, len(excludedItems))
	for id, excluded := range excludedItems {
		if excluded {
			excludedIDs = append(excludedIDs, id)
		}
	}
	sort.Strings(excludedIDs)
	return strings.Join(excludedIDs, ",")
}
