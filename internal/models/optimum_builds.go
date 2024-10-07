package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
)

type EvaluationConstraints struct {
	TraderLevels []TraderLevel
}

type ItemEvaluationResult struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	EvaluationType     string                  `json:"evaluation_type"`
	IsSubtree          bool                    `json:"is_subtree"`
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

func constraintsToTraderMap(constraints EvaluationConstraints) map[string]int {
	tradersMap := make(map[string]int)

	for i := 0; i < len(constraints.TraderLevels); i++ {
		tradersMap[constraints.TraderLevels[i].Name] = constraints.TraderLevels[i].Level
	}

	return tradersMap
}

// TODO: stop using []TraderLevel and use map[string]int instead
func TraderMapToTraderLevels(tradersMap map[string]int) []TraderLevel {
	levels := []TraderLevel{}

	for i := 0; i < len(TraderNames); i++ {
		traderLevel := TraderLevel{
			Name:  TraderNames[i],
			Level: tradersMap[TraderNames[i]],
		}
		levels = append(levels, traderLevel)
	}

	return levels
}

func SerialiseLevels(levels []TraderLevel) string {
	str := ""
	for i := 0; i < len(levels); i++ {
		str += fmt.Sprintf("%s-%d", levels[i].Name, levels[i].Level)
	}

	return str
}

func UpsertOptimumBuild(db *sql.DB, build *ItemEvaluationResult, constraints EvaluationConstraints) error {
	serialisedBuild, err := json.Marshal(build)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal build")
		return err
	}

	tradersMap := constraintsToTraderMap(constraints)

	query := `INSERT INTO optimum_builds (
			item_id,
			build,
			build_type,
			is_subtree,
            recoil_sum,
			ergonomics_sum,
            name,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (
		    item_id,
		    build_type,
		    jaeger_level,
		    prapor_level,
		    peacekeeper_level,
		    mechanic_level,
		    skier_level
		) DO UPDATE SET
			build = $2,
			recoil_sum = $5,
			ergonomics_sum = $6;`
	_, err = db.Exec(
		query,
		build.ID,
		serialisedBuild,
		build.EvaluationType,
		build.IsSubtree,
		build.RecoilSum,
		build.ErgonomicsSum,
		build.Name,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"])
	if err != nil {
		return err
	}

	return nil
}

func GetEvaluatedSubtree(ctx context.Context, db *sql.DB, itemId string, buildType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	tradersMap := constraintsToTraderMap(constraints)

	query := `
		SELECT
			build
		FROM optimum_builds
		where item_id = $1
			and build_type = $2
			and jaeger_level = $3
			and prapor_level = $4
			and peacekeeper_level = $5
			and mechanic_level = $6
			and skier_level = $7;`
	rows, err := db.QueryContext(
		ctx,
		query,
		itemId,
		buildType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ItemEvaluationResult
	for rows.Next() {
		result := ItemEvaluationResult{}
		var build string
		err := rows.Scan(&build)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(build), &result); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) > 1 {
		msg := fmt.Sprintf("Multiple Evaluated Subtrees found for: itemId: %s, buildType: %s, constraints: %v", itemId, buildType, constraints)
		return nil, errors.New(msg)
	}

	return &results[0], nil
}

func GetOptimumBuild(db *sql.DB, itemId string, buildType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	tradersMap := constraintsToTraderMap(constraints)

	query := `
		SELECT
			build
		FROM optimum_builds
		WHERE is_subtree = false
			AND item_id = $1
			AND build_type = $2
			AND jaeger_level = $3
			AND prapor_level = $4
			AND peacekeeper_level = $5
			AND mechanic_level = $6
			AND skier_level = $7;`
	rows, err := db.Query(
		query,
		itemId,
		buildType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ItemEvaluationResult
	for rows.Next() {
		result := ItemEvaluationResult{}
		var build string
		err := rows.Scan(&build)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(build), &result); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) > 1 {
		msg := fmt.Sprintf("Multiple Optimum Builds found for: itemId: %s, buildType: %s, constraints: %v", itemId, buildType, constraints)
		return nil, errors.New(msg)
	}

	return &results[0], nil
}

func PurgeOptimumBuilds(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE optimum_builds;")
	if err != nil {
		return err
	}

	return nil
}
