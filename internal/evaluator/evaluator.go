package evaluator

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"

	"github.com/rs/zerolog/log"
)

// getTraderLevel returns the level for a specific trader
func getTraderLevel(levels []models.TraderLevel, traderName string) int {
	for _, level := range levels {
		if level.Name == traderName {
			return level.Level
		}
	}
	return 0
}

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
	RecoilSum          int                       `json:"recoil_sum"`
	ErgonomicsSum      int                       `json:"ergonomics_sum"`
}

type SlotEvaluation struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Item    *ItemEvaluation `json:"item"`
	IsEmpty bool            `json:"empty"`
}

func (s *SlotEvaluation) ToSlotEvaluationResult(evaluationType string) models.SlotEvaluationResult {
	result := models.SlotEvaluationResult{
		ID:      s.ID,
		Name:    s.Name,
		Item:    models.ItemEvaluationResult{},
		IsEmpty: s.IsEmpty,
	}

	if s.IsEmpty {
		return result
	}

	result.Item = models.ItemEvaluationResult{
		ID:                 s.Item.ID,
		Name:               s.Item.Name,
		RecoilModifier:     s.Item.RecoilModifier,
		ErgonomicsModifier: s.Item.ErgonomicsModifier,
		RecoilSum:          s.Item.RecoilSum,
		ErgonomicsSum:      s.Item.ErgonomicsSum,
		Slots:              make([]models.SlotEvaluationResult, 0),
		IsSubtree:          true,
		EvaluationType:     evaluationType,
	}

	for _, slot := range s.Item.Slots {
		s := slot.ToSlotEvaluationResult(evaluationType)
		result.Item.Slots = append(result.Item.Slots, s)
	}

	return result
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

// TODO - do something about this
// ToItemEvaluationResult temporary - just makes sure we can seialise the same response/save builds as per the previous evaluator iteration
func (weapon *EvaluatedWeapon) ToItemEvaluationResult() models.ItemEvaluationResult {
	w := models.ItemEvaluationResult{
		ID:             weapon.ID,
		Name:           weapon.Name,
		IsSubtree:      false,
		EvaluationType: weapon.EvaluationType,
		RecoilSum:      weapon.RecoilSum,
		ErgonomicsSum:  weapon.ErgonomicsSum,
		Slots:          make([]models.SlotEvaluationResult, 0, len(weapon.Slots)),
	}

	w.RecoilSum = weapon.RecoilSum
	w.ErgonomicsSum = weapon.ErgonomicsSum

	for _, slot := range weapon.Slots {
		s := slot.ToSlotEvaluationResult(weapon.EvaluationType)
		w.Slots = append(w.Slots, s)
	}

	return w
}

type OptimalItem struct {
	Name         string
	ID           string
	SlotID       string
	HasConflicts bool
}

// Build represents a complete weapon configuration.
type Build struct {
	WeaponTree     *candidate_tree.CandidateTree
	OptimalItems   []OptimalItem
	RecoilSum      int `json:"recoil_sum"`
	ErgonomicsSum  int `json:"ergonomics_sum"`
	EvaluationType string
	ExcludedItems  []string
	HasConflicts   bool
	CacheHits      int64 `json:"cache_hits"`
	CacheMisses    int64 `json:"cache_misses"`
	ItemsEvaluated int64 `json:"items_evaluated"`
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
			newItems := make([]OptimalItem, 0)
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
	excludedItems map[string]bool, cache Cache) *Build {

	log.Debug().Msgf("Finding best build for %s", weapon.Item.Name)

	slotNameMap := make(map[string]*candidate_tree.ItemSlot)

	for _, slot := range weapon.Item.Slots {
		slotNameMap[slot.Name] = slot
	}

	weapon.UpdateAllowedItemSlots()
	weapon.UpdateAllowedItems()

	//stock := slotNameMap["Stock"]
	//stockBuild := processSlots(weapon, []*candidate_tree.ItemSlot{stock}, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil, map[string]*Build{})
	//
	//pg := slotNameMap["Pistol Grip"]
	//pgBuild := processSlots(weapon, []*candidate_tree.ItemSlot{pg}, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil, map[string]*Build{})
	//
	//conflict := false
	//for _, item := range stockBuild.OptimalItems {
	//	x := weapon.GetAllowedItem(item.ID)
	//	for _, conflictingItem := range x.ConflictingItems {
	//		for _, pgItem := range pgBuild.OptimalItems {
	//			if conflictingItem.ID == pgItem.ID {
	//				conflict = true
	//			}
	//		}
	//	}
	//}
	//
	//if conflict {
	//	if stockBuild.RecoilSum < pgBuild.RecoilSum {
	//		for _, item := range stockBuild.OptimalItems {
	//			ai := weapon.GetAllowedItem(item.ID)
	//			for _, conflictingItem := range ai.ConflictingItems {
	//				excludedItems[conflictingItem.ID] = true
	//			}
	//		}
	//	} else {
	//		for _, item := range pgBuild.OptimalItems {
	//			ai := weapon.GetAllowedItem(item.ID)
	//			for _, conflictingItem := range ai.ConflictingItems {
	//				excludedItems[conflictingItem.ID] = true
	//			}
	//		}
	//	}
	//}

	slotDescendantItemIDs := precomputeSlotDescendantItemIDs(weapon)

	var cacheHits, cacheMisses, itemsEvaluated int64
	build := processSlots(weapon, weapon.Item.Slots, []OptimalItem{}, focusedStat, 0, 0, excludedItems, nil, slotDescendantItemIDs, &cacheHits, &cacheMisses, &itemsEvaluated, cache)

	build.WeaponTree = weapon
	build.CacheHits = cacheHits
	build.CacheMisses = cacheMisses
	build.ItemsEvaluated = itemsEvaluated

	if cacheHits+cacheMisses > 0 {
		hitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
		log.Debug().Msgf("Conflict-free cache: %d hits, %d misses (%.1f%% hit rate)", cacheHits, cacheMisses, hitRate)
	}

	return build
}

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

// computeRecoilLowerBound returns the minimal possible final recoil sum achievable by
// filling the given slots from the current recoil sum, using each slot's MinRecoil potential.
func computeRecoilLowerBound(currentRecoil int, slots []*candidate_tree.ItemSlot) int {
	bound := currentRecoil
	for _, s := range slots {
		if s == nil {
			continue
		}
		bound += s.PotentialValues.MinRecoil
	}
	return bound
}

// computeErgoUpperBound returns the maximal possible final ergonomics sum achievable by
// filling the given slots from the current ergonomics sum, using each slot's MaxErgonomics potential.
func computeErgoUpperBound(currentErgo int, slots []*candidate_tree.ItemSlot) int {
	bound := currentErgo
	for _, s := range slots {
		if s == nil {
			continue
		}
		bound += s.PotentialValues.MaxErgonomics
	}
	return bound
}

// precomputeSlotDescendantItemIDs returns, for every slot in the candidate tree, the set of
// descendant allowed item IDs reachable from that slot. This is used to reduce the memo key
// to only exclusions that matter for the current subproblem (current slot + remaining slots).
func precomputeSlotDescendantItemIDs(root *candidate_tree.CandidateTree) map[string]map[string]bool {
	descendantMap := make(map[string]map[string]bool)

	// Ensure allowed item slots are up to date
	root.UpdateAllowedItemSlots()

	// Get every slot reachable from the root item (includes immediate and deeper slots)
	allSlots := root.Item.GetDescendantSlots()
	for _, slot := range allSlots {
		if slot == nil {
			continue
		}
		if _, ok := descendantMap[slot.ID]; !ok {
			descendantMap[slot.ID] = make(map[string]bool)
		}
		allowed := slot.GetDescendantAllowedItems()
		for _, it := range allowed {
			if it == nil {
				continue
			}
			descendantMap[slot.ID][it.ID] = true
		}
	}

	return descendantMap
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
	slotDescendantItemIDs map[string]map[string]bool,
	cacheHits *int64,
	cacheMisses *int64,
	itemsEvaluated *int64,
	cache Cache,
) *Build {
	clonedSlots := append([]*candidate_tree.ItemSlot{}, slotsToProcess...)

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

	currentSlot := clonedSlots[0]
	remainingSlots := clonedSlots[1:]

	if visitedSlots[currentSlot.ID] {
		return processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, excludedItems, visitedSlots, slotDescendantItemIDs, cacheHits, cacheMisses, itemsEvaluated, cache)
	}

	if visitedSlots == nil {
		visitedSlots = make(map[string]bool)
	}
	visitedSlots[currentSlot.ID] = true
	defer func() {
		delete(visitedSlots, currentSlot.ID)
	}()

	var best *Build = nil

	for _, item := range currentSlot.AllowedItems {
		// Track items evaluated
		atomic.AddInt64(itemsEvaluated, 1)

		if focusedStat == "recoil" {
			if item.PotentialValues.MinRecoil >= 0 {
				continue
			}
		} else if focusedStat == "ergonomics" {
			if item.PotentialValues.MaxErgonomics <= 0 {
				continue
			}
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

		// Check if this item is conflict-free (can be cached safely)
		isConflictFree := len(item.ConflictingItems) == 0

		// Try conflict-free cache lookup for pruning
		if isConflictFree && cache != nil {
			cachedEntry, err := cache.Get(context.Background(), item.ID, focusedStat, root.Constraints)
			if err == nil && cachedEntry != nil {
				atomic.AddInt64(cacheHits, 1)

				// Use cached stats for pruning - if we know the result won't be better, skip evaluation
				if best != nil {
					potentialRecoil := (recoilStatSum + item.RecoilModifier) + cachedEntry.RecoilSum
					potentialErgo := (ergoStatSum + item.ErgonomicsModifier) + cachedEntry.ErgonomicsSum

					if focusedStat == "recoil" && potentialRecoil >= best.RecoilSum {
						// Cached result won't be better, skip evaluation
						continue
					} else if focusedStat == "ergonomics" && potentialErgo <= best.ErgonomicsSum {
						// Cached result won't be better, skip evaluation
						continue
					}
				}
			} else {
				atomic.AddInt64(cacheMisses, 1)
			}
		}

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

		if best != nil {
			if focusedStat == "recoil" {
				lowerBound := computeRecoilLowerBound(newRecoil, newSlotsToProcess)
				if lowerBound > best.RecoilSum {
					continue
				}
			} else if focusedStat == "ergonomics" {
				upperBound := computeErgoUpperBound(newErgo, newSlotsToProcess)
				if upperBound < best.ErgonomicsSum {
					continue
				}
			}
		}

		candidate := processSlots(root, newSlotsToProcess, newChosen, focusedStat, newRecoil, newErgo, newExcluded, visitedSlots, slotDescendantItemIDs, cacheHits, cacheMisses, itemsEvaluated, cache)

		// Cache conflict-free results
		if isConflictFree && candidate != nil && cache != nil {
			// For conflict-free items, we can cache the result
			// since there are no conflicts to worry about
			_ = cache.Set(context.Background(), item.ID, focusedStat, root.Constraints, &CacheEntry{
				RecoilSum:     candidate.RecoilSum,
				ErgonomicsSum: candidate.ErgonomicsSum,
			})
		}

		if candidate != nil {
			if best == nil {
				// it could be better than possibly nothing, so store it as the best for now.
				best = candidate
			} else if doesImproveStats(candidate, best, focusedStat) {
				// it's better than the best we've seen so far
				best = candidate

				// do not break; later items may unlock better global builds due to conflicts
			}
		}
	}

	// this evaluates the outcome of leaving this slot empty. Basically, there's the possibility that some item which
	// opens up a better build can be slotted in elsewhere which would conflict with any build created using any item
	// in this slot.
	// Option to leave this slot empty; apply pruning before exploring
	if best != nil {
		switch focusedStat {
		case "recoil":
			lowerBound := computeRecoilLowerBound(recoilStatSum, remainingSlots)
			if lowerBound > best.RecoilSum {
				// cannot beat best even if remaining slots are ideal
				return best
			}
		case "ergonomics":
			upperBound := computeErgoUpperBound(ergoStatSum, remainingSlots)
			if upperBound < best.ErgonomicsSum {
				return best
			}
		}
	}
	candidateSkip := processSlots(root, remainingSlots, chosenItems, focusedStat, recoilStatSum, ergoStatSum, helpers.CloneMap(excludedItems), visitedSlots, slotDescendantItemIDs, cacheHits, cacheMisses, itemsEvaluated, cache)

	if candidateSkip != nil {
		if best == nil || doesImproveStats(candidateSkip, best, focusedStat) {
			best = candidateSkip
		}
	}

	return best
}
