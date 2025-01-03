package evaluator

import (
	"tarkov-build-optimiser/internal/weapon_tree"
)

type ItemEvaluation struct {
	ID   string
	Name string
}

type SlotEvaluation struct {
	ID    string
	Name  string
	Item  ItemEvaluation
	Empty bool
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
}

//func (b *Build) ToEvaluationResult() []models.ItemEvaluationResult {
//	var result []models.ItemEvaluationResult
//	for _, item := range b.OptimalItems {
//
//	}
//	return result
//}

// findBestBuild recursively traverses the build tree to find the optimal build.
// It returns the best Build found in the current recursion path.
func findBestBuild(slotsToProcess []*weapon_tree.ItemSlot, chosenItems []OptimalItem, focusedStat string, focusedStatSum int,
	excludedItems map[string]bool) *Build {

	// Base Case: No more slots to process
	if len(slotsToProcess) == 0 {
		return &Build{
			OptimalItems:   append([]OptimalItem{}, chosenItems...), // Make a copy
			FocusedStatSum: focusedStatSum,
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
		newExcluded := copyExclusions(excludedItems)
		for _, conflictItem := range item.ConflictingItems {
			newExcluded[conflictItem] = true
		}

		// Append subslots opened by this item to the remainingSlots
		newSlotsToProcess := append([]*weapon_tree.ItemSlot{}, remainingSlots...) // Copy to avoid mutation
		if len(item.Slots) > 0 {
			newSlotsToProcess = append(newSlotsToProcess, item.Slots...)
		}

		// Recurse with the updated build state and slots
		candidate := findBestBuild(newSlotsToProcess, newChosen, focusedStat, newRecoil, newExcluded)

		// Update the best build if necessary
		if candidate != nil {
			if best == nil || candidate.FocusedStatSum > best.FocusedStatSum {
				best = candidate
			}
		}
	}

	// Proceed without choosing any item for this slot
	candidate := findBestBuild(remainingSlots, chosenItems, focusedStat, focusedStatSum, excludedItems)
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

// copyExclusions creates a deep copy of the excluded items map.
func copyExclusions(original map[string]bool) map[string]bool {
	newMap := make(map[string]bool)
	for k, v := range original {
		newMap[k] = v
	}
	return newMap
}
