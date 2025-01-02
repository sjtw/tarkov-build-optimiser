package evaluator

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type Data struct {
}

type EvaluationDataProvider interface {
	GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error)
	GetTraderOffer(itemID string) ([]models.TraderOffer, error)
	SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints)
}

type Task struct {
	Constraints    models.EvaluationConstraints
	WeaponTree     WeaponTree
	EvaluationType string
}

type Evaluator struct {
	dataService EvaluationDataProvider
}

func CreateEvaluator(dataService EvaluationDataProvider) *Evaluator {
	return &Evaluator{
		dataService: dataService,
	}
}

func (e *Evaluator) filterCandidateItemsByTraderAvailability(candidates map[string]bool, traderLevels []models.TraderLevel) (map[string]bool, error) {
	filteredCandidates := map[string]bool{}

	for itemId := range candidates {
		offers, err := e.dataService.GetTraderOffer(itemId)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get trader offer for item [%s]", itemId)
			return nil, err
		}

		valid := validateTraderLevels(offers, traderLevels)
		if valid {
			filteredCandidates[itemId] = true
		}
	}

	return filteredCandidates, nil
}

func (e *Evaluator) EvaluateTask(task Task) (models.ItemEvaluationResult, error) {
	filteredCandidates, err := e.filterCandidateItemsByTraderAvailability(task.WeaponTree.CandidateItems, task.Constraints.TraderLevels)
	if err != nil {
		log.Error().Err(err).Msg("Failed to filter candidate items by trader availability")
		return models.ItemEvaluationResult{}, err
	}

	candidateSets := make([][]string, 0)

	// perform trader level filtering

	if len(task.WeaponTree.AllowedItemConflicts) > 0 {
		// generate all valid variations of candidates, including the weapon ID
		candidateSets = GenerateNonConflictingCandidateSets(filteredCandidates, task.WeaponTree.AllowedItemConflicts)
	} else {
		// the weapon has no possible conflicts, so we can use all candidate items from the candidate tree
		set := make([]string, len(filteredCandidates))
		for id := range task.WeaponTree.CandidateItems {
			set = append(set, id)
		}
		candidateSets = append(candidateSets, set)
	}

	optimum := models.ItemEvaluationResult{}
	optimumSum := 0
	for index, candidateItems := range candidateSets {
		log.Info().Msgf("index %d", index)
		result, err := e.evaluate(task.WeaponTree.Item, task.EvaluationType, task.Constraints, candidateItems)
		if err != nil {
			return models.ItemEvaluationResult{}, err
		}

		if task.EvaluationType == "recoil" {
			if result.RecoilSum < optimumSum {
				optimumSum = result.RecoilSum
				optimum = *result
			}
		} else {
			if result.ErgonomicsSum > optimumSum {
				optimumSum = result.ErgonomicsSum
				optimum = *result
			}
		}
	}

	return optimum, nil
}

func (e *Evaluator) evaluate(item *Item, evaluationType string, constraints models.EvaluationConstraints, candidates []string) (*models.ItemEvaluationResult, error) {
	outItem := &models.ItemEvaluationResult{
		ID:                 item.ID,
		Name:               item.Name,
		EvaluationType:     evaluationType,
		RecoilModifier:     item.RecoilModifier,
		ErgonomicsModifier: item.ErgonomicsModifier,
		Slots:              make([]models.SlotEvaluationResult, len(item.Slots)),
		ErgonomicsSum:      item.ErgonomicsModifier,
		RecoilSum:          item.RecoilModifier,
	}

	if item.Type == "weapon" {
		outItem.IsSubtree = false
	} else {
		outItem.IsSubtree = true
	}

	if item.Slots == nil {
		return outItem, nil
	}

	// TODO: pre evaluated items also need to be keyed by whitelisted candidate set
	//		else we can accidentally re-include conflicting items

	//preEvaluatedItem, err := e.dataService.GetSubtree(outItem.ID, evaluationType, constraints)
	//if err != nil {
	//	log.Error().Err(err).Msgf("Failed to get subtree for evaluation. item: %s", preEvaluatedItem.ID)
	//	return nil, err
	//}
	//
	//if preEvaluatedItem != nil {
	//	log.Debug().Msgf("Optimal [%s] evaluation for item [%s] already evaluated. returning.", outItem.EvaluationType, item.ID)
	//	return preEvaluatedItem, nil
	//}

	for i := 0; i < len(item.Slots); i++ {
		log.Debug().Msgf("Evaluating slot %d for item [%s]", i, item.ID)

		outSlot := &models.SlotEvaluationResult{
			ID:   item.Slots[i].ID,
			Name: item.Slots[i].Name,
			Item: nil,
		}

		if isIgnoredSlotName(item.Slots[i].Name, constraints.IgnoredSlotNames) {
			outItem.Slots[i] = *outSlot
			continue
		}

		slotErgo := 0
		slotRecoil := 0
		if outSlot.Name == "Gas Block" {
			log.Debug().Msgf("Gas Tube slot")
		}

		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			log.Debug().Msgf("Evaluating slot %d for item [%s]", j, item.ID)
			ai := item.Slots[i].AllowedItems[j]

			isCandidate := false
			for _, candidateID := range candidates {
				if candidateID == ai.ID {
					isCandidate = true
				}
			}

			if !isCandidate {
				log.Info().Msgf("Candidate item [%s] is not a whitelisted candidate for this evaluation", item.ID)
				continue
			} else {
				log.Info().Msgf("Candidate item [%s] is a valid candidate for this evaluation", item.ID)
			}

			highestItem, err := e.evaluate(ai, evaluationType, constraints, candidates)
			if err != nil {
				log.Debug().Msg("Failed to evaluate highest item")
				return nil, err
			}

			recoilSum := highestItem.RecoilSum
			ergoSum := highestItem.ErgonomicsSum

			log.Debug().Msgf("Evaluation type %s", evaluationType)
			if evaluationType == "ergonomics" {
				if ergoSum > slotErgo {
					log.Debug().Msgf("item %v IS new highest sum", highestItem)
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				} else if outSlot.Item == nil && recoilSum < slotRecoil {
					log.Debug().Msgf("Item %v does not improve ergonomics, but improves recoil over empty slot.", highestItem)
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				} else {
					log.Debug().Msgf("item %v does not improve stats", highestItem)
				}
			} else if evaluationType == "recoil" {
				if recoilSum < slotRecoil {
					log.Debug().Msgf("item %v IS new highest sum", highestItem)
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				} else if outSlot.Item == nil && ergoSum > slotErgo {
					log.Debug().Msgf("Item %v does not improve recoil, but improves ergo over empty slot.", highestItem)
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				} else {
					log.Debug().Msgf("item %v does not improve stats", highestItem)
				}
			} else {
				msg := fmt.Sprintf("Invalid evaluation type [%s]", evaluationType)
				log.Error().Msg(msg)
				return nil, errors.New(msg)
			}
		}

		outItem.Slots[i] = *outSlot
		outItem.RecoilSum += slotRecoil
		outItem.ErgonomicsSum += slotErgo
	}

	if !outItem.IsSubtree {
		e.dataService.SaveBuild(outItem, constraints)
	}

	log.Debug().Msgf("Output item %v", outItem)

	return outItem, nil
}

func isIgnoredSlotName(slotName string, ignoredSlots map[string]bool) bool {
	_, exists := ignoredSlots[slotName]
	return exists
}

func conflictsWithSet(conflictMap map[string]map[string]bool, candidate string, currentSet []string) bool {
	for _, member := range currentSet {
		if conflictMap[candidate] != nil && conflictMap[candidate][member] {
			return true
		}
	}
	return false
}

// GenerateNonConflictingCandidateSets - generates all maximal, non-conflicting sets of candidate item IDs given
// the candidate list and conflict maps.
func GenerateNonConflictingCandidateSets(candidates map[string]bool, conflicts map[string]map[string]bool) [][]string {
	// Some items do not have symmetrical conflicts (for example pistol grips with integrated buttstocks conflict)
	// with most stocks, however there is no conflict in the other direction. By ensuring all conflicts are symmetrical
	// up-front we never need to be concerned with the order items are checked/added to a build.
	symmetricConflicts := make(map[string]map[string]bool)
	for candidateId, conflictSet := range conflicts {
		if _, exists := symmetricConflicts[candidateId]; !exists {
			symmetricConflicts[candidateId] = make(map[string]bool)
		}

		for conflictId := range conflictSet {
			if _, exists := symmetricConflicts[conflictId]; !exists {
				symmetricConflicts[conflictId] = make(map[string]bool)
			}
			symmetricConflicts[candidateId][conflictId] = true
			symmetricConflicts[conflictId][candidateId] = true
		}
	}

	result := [][]string{}

	conflictFree := []string{}
	conflictingCandidates := []string{}
	for candidate := range candidates {
		if len(symmetricConflicts[candidate]) == 0 {
			conflictFree = append(conflictFree, candidate)
		} else {
			conflictingCandidates = append(conflictingCandidates, candidate)
		}
	}

	remaining := conflictingCandidates

	for len(remaining) > 0 {
		currentSet := []string{}
		newRemaining := []string{}

		for _, candidate := range remaining {
			if !conflictsWithSet(symmetricConflicts, candidate, currentSet) {
				currentSet = append(currentSet, candidate)
			} else {
				newRemaining = append(newRemaining, candidate)
			}
		}

		for _, candidate := range conflictFree {
			if !conflictsWithSet(symmetricConflicts, candidate, currentSet) {
				currentSet = append(currentSet, candidate)
			}
		}

		result = append(result, currentSet)
		remaining = newRemaining
	}

	return result
}
