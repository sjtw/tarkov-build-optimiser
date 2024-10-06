package evaluator

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

func GenerateOptimumWeaponBuilds(db *sql.DB, weapon Item, constraints models.EvaluationConstraints) error {

	traderOfferGetter := CreatePgTraderOfferGetter(db)
	buildSaver := CreatePgBuildSaver(db)
	subtreeGetter := CreatePgSubtreeGetter(db)
	evaluator := CreateEvaluator(traderOfferGetter, buildSaver, subtreeGetter)

	log.Debug().Msgf("Evaluating recoil for %s", weapon.ID)
	_, err := evaluator.evaluate(&weapon, "recoil", constraints)
	if err != nil {
		return err
	}

	_, err = evaluator.evaluate(&weapon, "ergonomics", constraints)
	if err != nil {
		return err
	}

	return nil
}

func CreateWeaponPossibilityTree(db *sql.DB, id string) (*Item, error) {
	w, err := models.GetWeaponById(db, id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}
	weapon := &Item{
		ID:                 id,
		Name:               w.Name,
		RecoilModifier:     w.RecoilModifier,
		ErgonomicsModifier: w.ErgonomicsModifier,
		Slots:              []*ItemSlot{},
		parentSlot:         nil,
		Type:               "weapon",
	}

	err = weapon.PopulateSlots(db, []string{"Sight", "Ubgl"})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", w.ID)
		return nil, err
	}

	return weapon, nil
}

type SubtreeGetter interface {
	Get(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error)
}

type TraderOfferGetter interface {
	Get(itemID string) ([]models.TraderOffer, error)
}

type BuildSaver interface {
	Save(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints, isSubtree bool) error
}

type Evaluator struct {
	traderOfferGetter TraderOfferGetter
	buildSaver        BuildSaver
	subtreeGetter     SubtreeGetter
}

func CreateEvaluator(traderOfferGetter TraderOfferGetter, buildSaver BuildSaver, subtreeGetter SubtreeGetter) *Evaluator {
	return &Evaluator{
		traderOfferGetter: traderOfferGetter,
		buildSaver:        buildSaver,
		subtreeGetter:     subtreeGetter,
	}
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

	preEvaluatedItem, err := e.subtreeGetter.Get(outItem.ID, evaluationType, constraints)
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
			offers, err := e.traderOfferGetter.Get(ai.ID)
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

	err = e.buildSaver.Save(outItem, constraints, outItem.IsSubtree)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to save evaluation result for item: %s", outItem.ID)
		return nil, err
	}

	log.Debug().Msgf("Output item %v", outItem)

	return outItem, nil
}
