package evaluator

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/weapon_tree"
)

type EvaluationDataProvider interface {
	GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error)
	GetTraderOffer(itemID string) ([]models.TraderOffer, error)
	SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints)
}

// WeaponEvaluationTask represents a task for evaluation. contains the weapon tree & additional constraints to be
// taken into account during evaluation.
type WeaponEvaluationTask struct {
	Constraints    models.EvaluationConstraints
	WeaponTree     weapon_tree.WeaponTree
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

func (e *Evaluator) EvaluateWeaponEvaluationTask(task WeaponEvaluationTask) (models.ItemEvaluationResult, error) {
	filteredCandidates, err := e.filterCandidateItemsByTraderAvailability(task.WeaponTree.CandidateItems, task.Constraints.TraderLevels)
	if err != nil {
		log.Error().Err(err).Msg("Failed to filter candidate items by trader availability")
		return models.ItemEvaluationResult{}, err
	}

	candidateSets := make([][]string, 0)

	if len(task.WeaponTree.AllowedItemConflicts) > 0 {
		// generate all valid variations of candidates, including the weapon ID
		candidateSets = generateNonConflictingCandidateSets(filteredCandidates, task.WeaponTree.AllowedItemConflicts)
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
	results := make([]models.ItemEvaluationResult, len(candidateSets))
	for index, candidateItems := range candidateSets {
		log.Info().Msgf("index %d", index)
		result, err := e.evaluateWeapon(task.WeaponTree.Item, task.EvaluationType, task.Constraints, candidateItems)
		if err != nil {
			return models.ItemEvaluationResult{}, err
		}

		if task.EvaluationType == "recoil" {
			if result.RecoilSum < optimumSum {
				optimumSum = result.RecoilSum
				optimum = result
			}
		} else {
			if result.ErgonomicsSum > optimumSum {
				optimumSum = result.ErgonomicsSum
				optimum = result
			}
		}

		results[index] = result
	}

	optimum.IsSubtree = false

	return optimum, nil
}

func (e *Evaluator) evaluateSlot(slotId string, slotName string, allowedItems []*weapon_tree.Item, evaluationType string, constraints models.EvaluationConstraints, candidates []string) (models.SlotEvaluationResult, error) {
	if slotName == "Barrel" {
		log.Info().Msgf("Barrel slot")
	}
	slotEvaluationResult := models.SlotEvaluationResult{
		ID:      slotId,
		Name:    slotName,
		Item:    models.ItemEvaluationResult{},
		IsEmpty: true,
	}

	if isIgnoredSlotName(slotName, constraints.IgnoredSlotNames) {
		return slotEvaluationResult, nil
	}

	var bestItem models.ItemEvaluationResult
	bestErgoSum := 0
	bestRecoilSum := 0

	if slotName == "Stock" {
		log.Info().Msgf("Stock slot")
	}

	if slotName == "Barrel" {
		log.Info().Msgf("Barrel slot")
	}

	for j := 0; j < len(allowedItems); j++ {
		log.Debug().Msgf("Evaluating slot %s", slotName)
		ai := allowedItems[j]

		isCandidate := false
		for _, candidateID := range candidates {
			if candidateID == ai.ID {
				isCandidate = true
			}
		}

		if !isCandidate {
			log.Info().Msgf("Candidate item [%s] is not a whitelisted candidate for this evaluation", ai.ID)
			continue
		} else {
			log.Info().Msgf("Candidate item [%s] is a valid candidate for this evaluation", ai.ID)
		}

		evaluatedAllowedItem, err := e.evaluateItem(ai, evaluationType, constraints, candidates)
		if err != nil {
			log.Debug().Msg("Failed to evaluate highest item")
			return slotEvaluationResult, err
		}

		log.Debug().Msgf("Evaluation type %s", evaluationType)
		if evaluationType == "ergonomics" {
			if evaluatedAllowedItem.ErgonomicsSum > bestErgoSum {
				log.Debug().Msgf("item %v IS new highest sum", evaluatedAllowedItem)
				bestErgoSum = evaluatedAllowedItem.ErgonomicsSum
				bestRecoilSum = evaluatedAllowedItem.RecoilSum
				bestItem = evaluatedAllowedItem
			} else if evaluatedAllowedItem.RecoilSum < bestRecoilSum {
				log.Debug().Msgf("Item %v does not improve ergonomics, but improves recoil over empty slot.", evaluatedAllowedItem)
				bestErgoSum = evaluatedAllowedItem.ErgonomicsSum
				bestRecoilSum = evaluatedAllowedItem.RecoilSum
				bestItem = evaluatedAllowedItem
			} else {
				log.Debug().Msgf("item %v does not improve stats", evaluatedAllowedItem)
			}
		} else if evaluationType == "recoil" {
			if evaluatedAllowedItem.RecoilSum < bestRecoilSum {
				log.Debug().Msgf("item %v IS new highest sum", evaluatedAllowedItem)
				bestErgoSum = evaluatedAllowedItem.ErgonomicsSum
				bestRecoilSum = evaluatedAllowedItem.RecoilSum
				bestItem = evaluatedAllowedItem
			} else if evaluatedAllowedItem.ErgonomicsSum > bestErgoSum {
				log.Debug().Msgf("Item %v does not improve recoil, but improves ergo over empty slot.", evaluatedAllowedItem)
				bestErgoSum = evaluatedAllowedItem.ErgonomicsSum
				bestRecoilSum = evaluatedAllowedItem.RecoilSum
				bestItem = evaluatedAllowedItem
			} else {
				log.Debug().Msgf("item %v does not improve stats", bestItem)
			}
		} else {
			msg := fmt.Sprintf("Invalid evaluation type [%s]", evaluationType)
			log.Error().Msg(msg)
			return slotEvaluationResult, errors.New(msg)
		}
	}

	slotEvaluationResult.Item = bestItem
	if slotEvaluationResult.Item.ID != "" {
		slotEvaluationResult.IsEmpty = false
	}

	return slotEvaluationResult, nil
}

func (e *Evaluator) evaluateWeapon(item *weapon_tree.Item, evaluationType string, constraints models.EvaluationConstraints, candidates []string) (models.ItemEvaluationResult, error) {
	return e.evaluateItem(item, evaluationType, constraints, candidates)
}

func (e *Evaluator) evaluateItem(item *weapon_tree.Item, evaluationType string, constraints models.EvaluationConstraints, candidates []string) (models.ItemEvaluationResult, error) {
	outItem := models.ItemEvaluationResult{
		ID:                 item.ID,
		Name:               item.Name,
		EvaluationType:     evaluationType,
		RecoilModifier:     item.RecoilModifier,
		ErgonomicsModifier: item.ErgonomicsModifier,
		Slots:              make([]models.SlotEvaluationResult, len(item.Slots)),
		ErgonomicsSum:      item.ErgonomicsModifier,
		RecoilSum:          item.RecoilModifier,
		IsSubtree:          true,
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

		if item.Slots[i].Name == "Mount" {
			log.Info().Msgf("Mount slot")
		}

		slotResult, err := e.evaluateSlot(item.Slots[i].ID, item.Slots[i].Name, item.Slots[i].AllowedItems, evaluationType, constraints, candidates)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to evaluate slot %s", item.Slots[i].ID)
			return models.ItemEvaluationResult{}, err
		}

		outItem.Slots[i] = slotResult
		outItem.RecoilSum += slotResult.Item.RecoilSum
		outItem.ErgonomicsSum += slotResult.Item.ErgonomicsSum
	}

	log.Debug().Msgf("Output item %v", outItem)

	return outItem, nil
}

func isIgnoredSlotName(slotName string, ignoredSlots []string) bool {
	for _, ignoredSlot := range ignoredSlots {
		if slotName == ignoredSlot {
			return true
		}
	}

	return false
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
func generateNonConflictingCandidateSets(candidates map[string]bool, conflicts map[string]map[string]bool) [][]string {
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
