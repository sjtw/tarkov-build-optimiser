package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"runtime"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/models"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	env := cli.GetFlags()

	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db")
	}

	if env.PurgeOptimumBuilds {
		log.Info().Msg("Purging optimum builds.")
		err = models.PurgeOptimumBuilds(dbClient.Conn)
		if err != nil {
			log.Error().Err(err).Msg("Failed to purge all models.")
		}
		log.Info().Msg("Models purged.")
	}

	log.Debug().Msg("Fetching weapon IDs")
	weaponIds, err := models.GetAllWeaponIds(dbClient.Conn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get all weapon IDs")
	}

	//weaponIds := []string{"5fc3e272f8b6a877a729eac5"}

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	maxWorkerCount := runtime.NumCPU() * 100
	var workerCount int
	if len(weaponIds) > maxWorkerCount {
		workerCount = maxWorkerCount
	} else {
		workerCount = len(weaponIds)
	}

	log.Info().Msgf("Worker pool size: %d", workerCount)
	weaponPossibilities := generateWeaponPossibilities(dbClient.Conn, weaponIds, workerCount)
	log.Info().Msgf("Generated %d weapon possibility trees.", len(weaponPossibilities))
	log.Info().Msg("Creating evaluation tasks.")

	tasks := createEvaluationTasks(weaponPossibilities)
	log.Info().Msgf("Scheduled %d evaluation tasks.", len(tasks))
	if len(tasks) > maxWorkerCount {
		workerCount = maxWorkerCount
	} else {
		workerCount = len(tasks)
	}
	log.Info().Msgf("Worker pool size: %d", workerCount)

	log.Info().Msg("Processing evaluation tasks.")
	processEvaluationTasks(dbClient.Conn, tasks, workerCount)
	log.Info().Msg("Evaluator done.")
}