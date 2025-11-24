package queue

import (
	"database/sql"
	"fmt"
	"tarkov-build-optimiser/internal/models"
	"time"

	"github.com/rs/zerolog/log"
)

// Priority constants for queue entries
const (
	PriorityAPI   = 100 // User-requested via API
	PriorityBatch = 10  // Batch evaluator
)

// QueueStatus represents the status of a queue entry
type QueueStatus string

const (
	StatusQueued     QueueStatus = "Queued"
	StatusProcessing QueueStatus = "Processing"
	StatusCompleted  QueueStatus = "Completed"
	StatusFailed     QueueStatus = "Failed"
)

// QueueEntry represents a build evaluation job in the queue
type QueueEntry struct {
	QueueID      int
	ItemID       string
	BuildType    string
	Constraints  models.EvaluationConstraints
	Priority     int
	Status       QueueStatus
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	ErrorMessage *string
}

// constraintsToTraderMap converts EvaluationConstraints to a map of trader levels
func constraintsToTraderMap(constraints models.EvaluationConstraints) map[string]int {
	tradersMap := make(map[string]int)
	for i := 0; i < len(constraints.TraderLevels); i++ {
		tradersMap[constraints.TraderLevels[i].Name] = constraints.TraderLevels[i].Level
	}
	return tradersMap
}

// traderMapToConstraints converts a trader map back to EvaluationConstraints
func traderMapToConstraints(tradersMap map[string]int) models.EvaluationConstraints {
	traderLevels := make([]models.TraderLevel, 0, len(tradersMap))
	for name, level := range tradersMap {
		traderLevels = append(traderLevels, models.TraderLevel{
			Name:  name,
			Level: level,
		})
	}
	return models.EvaluationConstraints{
		TraderLevels:     traderLevels,
		IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
		IgnoredItemIDs:   []string{},
	}
}

// CreateQueueEntry adds a new build evaluation job to the queue
func CreateQueueEntry(db *sql.DB, itemID string, buildType string, constraints models.EvaluationConstraints, priority int) (int, error) {
	tradersMap := constraintsToTraderMap(constraints)

	query := `INSERT INTO build_queue (
		item_id,
		build_type,
		jaeger_level,
		prapor_level,
		peacekeeper_level,
		mechanic_level,
		skier_level,
		priority,
		status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING queue_id;`

	var queueID int
	err := db.QueryRow(
		query,
		itemID,
		buildType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
		priority,
		StatusQueued,
	).Scan(&queueID)

	if err != nil {
		return -1, fmt.Errorf("failed to create queue entry: %w", err)
	}

	log.Info().Msgf("Created queue entry %d for item %s with priority %d", queueID, itemID, priority)
	return queueID, nil
}

// GetNextQueuedBuild fetches the next pending job from the queue (highest priority first)
func GetNextQueuedBuild(db *sql.DB) (*QueueEntry, error) {
	query := `
		SELECT 
			queue_id,
			item_id,
			build_type,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level,
			priority,
			status,
			created_at,
			started_at,
			completed_at,
			error_message
		FROM build_queue
		WHERE status = $1
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED;`

	row := db.QueryRow(query, StatusQueued)

	var entry QueueEntry
	var jaegerLevel, praporLevel, peacekeeperLevel, mechanicLevel, skierLevel int

	err := row.Scan(
		&entry.QueueID,
		&entry.ItemID,
		&entry.BuildType,
		&jaegerLevel,
		&praporLevel,
		&peacekeeperLevel,
		&mechanicLevel,
		&skierLevel,
		&entry.Priority,
		&entry.Status,
		&entry.CreatedAt,
		&entry.StartedAt,
		&entry.CompletedAt,
		&entry.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No jobs available
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch next queued build: %w", err)
	}

	// Convert trader levels to constraints
	tradersMap := map[string]int{
		"Jaeger":      jaegerLevel,
		"Prapor":      praporLevel,
		"Peacekeeper": peacekeeperLevel,
		"Mechanic":    mechanicLevel,
		"Skier":       skierLevel,
	}
	entry.Constraints = traderMapToConstraints(tradersMap)

	return &entry, nil
}

// SetQueueProcessing marks a queue entry as being processed
func SetQueueProcessing(db *sql.DB, queueID int) error {
	query := `
		UPDATE build_queue
		SET status = $1, started_at = $2
		WHERE queue_id = $3;`

	_, err := db.Exec(query, StatusProcessing, time.Now(), queueID)
	if err != nil {
		return fmt.Errorf("failed to set queue entry %d as processing: %w", queueID, err)
	}

	log.Debug().Msgf("Queue entry %d marked as processing", queueID)
	return nil
}

// SetQueueCompleted marks a queue entry as completed
func SetQueueCompleted(db *sql.DB, queueID int) error {
	query := `
		UPDATE build_queue
		SET status = $1, completed_at = $2
		WHERE queue_id = $3;`

	_, err := db.Exec(query, StatusCompleted, time.Now(), queueID)
	if err != nil {
		return fmt.Errorf("failed to set queue entry %d as completed: %w", queueID, err)
	}

	log.Info().Msgf("Queue entry %d marked as completed", queueID)
	return nil
}

// SetQueueFailed marks a queue entry as failed with an error message
func SetQueueFailed(db *sql.DB, queueID int, errorMsg string) error {
	query := `
		UPDATE build_queue
		SET status = $1, completed_at = $2, error_message = $3
		WHERE queue_id = $4;`

	_, err := db.Exec(query, StatusFailed, time.Now(), errorMsg, queueID)
	if err != nil {
		return fmt.Errorf("failed to set queue entry %d as failed: %w", queueID, err)
	}

	log.Error().Msgf("Queue entry %d marked as failed: %s", queueID, errorMsg)
	return nil
}

// CheckQueueStatus checks if a build is already queued or processing
func CheckQueueStatus(db *sql.DB, itemID string, buildType string, constraints models.EvaluationConstraints) (*QueueEntry, error) {
	tradersMap := constraintsToTraderMap(constraints)

	query := `
		SELECT 
			queue_id,
			item_id,
			build_type,
			jaeger_level,
			prapor_level,
			peacekeeper_level,
			mechanic_level,
			skier_level,
			priority,
			status,
			created_at,
			started_at,
			completed_at,
			error_message
		FROM build_queue
		WHERE 
			item_id = $1
			AND build_type = $2
			AND jaeger_level = $3
			AND prapor_level = $4
			AND peacekeeper_level = $5
			AND mechanic_level = $6
			AND skier_level = $7
			AND status IN ($8, $9)
		ORDER BY created_at DESC
		LIMIT 1;`

	row := db.QueryRow(
		query,
		itemID,
		buildType,
		tradersMap["Jaeger"],
		tradersMap["Prapor"],
		tradersMap["Peacekeeper"],
		tradersMap["Mechanic"],
		tradersMap["Skier"],
		StatusQueued,
		StatusProcessing,
	)

	var entry QueueEntry
	var jaegerLevel, praporLevel, peacekeeperLevel, mechanicLevel, skierLevel int

	err := row.Scan(
		&entry.QueueID,
		&entry.ItemID,
		&entry.BuildType,
		&jaegerLevel,
		&praporLevel,
		&peacekeeperLevel,
		&mechanicLevel,
		&skierLevel,
		&entry.Priority,
		&entry.Status,
		&entry.CreatedAt,
		&entry.StartedAt,
		&entry.CompletedAt,
		&entry.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not queued
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check queue status: %w", err)
	}

	// Convert trader levels to constraints
	tradersMap = map[string]int{
		"Jaeger":      jaegerLevel,
		"Prapor":      praporLevel,
		"Peacekeeper": peacekeeperLevel,
		"Mechanic":    mechanicLevel,
		"Skier":       skierLevel,
	}
	entry.Constraints = traderMapToConstraints(tradersMap)

	return &entry, nil
}
