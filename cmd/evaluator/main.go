package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/evaluator"
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

	var weaponIds []string
	if env.TestRun {
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
	weaponPossibilities := createWeaponPossibilities(weaponIds, dataService)
	log.Info().Msgf("Generated %d weapon possibility trees.", len(weaponPossibilities))

	log.Info().Msg("Creating evaluation tasks.")
	tasks := createEvaluationTasks(weaponPossibilities, []string{"recoil", "ergonomics"})
	log.Info().Msgf("Scheduled %d evaluation tasks.", len(tasks))

	log.Info().Msg("Processing evaluation tasks.")
	processEvaluationTasks(dataService, tasks)
	log.Info().Msg("Evaluator done.")
}
