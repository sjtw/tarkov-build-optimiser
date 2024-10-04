package evaluator

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

func GenerateOptimumWeaponBuilds(db *sql.DB, weaponId string, constraints models.EvaluationConstraints) error {
	weapon, err := createWeaponPossibilityTree(db, weaponId)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", weaponId)
		return err
	}

	traderOfferGetter := CreatePgTraderOfferGetter(db)
	buildSaver := CreatePgBuildSaver(db)
	subtreeGetter := CreatePgSubtreeGetter(db)
	evaluator := CreateEvaluator(traderOfferGetter, buildSaver, subtreeGetter)

	_, err = evaluator.evaluate(weapon, "recoil", constraints)
	if err != nil {
		return err
	}

	_, err = evaluator.evaluate(weapon, "ergonomics", constraints)
	if err != nil {
		return err
	}

	return nil
}

func createWeaponPossibilityTree(db *sql.DB, id string) (*Item, error) {
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

	err = weapon.PopulateSlots(db)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", w.ID)
		return nil, err
	}

	return weapon, nil
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

	for i := 0; i < len(item.Slots); i++ {
		if item.Slots[i].AllowedItems == nil || len(item.Slots[i].AllowedItems) == 0 {
			return outItem, nil
		}

		outSlot := &models.SlotEvaluationResult{
			ID:   item.Slots[i].ID,
			Name: item.Slots[i].Name,
			Item: nil,
		}

		slotErgo := 0
		slotRecoil := 0
		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			ai := item.Slots[i].AllowedItems[j]
			offers, err := e.traderOfferGetter.Get(ai.ID)
			if err != nil {
				return nil, err
			}

			valid := validateConstraints(offers, constraints)
			if valid != true {
				break
			}

			var highestItem *models.ItemEvaluationResult
			highestItem, err = e.subtreeGetter.Get(ai.ID, evaluationType, constraints)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to get subtree for evaluation. item: %s", ai.ID)
				return nil, err
			}

			if highestItem == nil {
				log.Info().Msgf("Item subtree not yet evaluated. item: %s", ai.ID)
				highestItem, err = e.evaluate(ai, evaluationType, constraints)
				if err != nil {
					return nil, err
				}
			} else {
				log.Info().Msgf("Item subtree already evaluated. item: %s", ai.ID)
			}

			recoilSum := highestItem.RecoilSum
			ergoSum := highestItem.ErgonomicsSum

			if evaluationType == "ergonomics" {
				if ergoSum > slotErgo {
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				}
			}

			if evaluationType == "recoil" {
				if recoilSum < slotRecoil {
					slotErgo = ergoSum
					slotRecoil = recoilSum
					outSlot.Item = highestItem
				}
			}
		}

		outItem.Slots[i] = outSlot
		outItem.RecoilSum += slotRecoil
		outItem.ErgonomicsSum += slotErgo
	}

	err := e.buildSaver.Save(outItem, constraints, outItem.IsSubtree)
	if err != nil {
		return nil, err
	}

	return outItem, nil
}
