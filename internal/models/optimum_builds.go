package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"time"
)

type EvaluationConstraints struct {
	TraderLevels     []TraderLevel
	IgnoredSlotNames []string
	IgnoredItemIDs   []string
	RubBudget        *int `json:"rub_budget,omitempty"`
}

type ItemEvaluationResult struct {
	BuildID            int                    `json:"build_id"`
	Status             string                 `json:"status"`
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	EvaluationType     string                 `json:"evaluation_type"`
	IsSubtree          bool                   `json:"is_subtree"`
	RecoilModifier     int                    `json:"recoil_modifier"`
	ErgonomicsModifier int                    `json:"ergonomics_modifier"`
	Slots              []SlotEvaluationResult `json:"slots"`
	RecoilSum          int                    `json:"recoil_sum"`
	ErgonomicsSum      int                    `json:"ergonomics_sum"`
	TotalCost          int                    `json:"total_cost"`
	WeaponBaseCost     int                    `json:"weapon_base_cost"`
	RubBudget          *int                   `json:"rub_budget,omitempty"`
}

type SlotEvaluationResult struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Item    ItemEvaluationResult `json:"item"`
	IsEmpty bool                 `json:"empty"`
}

// MarshalJSON - custom JSON marshalling for SlotEvaluationResult to handle empty slots
func (s *SlotEvaluationResult) MarshalJSON() ([]byte, error) {
	if s.IsEmpty {
		return json.Marshal(map[string]interface{}{
			"id":   s.ID,
			"name": s.Name,
			"item": nil,
		})
	}

	return json.Marshal(map[string]interface{}{
		"id":   s.ID,
		"name": s.Name,
		"item": s.Item,
	})
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

func SerialiseLevels(levels []TraderLevel) string {
	str := ""
	for i := 0; i < len(levels); i++ {
		str += fmt.Sprintf("%s-%d", levels[i].Name, levels[i].Level)
	}

	return str
}

type EvaluatorStatus int

const (
	EvaluationPending EvaluatorStatus = iota // iota automatically increments starting from 0
	EvaluationInProgress
	EvaluationCompleted
	EvaluationFailed
)

func (s EvaluatorStatus) ToString() string {
	return [...]string{"Pending", "InProgress", "Completed", "Failed"}[s]
}

func CreatePendingOptimumBuild(db *sql.DB, id string, evaluationType string, constraints EvaluationConstraints) (int, error) {
	tradersMap := constraintsToTraderMap(constraints)

	var rubBudget sql.NullInt64
	if constraints.RubBudget != nil {
		rubBudget = sql.NullInt64{
			Int64: int64(*constraints.RubBudget),
			Valid: true,
		}
	}

	query := `INSERT INTO optimum_builds (
			item_id,
			build_type,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level,
			rub_budget,
			evaluation_start,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		returning build_id;`
	var buildID int
	err := db.QueryRow(
		query,
		id,
		evaluationType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
		rubBudget,
		time.Now(),
		EvaluationPending.ToString(),
	).Scan(&buildID)
	if err != nil {
		return -1, err
	}

	return buildID, nil
}

func SetBuildInProgress(db *sql.DB, buildID int) error {
	query := `UPDATE optimum_builds
		SET status = $1
		WHERE build_id = $2;`
	_, err := db.Exec(query, EvaluationInProgress.ToString(), buildID)
	if err != nil {
		return err
	}

	return nil
}

func SetBuildFailed(db *sql.DB, buildID int) error {
	query := `UPDATE optimum_builds
		SET status = $1
		WHERE build_id = $2;`
	_, err := db.Exec(query, EvaluationFailed.ToString(), buildID)
	if err != nil {
		return err
	}

	return nil
}

func SetBuildCompleted(db *sql.DB, buildID int, build *ItemEvaluationResult) error {
	serialisedBuild, err := json.Marshal(build)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal build")
		return err
	}

	query := `update optimum_builds set
			build = $1,
			is_subtree = $2,
            recoil_sum = $3,
			ergonomics_sum = $4,
			total_cost = $5,
			weapon_base_cost = $6,
			status = $7,
			evaluation_end = $8
		where build_id = $9;`
	_, err = db.Exec(
		query,
		serialisedBuild,
		build.IsSubtree,
		build.RecoilSum,
		build.ErgonomicsSum,
		build.TotalCost,
		build.WeaponBaseCost,
		EvaluationCompleted.ToString(),
		time.Now(),
		buildID)
	if err != nil {
		return err
	}

	return nil
}

func GetEvaluatedSubtree(ctx context.Context, db *sql.DB, itemId string, buildType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	tradersMap := constraintsToTraderMap(constraints)

	var rubBudget sql.NullInt64
	if constraints.RubBudget != nil {
		rubBudget = sql.NullInt64{
			Int64: int64(*constraints.RubBudget),
			Valid: true,
		}
	}

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
			and skier_level = $7
			and coalesce(rub_budget, -1) = coalesce($8, -1);`
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
		rubBudget,
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

func GetOptimumBuildByConstraints(db *sql.DB, itemId string, buildType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	tradersMap := constraintsToTraderMap(constraints)
	var rubBudget sql.NullInt64
	if constraints.RubBudget != nil {
		rubBudget = sql.NullInt64{
			Int64: int64(*constraints.RubBudget),
			Valid: true,
		}
	}

	query := `
		SELECT
		    build_id,
			build,
			status,
			total_cost,
			weapon_base_cost,
			rub_budget
		FROM optimum_builds
		WHERE
		    item_id = $1
			AND build_type = $2
			AND jaeger_level = $3
			AND prapor_level = $4
			AND peacekeeper_level = $5
			AND mechanic_level = $6
			AND skier_level = $7
			AND coalesce(rub_budget, -1) = coalesce($8, -1);`
	rows, err := db.Query(
		query,
		itemId,
		buildType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
		rubBudget,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ItemEvaluationResult
	var buildID int
	var status string
	var totalCost sql.NullInt64
	var baseCost sql.NullInt64
	var storedBudget sql.NullInt64
	for rows.Next() {
		result := ItemEvaluationResult{}
		var build sql.NullString
		err := rows.Scan(&buildID, &build, &status, &totalCost, &baseCost, &storedBudget)
		if err != nil {
			return nil, err
		}

		if build.Valid {
			if err := json.Unmarshal([]byte(build.String), &result); err != nil {
				return nil, err
			}
		}

		result.BuildID = buildID
		if totalCost.Valid {
			result.TotalCost = int(totalCost.Int64)
		}
		if baseCost.Valid {
			result.WeaponBaseCost = int(baseCost.Int64)
		}
		if storedBudget.Valid {
			val := int(storedBudget.Int64)
			result.RubBudget = &val
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

	results[0].Status = status

	return &results[0], nil
}

func PurgeOptimumBuilds(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE optimum_builds;")
	if err != nil {
		return err
	}

	return nil
}
