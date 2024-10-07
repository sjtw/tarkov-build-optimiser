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
	SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) error
}

type Task struct {
	Constraints    models.EvaluationConstraints
	Weapon         Item
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

func (e *Evaluator) EvaluateTask(task Task) (models.ItemEvaluationResult, error) {
	result, err := e.evaluate(&task.Weapon, task.EvaluationType, task.Constraints)
	if err != nil {
		return models.ItemEvaluationResult{}, err
	}

	return *result, nil
}

func (e *Evaluator) evaluate(item *Item, evaluationType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	outItem := &models.ItemEvaluationResult{
		ID:                 item.ID,
		Name:               item.Name,
		EvaluationType:     evaluationType,
		RecoilModifier:     item.RecoilModifier,
		ErgonomicsModifier: item.ErgonomicsModifier,
		Slots:              make([]*models.SlotEvaluationResult, len(item.Slots)),
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

	preEvaluatedItem, err := e.dataService.GetSubtree(outItem.ID, evaluationType, constraints)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get subtree for evaluation. item: %s", preEvaluatedItem.ID)
		return nil, err
	}

	if preEvaluatedItem != nil {
		log.Debug().Msgf("Optimal [%s] evaluation for item [%s] already evaluated. returning.", outItem.EvaluationType, item.ID)
		return preEvaluatedItem, nil
	}

	for i := 0; i < len(item.Slots); i++ {
		log.Debug().Msgf("Evaluating slot %d for item [%s]", i, item.ID)

		outSlot := &models.SlotEvaluationResult{
			ID:   item.Slots[i].ID,
			Name: item.Slots[i].Name,
			Item: nil,
		}

		slotErgo := 0
		slotRecoil := 0
		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			log.Debug().Msgf("Evaluating slot %d for item [%s]", j, item.ID)
			ai := item.Slots[i].AllowedItems[j]
			offers, err := e.dataService.GetTraderOffer(ai.ID)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to get trader offer for item [%s]", item.ID)
				return nil, err
			}

			valid := validateConstraints(offers, constraints)
			if valid != true {
				log.Debug().Msg("item does not meet build constraints, skipping")
				continue
			}

			highestItem, err := e.evaluate(ai, evaluationType, constraints)
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

		outItem.Slots[i] = outSlot
		outItem.RecoilSum += slotRecoil
		outItem.ErgonomicsSum += slotErgo
	}

	go func(out models.ItemEvaluationResult, constraints models.EvaluationConstraints) {
		err = e.dataService.SaveBuild(&out, constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to save evaluation result for item: %s", outItem.ID)
		}
	}(*outItem, constraints)

	log.Debug().Msgf("Output item %v", outItem)

	return outItem, nil
}
