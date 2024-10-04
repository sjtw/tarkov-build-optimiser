package evaluator

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

func GenerateOptimumWeaponBuilds(db *sql.DB, weaponId string) error {
	weapon, err := createWeaponPossibilityTree(db, weaponId)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", weaponId)
		return err
	}

	traderOfferGetter := CreateTraderOfferGetter(db)

	constraints := EvaluationConstraints{
		TraderLevels: []TraderLevel{
			{Name: "Jaeger", Level: 5},
			{Name: "Prapor", Level: 5},
			{Name: "Peacekeeper", Level: 5},
			{Name: "Mechanic", Level: 5},
			{Name: "Skier", Level: 5},
		},
	}

	bestRecoilItem, err := evaluate(traderOfferGetter, weapon, "recoil", constraints)
	if err != nil {
		return err
	}
	err = upsertOptimumBuild(db, weaponId, "recoil", bestRecoilItem.RecoilSum, bestRecoilItem, bestRecoilItem.Name, constraints)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to upsert optimum recoil build")
		return err
	}

	bestErgoItem, err := evaluate(traderOfferGetter, weapon, "ergonomics", constraints)
	if err != nil {
		return err
	}
	err = upsertOptimumBuild(db, weaponId, "ergo", bestErgoItem.ErgonomicsSum, bestErgoItem, bestErgoItem.Name, constraints)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to upsert optimum ergo build")
		return err
	}

	return nil
}

func upsertOptimumBuild(db *sql.DB, itemId string, buildType string, sum int, build *ItemEvaluationResult, name string, constraints EvaluationConstraints) error {
	b, err := json.Marshal(build)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal build")
		return err
	}

	tradersMap := make(map[string]int)

	for i := 0; i < len(constraints.TraderLevels); i++ {
		tradersMap[constraints.TraderLevels[i].Name] = constraints.TraderLevels[i].Level
	}

	query := `INSERT INTO optimum_builds (
			item_id,
			build,
			build_type,
            modifier_sum,
            name,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`
	_, err = db.Exec(
		query,
		itemId,
		b,
		buildType,
		sum,
		name,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
	)
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

type TraderLevel struct {
	Name  string
	Level int
}

type EvaluationConstraints struct {
	TraderLevels []TraderLevel
}

type ItemEvaluationResult struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	EvaluationType     string                  `json:"evaluation_type"`
	RecoilModifier     int                     `json:"recoil_modifier"`
	ErgonomicsModifier int                     `json:"ergonomics_modifier"`
	Slots              []*SlotEvaluationResult `json:"slots"`
	RecoilSum          int                     `json:"recoil_sum"`
	ErgonomicsSum      int                     `json:"ergonomics_sum"`
}

type SlotEvaluationResult struct {
	ID   string                `json:"id"`
	Name string                `json:"name"`
	Item *ItemEvaluationResult `json:"item"`
}

func validateConstraints(offers []models.TraderOffer, constraints EvaluationConstraints) bool {
	for i := 0; i < len(offers); i++ {
		for j := i + 1; j < len(constraints.TraderLevels); j++ {
			tc := constraints.TraderLevels[j]
			if offers[i].Trader == tc.Name && tc.Level >= offers[i].MinTraderLevel {
				return true
			}
		}
	}

	return false
}

type TraderOfferGetter interface {
	Get(itemID string) ([]models.TraderOffer, error)
}

type TraderOffers struct {
	db *sql.DB
}

func CreateTraderOfferGetter(db *sql.DB) TraderOfferGetter {
	return &TraderOffers{
		db: db,
	}
}

func (to *TraderOffers) Get(itemID string) ([]models.TraderOffer, error) {
	offers, err := models.GetTraderOffersByItemID(to.db, itemID)
	if err != nil {
		return nil, err
	}
	return offers, nil
}

func evaluate(to TraderOfferGetter, item *Item, evaluationType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	outItem := &ItemEvaluationResult{
		ID:                 item.ID,
		Name:               item.Name,
		EvaluationType:     evaluationType,
		RecoilModifier:     item.RecoilModifier,
		ErgonomicsModifier: item.ErgonomicsModifier,
		Slots:              make([]*SlotEvaluationResult, len(item.Slots)),
		ErgonomicsSum:      item.ErgonomicsModifier,
		RecoilSum:          item.RecoilModifier,
	}

	if item.Slots == nil {
		return outItem, nil
	}

	for i := 0; i < len(item.Slots); i++ {
		if item.Slots[i].AllowedItems == nil || len(item.Slots[i].AllowedItems) == 0 {
			return outItem, nil
		}

		outSlot := &SlotEvaluationResult{
			ID:   item.Slots[i].ID,
			Name: item.Slots[i].Name,
			Item: nil,
		}

		slotErgo := 0
		slotRecoil := 0
		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			ai := item.Slots[i].AllowedItems[j]
			offers, err := to.Get(ai.ID)
			if err != nil {
				return nil, err
			}

			valid := validateConstraints(offers, constraints)
			if valid != true {
				break
			}

			highestItem, err := evaluate(to, ai, evaluationType, constraints)
			if err != nil {
				return nil, err
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

	return outItem, nil
}
