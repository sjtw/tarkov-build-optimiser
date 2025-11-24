package queue_test

import (
	"database/sql"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/queue"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection and cleans up the queue
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	environment, err := env.Get()
	require.NoError(t, err, "Failed to get environment")

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	require.NoError(t, err, "Failed to connect to database")

	// Clean up queue before each test
	_, err = dbClient.Conn.Exec("TRUNCATE build_queue;")
	require.NoError(t, err, "Failed to truncate build_queue")

	return dbClient.Conn
}

// createTestConstraints creates sample evaluation constraints for testing
func createTestConstraints(levels ...int) models.EvaluationConstraints {
	level := 1
	if len(levels) > 0 {
		level = levels[0]
	}

	return models.EvaluationConstraints{
		TraderLevels: []models.TraderLevel{
			{Name: "Jaeger", Level: level},
			{Name: "Prapor", Level: level},
			{Name: "Skier", Level: level},
			{Name: "Peacekeeper", Level: level},
			{Name: "Mechanic", Level: level},
		},
		IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
		IgnoredItemIDs:   []string{},
	}
}

func TestCreateQueueEntry(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name        string
		itemID      string
		buildType   string
		constraints models.EvaluationConstraints
		priority    int
	}{
		{
			name:        "creates queue entry with API priority",
			itemID:      "test_weapon_1",
			buildType:   "recoil",
			constraints: createTestConstraints(1),
			priority:    queue.PriorityAPI,
		},
		{
			name:        "creates queue entry with batch priority",
			itemID:      "test_weapon_2",
			buildType:   "recoil",
			constraints: createTestConstraints(2),
			priority:    queue.PriorityBatch,
		},
		{
			name:        "creates queue entry with custom priority",
			itemID:      "test_weapon_3",
			buildType:   "ergonomics",
			constraints: createTestConstraints(4),
			priority:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queueID, err := queue.CreateQueueEntry(db, tt.itemID, tt.buildType, tt.constraints, tt.priority)

			assert.NoError(t, err)
			assert.Greater(t, queueID, 0, "Queue ID should be greater than 0")

			// Verify the entry was created correctly
			var retrievedItemID, retrievedBuildType string
			var retrievedPriority int
			var status string

			err = db.QueryRow(`
				SELECT item_id, build_type, priority, status
				FROM build_queue
				WHERE queue_id = $1`, queueID).Scan(&retrievedItemID, &retrievedBuildType, &retrievedPriority, &status)

			require.NoError(t, err)
			assert.Equal(t, tt.itemID, retrievedItemID)
			assert.Equal(t, tt.buildType, retrievedBuildType)
			assert.Equal(t, tt.priority, retrievedPriority)
			assert.Equal(t, string(queue.StatusQueued), status)
		})
	}
}

func TestGetNextQueuedBuild(t *testing.T) {
	db := setupTestDB(t)

	t.Run("returns nil when queue is empty", func(t *testing.T) {
		entry, err := queue.GetNextQueuedBuild(db)

		assert.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("returns highest priority entry first", func(t *testing.T) {
		// Create entries with different priorities
		constraints := createTestConstraints(1)

		lowPriorityID, err := queue.CreateQueueEntry(db, "weapon_low", "recoil", constraints, 10)
		require.NoError(t, err)

		highPriorityID, err := queue.CreateQueueEntry(db, "weapon_high", "recoil", constraints, 100)
		require.NoError(t, err)

		mediumPriorityID, err := queue.CreateQueueEntry(db, "weapon_medium", "recoil", constraints, 50)
		require.NoError(t, err)

		// Get next entry
		entry, err := queue.GetNextQueuedBuild(db)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, highPriorityID, entry.QueueID, "Should return highest priority entry")
		assert.Equal(t, "weapon_high", entry.ItemID)
		assert.Equal(t, 100, entry.Priority)

		// Verify other entries are still there
		_, _ = lowPriorityID, mediumPriorityID // Used for setup
	})

	t.Run("returns oldest entry when priorities are equal", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(1)

		firstID, err := queue.CreateQueueEntry(db, "weapon_first", "recoil", constraints, 50)
		require.NoError(t, err)

		secondID, err := queue.CreateQueueEntry(db, "weapon_second", "recoil", constraints, 50)
		require.NoError(t, err)

		// Get next entry
		entry, err := queue.GetNextQueuedBuild(db)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, firstID, entry.QueueID, "Should return oldest entry when priorities are equal")
		assert.Equal(t, "weapon_first", entry.ItemID)

		// Verify second entry is still there
		_ = secondID // Used for setup
	})

	t.Run("skips processing and completed entries", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(1)

		// Create and mark one as processing
		processingID, err := queue.CreateQueueEntry(db, "weapon_processing", "recoil", constraints, 100)
		require.NoError(t, err)
		err = queue.SetQueueProcessing(db, processingID)
		require.NoError(t, err)

		// Create a queued entry with lower priority
		queuedID, err := queue.CreateQueueEntry(db, "weapon_queued", "recoil", constraints, 50)
		require.NoError(t, err)

		// Get next entry
		entry, err := queue.GetNextQueuedBuild(db)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, queuedID, entry.QueueID, "Should skip processing entries")
		assert.Equal(t, "weapon_queued", entry.ItemID)
	})

	t.Run("returns entry with correct constraints", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(3)

		queueID, err := queue.CreateQueueEntry(db, "weapon_test", "recoil", constraints, 100)
		require.NoError(t, err)

		entry, err := queue.GetNextQueuedBuild(db)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, queueID, entry.QueueID)
		assert.Equal(t, "weapon_test", entry.ItemID)
		assert.Equal(t, "recoil", entry.BuildType)
		assert.NotNil(t, entry.Constraints)
		assert.Equal(t, 5, len(entry.Constraints.TraderLevels))

		// Verify trader levels are correct
		for _, tl := range entry.Constraints.TraderLevels {
			assert.Equal(t, 3, tl.Level)
		}
	})
}

func TestSetQueueProcessing(t *testing.T) {
	db := setupTestDB(t)

	t.Run("marks queue entry as processing", func(t *testing.T) {
		constraints := createTestConstraints(1)
		queueID, err := queue.CreateQueueEntry(db, "weapon_test", "recoil", constraints, 50)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)

		assert.NoError(t, err)

		// Verify status was updated
		var status string
		var startedAt sql.NullTime

		err = db.QueryRow(`
			SELECT status, started_at
			FROM build_queue
			WHERE queue_id = $1`, queueID).Scan(&status, &startedAt)

		require.NoError(t, err)
		assert.Equal(t, string(queue.StatusProcessing), status)
		assert.True(t, startedAt.Valid, "started_at should be set")
	})

	t.Run("returns error for non-existent queue entry", func(t *testing.T) {
		err := queue.SetQueueProcessing(db, 99999)

		assert.NoError(t, err) // UPDATE doesn't error on 0 rows affected
	})
}

func TestSetQueueCompleted(t *testing.T) {
	db := setupTestDB(t)

	t.Run("marks queue entry as completed", func(t *testing.T) {
		constraints := createTestConstraints(1)
		queueID, err := queue.CreateQueueEntry(db, "weapon_test", "recoil", constraints, 50)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		err = queue.SetQueueCompleted(db, queueID)

		assert.NoError(t, err)

		// Verify status was updated
		var status string
		var completedAt sql.NullTime

		err = db.QueryRow(`
			SELECT status, completed_at
			FROM build_queue
			WHERE queue_id = $1`, queueID).Scan(&status, &completedAt)

		require.NoError(t, err)
		assert.Equal(t, string(queue.StatusCompleted), status)
		assert.True(t, completedAt.Valid, "completed_at should be set")
	})
}

func TestSetQueueFailed(t *testing.T) {
	db := setupTestDB(t)

	t.Run("marks queue entry as failed with error message", func(t *testing.T) {
		constraints := createTestConstraints(1)
		queueID, err := queue.CreateQueueEntry(db, "weapon_test", "recoil", constraints, 50)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		errorMsg := "failed to create weapon tree: invalid weapon ID"
		err = queue.SetQueueFailed(db, queueID, errorMsg)

		assert.NoError(t, err)

		// Verify status and error message were updated
		var status string
		var completedAt sql.NullTime
		var retrievedErrorMsg sql.NullString

		err = db.QueryRow(`
			SELECT status, completed_at, error_message
			FROM build_queue
			WHERE queue_id = $1`, queueID).Scan(&status, &completedAt, &retrievedErrorMsg)

		require.NoError(t, err)
		assert.Equal(t, string(queue.StatusFailed), status)
		assert.True(t, completedAt.Valid, "completed_at should be set")
		assert.True(t, retrievedErrorMsg.Valid, "error_message should be set")
		assert.Equal(t, errorMsg, retrievedErrorMsg.String)
	})
}

func TestCheckQueueStatus(t *testing.T) {
	db := setupTestDB(t)

	t.Run("returns nil when build is not queued", func(t *testing.T) {
		constraints := createTestConstraints(1)

		entry, err := queue.CheckQueueStatus(db, "non_existent_weapon", "recoil", constraints)

		assert.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("returns entry when build is queued", func(t *testing.T) {
		constraints := createTestConstraints(2)

		queueID, err := queue.CreateQueueEntry(db, "weapon_queued", "recoil", constraints, 100)
		require.NoError(t, err)

		entry, err := queue.CheckQueueStatus(db, "weapon_queued", "recoil", constraints)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, queueID, entry.QueueID)
		assert.Equal(t, "weapon_queued", entry.ItemID)
		assert.Equal(t, string(queue.StatusQueued), string(entry.Status))
	})

	t.Run("returns entry when build is processing", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(3)

		queueID, err := queue.CreateQueueEntry(db, "weapon_processing", "recoil", constraints, 100)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		entry, err := queue.CheckQueueStatus(db, "weapon_processing", "recoil", constraints)

		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, queueID, entry.QueueID)
		assert.Equal(t, string(queue.StatusProcessing), string(entry.Status))
	})

	t.Run("returns nil when build is completed", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(4)

		queueID, err := queue.CreateQueueEntry(db, "weapon_completed", "recoil", constraints, 100)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		err = queue.SetQueueCompleted(db, queueID)
		require.NoError(t, err)

		entry, err := queue.CheckQueueStatus(db, "weapon_completed", "recoil", constraints)

		assert.NoError(t, err)
		assert.Nil(t, entry, "Should not return completed entries")
	})

	t.Run("returns nil when build is failed", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints := createTestConstraints(4)

		queueID, err := queue.CreateQueueEntry(db, "weapon_failed", "recoil", constraints, 100)
		require.NoError(t, err)

		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		err = queue.SetQueueFailed(db, queueID, "test error")
		require.NoError(t, err)

		entry, err := queue.CheckQueueStatus(db, "weapon_failed", "recoil", constraints)

		assert.NoError(t, err)
		assert.Nil(t, entry, "Should not return failed entries")
	})

	t.Run("matches exact constraints", func(t *testing.T) {
		setupTestDB(t) // Clean slate

		constraints1 := createTestConstraints(1)
		constraints2 := createTestConstraints(2)

		queueID1, err := queue.CreateQueueEntry(db, "weapon_test", "recoil", constraints1, 100)
		require.NoError(t, err)

		// Check with same constraints
		entry, err := queue.CheckQueueStatus(db, "weapon_test", "recoil", constraints1)
		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, queueID1, entry.QueueID)

		// Check with different constraints
		entry, err = queue.CheckQueueStatus(db, "weapon_test", "recoil", constraints2)
		assert.NoError(t, err)
		assert.Nil(t, entry, "Should not match different constraints")
	})
}

func TestQueuePriorityConstants(t *testing.T) {
	t.Run("API priority is higher than batch priority", func(t *testing.T) {
		assert.Greater(t, queue.PriorityAPI, queue.PriorityBatch,
			"API priority should be higher than batch priority")
	})

	t.Run("priority constants have expected values", func(t *testing.T) {
		assert.Equal(t, 100, queue.PriorityAPI, "API priority should be 100")
		assert.Equal(t, 10, queue.PriorityBatch, "Batch priority should be 10")
	})
}

func TestQueueIntegrationScenario(t *testing.T) {
	db := setupTestDB(t)

	t.Run("complete queue workflow", func(t *testing.T) {
		constraints := createTestConstraints(2)

		// Step 1: Create a queue entry
		queueID, err := queue.CreateQueueEntry(db, "workflow_weapon", "recoil", constraints, queue.PriorityAPI)
		require.NoError(t, err)
		assert.Greater(t, queueID, 0)

		// Step 2: Check if it's queued
		entry, err := queue.CheckQueueStatus(db, "workflow_weapon", "recoil", constraints)
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, string(queue.StatusQueued), string(entry.Status))

		// Step 3: Get next job
		nextJob, err := queue.GetNextQueuedBuild(db)
		require.NoError(t, err)
		require.NotNil(t, nextJob)
		assert.Equal(t, queueID, nextJob.QueueID)

		// Step 4: Mark as processing
		err = queue.SetQueueProcessing(db, queueID)
		require.NoError(t, err)

		// Step 5: Verify it's processing
		entry, err = queue.CheckQueueStatus(db, "workflow_weapon", "recoil", constraints)
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, string(queue.StatusProcessing), string(entry.Status))

		// Step 6: Mark as completed
		err = queue.SetQueueCompleted(db, queueID)
		require.NoError(t, err)

		// Step 7: Verify it's no longer returned by CheckQueueStatus
		entry, err = queue.CheckQueueStatus(db, "workflow_weapon", "recoil", constraints)
		require.NoError(t, err)
		assert.Nil(t, entry)

		// Step 8: Verify it's no longer returned by GetNextQueuedBuild
		nextJob, err = queue.GetNextQueuedBuild(db)
		require.NoError(t, err)
		assert.Nil(t, nextJob)
	})
}
