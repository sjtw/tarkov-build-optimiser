package models

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog/log"
)

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

type TraderLevel struct {
	Name  string
	Level int
}

var TraderNames = []string{"Jaeger", "Prapor", "Peacekeeper", "Mechanic", "Skier"}

func UpsertOptimumBuild(db *sql.DB, itemId string, buildType string, itemType string, sum int, build *ItemEvaluationResult, name string, constraints EvaluationConstraints) error {
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
			item_type,
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
		itemType,
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
