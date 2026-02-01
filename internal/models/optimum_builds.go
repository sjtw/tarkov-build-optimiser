package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type EvaluationConstraints struct {
	TraderLevels     []TraderLevel
	IgnoredSlotNames []string
	IgnoredItemIDs   []string
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

	tx, err := db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	query := `INSERT INTO optimum_builds (
			item_id,
			build_type,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		returning build_id;`
	var buildID int
	err = tx.QueryRow(
		query,
		id,
		evaluationType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
	).Scan(&buildID)
	if err != nil {
		return -1, err
	}

	queryStatus := `INSERT INTO optimal_build_status (
			build_id,
			status,
			evaluation_start
		)
		VALUES ($1, $2, $3);`
	_, err = tx.Exec(
		queryStatus,
		buildID,
		EvaluationPending.ToString(),
		time.Now(),
	)
	if err != nil {
		return -1, err
	}

	return buildID, tx.Commit()
}

func SetBuildInProgress(db *sql.DB, buildID int) error {
	query := `UPDATE optimal_build_status
		SET status = $1,
		    evaluation_start = $2
		WHERE build_id = $3;`
	_, err := db.Exec(query, EvaluationInProgress.ToString(), time.Now(), buildID)
	if err != nil {
		return err
	}

	return nil
}

func SetBuildFailed(db *sql.DB, buildID int) error {
	query := `UPDATE optimal_build_status
		SET status = $1,
		    evaluation_end = $2
		WHERE build_id = $3;`
	_, err := db.Exec(query, EvaluationFailed.ToString(), time.Now(), buildID)
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryBuild := `update optimum_builds set
			build = $1,
			is_subtree = $2,
            recoil_sum = $3,
			ergonomics_sum = $4
		where build_id = $5;`
	_, err = tx.Exec(
		queryBuild,
		serialisedBuild,
		build.IsSubtree,
		build.RecoilSum,
		build.ErgonomicsSum,
		buildID)
	if err != nil {
		return err
	}

	queryStatus := `update optimal_build_status set
			status = $1,
			evaluation_end = $2
		where build_id = $3;`
	_, err = tx.Exec(
		queryStatus,
		EvaluationCompleted.ToString(),
		time.Now(),
		buildID)
	if err != nil {
		return err
	}

	return tx.Commit()
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

func GetOptimumBuildByConstraints(db *sql.DB, itemId string, buildType string, constraints EvaluationConstraints) (*ItemEvaluationResult, error) {
	tradersMap := constraintsToTraderMap(constraints)

	query := `
		SELECT
		    ob.build_id,
			ob.build,
			obs.status
		FROM optimum_builds ob
		JOIN optimal_build_status obs ON ob.build_id = obs.build_id
		WHERE
		    ob.item_id = $1
			AND ob.build_type = $2
			AND ob.jaeger_level = $3
			AND ob.prapor_level = $4
			AND ob.peacekeeper_level = $5
			AND ob.mechanic_level = $6
			AND ob.skier_level = $7;`
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
	var buildID int
	var status string
	for rows.Next() {
		result := ItemEvaluationResult{}
		var build sql.NullString
		err := rows.Scan(&buildID, &build, &status)
		if err != nil {
			return nil, err
		}

		if build.Valid {
			if err := json.Unmarshal([]byte(build.String), &result); err != nil {
				return nil, err
			}
		}

		result.BuildID = buildID
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

func ResetInProgressBuilds(db *sql.DB) error {
	query := `UPDATE optimal_build_status
		SET status = $1
		WHERE status = $2;`
	_, err := db.Exec(query, EvaluationPending.ToString(), EvaluationInProgress.ToString())
	return err
}

func PurgeOptimumBuilds(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE optimum_builds CASCADE;")
	return err
}
