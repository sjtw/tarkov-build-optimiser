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

	log.Info().Msgf("Finding best build for %s", weapon.Item.Name)

	slotNameMap := make(map[string]*candidate_tree.ItemSlot)

	for _, slot := range weapon.Item.Slots {
		slotNameMap[slot.Name] = slot
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

	// select the next slot in the array. remaining slots will be passed onto the next recursive call.
	currentSlot := clonedSlots[0]
	remainingSlots := clonedSlots[1:]

	// visitedSlots just keeps track of how often we're hitting certain slots. Can be useful for debugging, but it's
	// a good safe guard against infinite loops. in theory CandidateTree should be handling this.
	if visitedSlots[currentSlot.ID] {
		return processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, excludedItems, visitedSlots, memo)
	}

	if visitedSlots == nil {
		visitedSlots = make(map[string]bool)
	}
	visitedSlots[currentSlot.ID] = true
	defer func() {
		delete(visitedSlots, currentSlot.ID)
	}()

	var best *Build = nil

	// pretty straightforward:
	// - loop through all the items we can put in this slot see what the outcome is
	// - pick the best
	// - alloweditems are sorted by potential value, so if we can slot something together earlier, nothing later on in
	//   the AllowedItems slice is going to improve things. edit: actually they could, needs improvement.
	for _, item := range currentSlot.AllowedItems {
		// if we've already evaluated this exact slot with the exact same exclusion list, we can just return the result
		// in practice this has never happened as far as I can tell. It should being way more value when persisted and
		// budget constraints are added.
		key := getMemoKey(item.ID, excludedItems)
		if cached, ok := memo[key]; ok {
			best = cached
			break
		}

		// if this item is explicitly excluded, we can skip it
		// any conflicts with items so far should also be in here.
		if excludedItems[item.ID] {
			continue
		}

		// i've a feeling this just does the same as the above now...
		// TODO - confirm & remove if so
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

		// from here until the end of the loop we're going to see what happens if we slot this item into this slot.
		newChosen := append(chosenItems, OptimalItem{
			Name:   item.Name,
			ID:     item.ID,
			SlotID: currentSlot.ID,
		})

		newRecoil := recoilStatSum + item.RecoilModifier
		newErgo := ergoStatSum + item.ErgonomicsModifier

		newSlotsToProcess := append([]*candidate_tree.ItemSlot{}, item.Slots...)
		newSlotsToProcess = append(newSlotsToProcess, remainingSlots...)

		newExcluded := helpers.CloneMap(excludedItems)
		for _, c := range item.ConflictingItems {
			newExcluded[c.ID] = true
		}

		candidate := processSlots(root, newSlotsToProcess, newChosen, focusedStat, newRecoil, newErgo, newExcluded, visitedSlots, memo)

		if candidate != nil {
			if best == nil {
				// it could be better than possibly nothing, so store it as the best for now.
				best = candidate
			} else if doesImproveStats(candidate, best, focusedStat) {
				// it's better than the best we've seen so far
				best = candidate

				// we're assuming that because AllowedItems is sorted by its potential stat sum we can break here.
				// this isn't 100% accurate - more of an estimation - we still need to check if this build is
				// ACTUALLY better than the potential of others or likelyhood of them being so.
				break
			}
		}
	}

	// this evaluates the outcome of leaving this slot empty. Basically, there's the possibility that some item which
	// opens up a better build can be slotted in elsewhere which would conflict with any build created using any item
	// in this slot.
	candidateSkip := processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, helpers.CloneMap(excludedItems), visitedSlots, memo)

	if candidateSkip != nil {
		if best == nil || doesImproveStats(candidateSkip, best, focusedStat) {
			best = candidateSkip
		}
	}

	// memoize the best build for this slot. I'm pretty certain this is actually never useful at the moment
	// as we don't persist anything, and technically shouldn't be reevaluating the same path twice anyway...
	//
	// TDOO: persist these in db - including the candidate tree constraints we're operating on. This might at least
	// help when re-evaluating everything after a new patch.
	//
	// the only constraints for this build are the current exclusion list and the current slot we're evaluating.
	// the exclusion list contains:
	// - anything explicitly excluded by the caller
	// - any items which conflict with an item selected by the best build
	if best != nil {
		key := getMemoKey(currentSlot.ID, excludedItems)
		memo[key] = best
	}

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

// just an idea - basically a complete build might never be used again, but a partial one may be suitable for reuse if
// it never conflicts with anything else, or if has a low number of conflicts. AKs in particular come to mind.
//func getNonConflictingTrees(build *Build, itemMap map[string]*candidate_tree.Item) map[string]*ChosenTree {
//	chosenMap := map[string]*ChosenTree{}
//
//	for _, item := range build.OptimalItems {
//		hasConflicts := false
//		if itemMap[item.ID].ConflictingItems != nil && len(itemMap[item.ID].ConflictingItems) > 0 {
//			hasConflicts = true
//		}
//
//		chosenMap[item.ID] = &ChosenTree{
//			ID:                 item.ID,
//			Name:               item.Name,
//			Slots:              map[string]*ChosenTree{},
//			HasConflicts:       hasConflicts,
//			RecoilModifier:     itemMap[item.ID].RecoilModifier,
//			ErgonomicsModifier: itemMap[item.ID].ErgonomicsModifier,
//			SlotID:             item.SlotID,
//			OptimalItems:       []OptimalItem{item},
//		}
//
//		for _, slot := range itemMap[item.ID].Slots {
//			for _, sub := range build.OptimalItems {
//				if slot.ID == sub.SlotID {
//					chosenMap[item.ID].Slots[slot.ID] = &ChosenTree{
//						ID:    sub.ID,
//						Name:  sub.Name,
//						Slots: map[string]*ChosenTree{},
//					}
//					chosenMap[item.ID].OptimalItems = append(chosenMap[item.ID].OptimalItems, sub)
//				}
//			}
//		}
//	}
//
//	filtered := map[string]*ChosenTree{}
//	for _, item := range build.OptimalItems {
//		if !chosenMap[item.ID].HasConflicts && len(chosenMap[item.ID].Slots) > 0 {
//			filtered[item.ID] = chosenMap[item.ID]
//		}
//	}
//
//	for _, item := range filtered {
//		recoil, ergo := calculateSums(item, 0, 0)
//		item.RecoilSum = recoil
//		item.ErgonomicsSum = ergo
//	}
//
//	return filtered
//}

//func calculateSums(item *ChosenTree, recoilSum int, ergoSum int) (int, int) {
//	slotRecoil := 0
//	slotErgo := 0
//	for _, slot := range item.Slots {
//		slotRecoil, slotErgo = calculateSums(slot, recoilSum, ergoSum)
//	}
//
//	recoilSum += item.RecoilModifier + slotRecoil
//	ergoSum += item.ErgonomicsModifier + slotErgo
//
//	return recoilSum, ergoSum
//
//}
