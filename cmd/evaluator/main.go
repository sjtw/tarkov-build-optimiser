package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"runtime"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	flags := cli.GetFlags()
	environment, err := env.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get environment variables")
	}

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db")
	}

	if flags.PurgeOptimumBuilds {
		log.Info().Msg("Purging optimum builds.")
		err = models.PurgeOptimumBuilds(dbClient.Conn)
		if err != nil {
			log.Error().Err(err).Msg("Failed to purge all models.")
		}
		log.Info().Msg("Models purged.")
	}

	var weaponIds []string
	if flags.TestRun {
		log.Info().Msg("Using test weapon IDs")
		weaponIds = []string{
			"5447a9cd4bdc2dbd208b4567",
			//"5448bd6b4bdc2dfc2f8b4569",
			//"54491c4f4bdc2db1078b4568",
		}
	} else {
		log.Info().Msg("Fetching weapon IDs")
		weaponIds, err = models.GetAllWeaponIds(dbClient.Conn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get all weapon IDs")
		}
	}

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	dataService := evaluator.CreateDataService(dbClient.Conn)
	candidateTree := createWeaponCandidateTree(weaponIds, dataService)
	log.Info().Msgf("Generated %d weapon possibility trees.", len(candidateTree))

	for i := 0; i < len(candidateTree); i++ {
		if len(candidateTree[i].Item.ConflictingItems) > 0 {
			log.Info().Msgf("conflicted item at index %d", i)
		}
	}

	log.Info().Msg("Creating evaluation tasks.")
	tasks := createEvaluationTasks(candidateTree, []string{"recoil", "ergonomics"})
	log.Info().Msgf("Scheduled %d evaluation tasks.", len(tasks))

	log.Info().Msg("Processing evaluation tasks.")

	workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor
	processEvaluationTasks(dataService, tasks, workerCount)
	log.Info().Msg("Evaluator done.")
}
