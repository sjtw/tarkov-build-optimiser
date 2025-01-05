package evaluator

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"sort"
	"strings"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/helpers"
)

type ItemEvaluationConflicts struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type ItemEvaluation struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Slots              []*SlotEvaluation         `json:"slots"`
	RecoilModifier     int                       `json:"recoil_modifier"`
	ErgonomicsModifier int                       `json:"ergonomics_modifier"`
	Conflicts          []ItemEvaluationConflicts `json:"conflicts"`
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
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	EvaluationType string                    `json:"evaluation_type"`
	Slots          []*SlotEvaluation         `json:"slots"`
	Conflicts      []ItemEvaluationConflicts `json:"conflicts"`
	RecoilSum      int                       `json:"recoil_sum"`
	ErgonomicsSum  int                       `json:"ergonomics_sum"`
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
	Name         string
	ID           string
	SlotID       string
	HasConflicts bool
}

// Build represents a complete weapon configuration.
type Build struct {
	WeaponTree     candidate_tree.CandidateTree
	OptimalItems   []OptimalItem
	RecoilSum      int `json:"recoil_sum"`
	ErgonomicsSum  int `json:"ergonomics_sum"`
	EvaluationType string
	ExcludedItems  []string
	HasConflicts   bool
}

func (b *Build) ToEvaluatedWeapon() (EvaluatedWeapon, error) {
	result := EvaluatedWeapon{
		ID:             b.WeaponTree.Item.ID,
		Name:           b.WeaponTree.Item.Name,
		EvaluationType: b.EvaluationType,
		Slots:          make([]*SlotEvaluation, 0, len(b.WeaponTree.Item.Slots)),
		Conflicts:      make([]ItemEvaluationConflicts, 0),
		RecoilSum:      b.RecoilSum,
		ErgonomicsSum:  b.ErgonomicsSum,
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
				Conflicts:          make([]ItemEvaluationConflicts, 0),
			}

			for j := 0; j < len(source.ConflictingItems); j++ {
				conflict := ItemEvaluationConflicts{
					ID:       source.ConflictingItems[j].ID,
					Name:     source.ConflictingItems[j].Name,
					Category: source.ConflictingItems[j].CategoryName,
				}
				evaluated.Conflicts = append(evaluated.Conflicts, conflict)
				result.Conflicts = append(result.Conflicts, conflict)
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

func FindBestBuild(weapon *candidate_tree.CandidateTree, focusedStat string,
	excludedItems map[string]bool) *Build {
	//itemHits := map[string]int{}
	//memo := map[string]*Build{}

	log.Info().Msgf("Finding best build for %s", weapon.Item.Name)
	//for index, slot := range weapon.Item.Slots {
	//	log.Info().Msgf("Processing slot %d", index)
	//	slots := make([]*candidate_tree.ItemSlot, 0)
	//	slots = append(slots, slot)
	//	build := processSlots(slots, []OptimalItem{}, focusedStat, 0, 0, excludedItems, []string{}, uselessItems, memo, itemHits)
	//
	//	log.Info().Msgf("Slot %d, Build: %v", index, build)
	//}
	//slots := []*candidate_tree.ItemSlot{
	//	weapon.Item.Slots[0],
	//	//weapon.Item.Slots[1],
	//	//weapon.Item.Slots[2],
	//	weapon.Item.Slots[3],
	//	//weapon.Item.Slots[4],
	//}

	slotNameMap := make(map[string]*candidate_tree.ItemSlot)

	for _, slot := range weapon.Item.Slots {
		slotNameMap[slot.Name] = slot
		//build := processSlots([]*candidate_tree.ItemSlot{slot}, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil)
		//build.WeaponTree = *weapon
		//evaledSlot, err := build.ToEvaluatedWeapon()
		//if err != nil {
		//	log.Error().Err(err).Msgf("Failed to convert build to evaluated weapon for %s", weapon.Item.Name)
		//}
		//log.Info().Msgf("Slot %s, Build: %v", slot.Name, evaledSlot)
	}

	weapon.UpdateAllowedItemSlots()
	weapon.UpdateAllowedItems()

	stock := slotNameMap["Stock"]
	stockBuild := processSlots(weapon, []*candidate_tree.ItemSlot{stock}, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil, map[string]*Build{})

	pg := slotNameMap["Pistol Grip"]
	pgBuild := processSlots(weapon, []*candidate_tree.ItemSlot{pg}, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil, map[string]*Build{})

	conflict := false
	for _, item := range stockBuild.OptimalItems {
		x := weapon.GetAllowedItem(item.ID)
		for _, conflictingItem := range x.ConflictingItems {
			for _, pgItem := range pgBuild.OptimalItems {
				if conflictingItem.ID == pgItem.ID {
					conflict = true
				}
			}
		}
	}

	if conflict {
		if stockBuild.RecoilSum < pgBuild.RecoilSum {
			for _, item := range stockBuild.OptimalItems {
				ai := weapon.GetAllowedItem(item.ID)
				for _, conflictingItem := range ai.ConflictingItems {
					excludedItems[conflictingItem.ID] = true
				}
			}
		} else {
			for _, item := range pgBuild.OptimalItems {
				ai := weapon.GetAllowedItem(item.ID)
				for _, conflictingItem := range ai.ConflictingItems {
					excludedItems[conflictingItem.ID] = true
				}
			}
		}
	}

	memo := map[string]*Build{}
	//build := processSlots(slots, []OptimalItem{}, focusedStat, 0, 0, excludedItems, []string{}, memo, itemHits)
	build := processSlots(weapon, weapon.Item.Slots, []OptimalItem{}, focusedStat, -35, 0, excludedItems, nil, memo)

	build.WeaponTree = *weapon

	return build
}

// conflictsWith checks if two items conflict based on the conflict map.
func conflictsWith(item *candidate_tree.Item, chosen OptimalItem) bool {
	for _, conflict := range item.ConflictingItems {
		if conflict.ID == chosen.ID {
			return true
		}
	}
	return false
}

func doesImproveStats(candidate *Build, best *Build, focusedStat string) bool {
	if focusedStat == "recoil" {
		if candidate.RecoilSum < best.RecoilSum {
			return true
		} else if candidate.RecoilSum == best.RecoilSum {
			return candidate.ErgonomicsSum > best.ErgonomicsSum
		}
	} else if focusedStat == "ergonomics" {
		if candidate.ErgonomicsSum > best.ErgonomicsSum {
			return true
		} else if candidate.ErgonomicsSum == best.ErgonomicsSum {
			return candidate.RecoilSum < best.RecoilSum
		}
	}

	return false
}

func getMemoKey(slotID string, excludedItems map[string]bool) string {
	items := make([]string, 0)
	for id, excluded := range excludedItems {
		if excluded {
			items = append(items, id)
		}
	}

	sort.Strings(items)
	return fmt.Sprintf("%s:%v", slotID, strings.Join(items, "-"))
}

// Refactored processSlots function
func processSlots(
	root *candidate_tree.CandidateTree,
	slotsToProcess []*candidate_tree.ItemSlot,
	chosenItems []OptimalItem,
	focusedStat string,
	recoilStatSum int,
	ergoStatSum int,
	excludedItems map[string]bool,
	visitedSlots map[string]bool,
	memo map[string]*Build,
) *Build {
	clonedSlots := append([]*candidate_tree.ItemSlot{}, slotsToProcess...)
	//clonedSlots = filterSlots(clonedSlots, []string{"Tactical", "Ubgl", "Mount", "Scope"}) // No filtering

	// Base case: No more slots to process
	if len(clonedSlots) == 0 {
		exclusions := make([]string, 0)
		for excludedID, isExcluded := range excludedItems {
			if isExcluded {
				exclusions = append(exclusions, excludedID)
			}
		}

		return &Build{
			OptimalItems:   append([]OptimalItem{}, chosenItems...), // Make a copy
			RecoilSum:      recoilStatSum,
			ErgonomicsSum:  ergoStatSum,
			EvaluationType: focusedStat,
			ExcludedItems:  exclusions,
		}
	}

	// Take the first slot to process
	currentSlot := clonedSlots[0]
	remainingSlots := clonedSlots[1:]

	// Cycle Detection
	if visitedSlots[currentSlot.ID] {
		// Slot has already been processed in this path, skip to prevent cycle
		return processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, excludedItems, visitedSlots, memo)
	}

	// Mark the current slot as visited
	if visitedSlots == nil {
		visitedSlots = make(map[string]bool)
	}
	visitedSlots[currentSlot.ID] = true
	defer func() {
		delete(visitedSlots, currentSlot.ID)
	}()

	var best *Build = nil

	// Iterate through all allowed items for the current slot
	for _, item := range currentSlot.AllowedItems {
		key := getMemoKey(item.ID, excludedItems)
		if cached, ok := memo[key]; ok {
			best = cached
			log.Info().Msgf("cache hit for %s", item.Name)
			break
		}

		// Skip if the item is excluded
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

		// Assign the item
		newChosen := append(chosenItems, OptimalItem{
			Name:   item.Name,
			ID:     item.ID,
			SlotID: currentSlot.ID,
		})

		//if item.ConflictingItems == nil || len(item.ConflictingItems) == 0 {
		//	for _, chosen := range chosenItems {
		//		// Iterate through chosenItems instead of currentSlot.AllowedItems
		//
		//	}
		//}

		// Update sums
		newRecoil := recoilStatSum + item.RecoilModifier
		newErgo := ergoStatSum + item.ErgonomicsModifier

		// Prepare new slots to process, **prepend** any sub-slots from the current item
		newSlotsToProcess := append([]*candidate_tree.ItemSlot{}, item.Slots...)
		newSlotsToProcess = append(newSlotsToProcess, remainingSlots...)

		// Clone and update excludedItems for conflicts
		newExcluded := helpers.CloneMap(excludedItems)
		for _, c := range item.ConflictingItems {
			newExcluded[c.ID] = true
		}

		// Recursive call with updated visitedSlots
		candidate := processSlots(root, newSlotsToProcess, newChosen, focusedStat, newRecoil, newErgo, newExcluded, visitedSlots, memo)

		// Compare and retain the best build
		if candidate != nil {
			if best == nil || doesImproveStats(candidate, best, focusedStat) {
				best = candidate
			}
		}
	}

	// **Attempt to Skip the Current Slot**
	// After trying all items, also attempt to skip assigning any item to the current slot
	candidateSkip := processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, helpers.CloneMap(excludedItems), visitedSlots, memo)

	if candidateSkip != nil {
		if best == nil || doesImproveStats(candidateSkip, best, focusedStat) {
			best = candidateSkip
		}
	}

	if best != nil {
		key := getMemoKey(currentSlot.ID, excludedItems)
		memo[key] = best
	}

	// Return the best build found
	return best
}

type ChosenTree struct {
	ID                 string
	Name               string
	HasConflicts       bool
	Slots              map[string]*ChosenTree
	RecoilModifier     int
	ErgonomicsModifier int
	RecoilSum          int
	ErgonomicsSum      int
	SlotID             string
	OptimalItems       []OptimalItem
}

func getNonConflictingTrees(build *Build, itemMap map[string]*candidate_tree.Item) map[string]*ChosenTree {
	chosenMap := map[string]*ChosenTree{}

	for _, item := range build.OptimalItems {
		hasConflicts := false
		if itemMap[item.ID].ConflictingItems != nil && len(itemMap[item.ID].ConflictingItems) > 0 {
			hasConflicts = true
		}

		chosenMap[item.ID] = &ChosenTree{
			ID:                 item.ID,
			Name:               item.Name,
			Slots:              map[string]*ChosenTree{},
			HasConflicts:       hasConflicts,
			RecoilModifier:     itemMap[item.ID].RecoilModifier,
			ErgonomicsModifier: itemMap[item.ID].ErgonomicsModifier,
			SlotID:             item.SlotID,
			OptimalItems:       []OptimalItem{item},
		}

		for _, slot := range itemMap[item.ID].Slots {
			for _, sub := range build.OptimalItems {
				if slot.ID == sub.SlotID {
					chosenMap[item.ID].Slots[slot.ID] = &ChosenTree{
						ID:    sub.ID,
						Name:  sub.Name,
						Slots: map[string]*ChosenTree{},
					}
					chosenMap[item.ID].OptimalItems = append(chosenMap[item.ID].OptimalItems, sub)
				}
			}
		}
	}

	filtered := map[string]*ChosenTree{}
	for _, item := range build.OptimalItems {
		if !chosenMap[item.ID].HasConflicts && len(chosenMap[item.ID].Slots) > 0 {
			filtered[item.ID] = chosenMap[item.ID]
		}
	}

	for _, item := range filtered {
		recoil, ergo := calculateSums(item, 0, 0)
		item.RecoilSum = recoil
		item.ErgonomicsSum = ergo
	}

	return filtered
}

func calculateSums(item *ChosenTree, recoilSum int, ergoSum int) (int, int) {
	slotRecoil := 0
	slotErgo := 0
	for _, slot := range item.Slots {
		slotRecoil, slotErgo = calculateSums(slot, recoilSum, ergoSum)
	}

	recoilSum += item.RecoilModifier + slotRecoil
	ergoSum += item.ErgonomicsModifier + slotErgo

	return recoilSum, ergoSum

}
