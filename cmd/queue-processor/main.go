package main

import (
	"database/sql"
	"runtime"
	"sync"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/queue"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	flags := cli.GetFlags()

	// Set log level based on CLI flag
	cli.SetLogLevel(flags.LogLevel)
	environment, err := env.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get environment variables")
	}

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db")
	}

	// Create cache (choose between memory or database)
	var cache evaluator.Cache
	if flags.UseDatabaseCache {
		cache = evaluator.NewDatabaseCache(dbClient.Conn)
	} else {
		cache = evaluator.NewMemoryCache()
	}

	workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor
	log.Info().Msgf("Starting queue processor with %d workers", workerCount)

	dataService := candidate_tree.CreateDataService(dbClient.Conn)

	// Start continuous processing
	processQueue(dbClient.Conn, dataService, workerCount, cache)
}

// QueueJob represents a job to be processed
type QueueJob struct {
	Entry *queue.QueueEntry
}

// processQueue continuously polls for and processes queued builds
func processQueue(db *sql.DB, dataProvider candidate_tree.TreeDataProvider, workerCount int, cache evaluator.Cache) {
	inputChan := make(chan QueueJob, workerCount*2)
	wg := sync.WaitGroup{}

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(db, dataProvider, cache, inputChan, &wg)
	}

	// Polling loop
	log.Info().Msg("Queue processor started, polling for jobs...")
	pollInterval := 2 * time.Second

	for {
		// Fetch next job from queue
		entry, err := queue.GetNextQueuedBuild(db)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch next queued build")
			time.Sleep(pollInterval)
			continue
		}

		if entry == nil {
			// No jobs available, wait before polling again
			log.Debug().Msg("No jobs in queue, waiting...")
			time.Sleep(pollInterval)
			continue
		}

		log.Info().Msgf("Found queued job %d for item %s", entry.QueueID, entry.ItemID)

		// Send job to worker pool
		inputChan <- QueueJob{Entry: entry}
	}
}

// worker processes queue jobs
func worker(db *sql.DB, dataProvider candidate_tree.TreeDataProvider, cache evaluator.Cache, inputChan chan QueueJob, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range inputChan {
		entry := job.Entry
		log.Info().Msgf("Processing queue entry %d for item %s", entry.QueueID, entry.ItemID)

		// Mark as processing
		err := queue.SetQueueProcessing(db, entry.QueueID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to mark queue entry %d as processing", entry.QueueID)
			continue
		}

		// Check if build already exists in optimum_builds
		existingBuild, err := models.GetOptimumBuildByConstraints(db, entry.ItemID, entry.BuildType, entry.Constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to check for existing build for queue entry %d", entry.QueueID)
			_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
			continue
		}

		// If build exists and is completed, just mark queue as completed
		if existingBuild != nil && existingBuild.Status == models.EvaluationCompleted.ToString() {
			log.Info().Msgf("Build already exists for queue entry %d, marking as completed", entry.QueueID)
			err = queue.SetQueueCompleted(db, entry.QueueID)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to mark queue entry %d as completed", entry.QueueID)
			}
			continue
		}

		// Create or get the build ID in optimum_builds
		var buildID int
		if existingBuild != nil {
			buildID = existingBuild.BuildID
		} else {
			buildID, err = models.CreatePendingOptimumBuild(db, entry.ItemID, entry.BuildType, entry.Constraints)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to create optimum build entry for queue entry %d", entry.QueueID)
				_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
				continue
			}
		}

		// Mark optimum_builds as in progress
		err = models.SetBuildInProgress(db, buildID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to update optimum build status to in progress for queue entry %d", entry.QueueID)
			_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
			continue
		}

		// Create weapon candidate tree
		weapon, err := candidate_tree.CreateWeaponCandidateTree(entry.ItemID, entry.BuildType, entry.Constraints, dataProvider)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create weapon tree for queue entry %d", entry.QueueID)
			_ = models.SetBuildFailed(db, buildID)
			_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
			continue
		}

		weapon.SortAllowedItems("recoil-min")

		log.Info().Msgf("Generated weapon candidate tree for queue entry %d", entry.QueueID)

		// Find best build
		build := evaluator.FindBestBuild(weapon, entry.BuildType, map[string]bool{}, cache)

		log.Info().Msgf("Evaluation complete for queue entry %d", entry.QueueID)

		// Convert to evaluated weapon
		evaledWeapon, err := build.ToEvaluatedWeapon()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to convert result to evaluated weapon for queue entry %d", entry.QueueID)
			_ = models.SetBuildFailed(db, buildID)
			_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
			continue
		}

		evaluationResult := evaledWeapon.ToItemEvaluationResult()

		// Save build to optimum_builds
		err = models.SetBuildCompleted(db, buildID, &evaluationResult)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to save build for queue entry %d", entry.QueueID)
			_ = models.SetBuildFailed(db, buildID)
			_ = queue.SetQueueFailed(db, entry.QueueID, err.Error())
			continue
		}

		log.Info().Msgf("Saved build for queue entry %d", entry.QueueID)

		// Mark queue entry as completed
		err = queue.SetQueueCompleted(db, entry.QueueID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to mark queue entry %d as completed", entry.QueueID)
			continue
		}

		log.Info().Msgf("Queue entry %d completed successfully", entry.QueueID)
	}
}
