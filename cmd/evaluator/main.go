package main

import (
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/queue"

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

	log.Info().Msg("Batch evaluator starting - queueing builds for calculation")

	log.Info().Msg("Purging optimum builds.")
	err = models.PurgeOptimumBuilds(dbClient.Conn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to purge optimum builds.")
		return
	}

	log.Info().Msg("Models purged.")

	traderLevels := evaluator.GenerateTraderLevelVariations(models.TraderNames)
	traderLevels = evaluator.SortTraderLevelsByAvg(traderLevels)

	var weaponIds []string
	if flags.TestRun {
		log.Info().Msg("Using test weapon IDs")
		weaponIds = []string{
			"5447a9cd4bdc2dbd208b4567", // Colt M4A1 5.56x45 assault rifle
			"6895bb82c4519957df062f82", // Radian Weapons Model 1 FA 5.56x45 assault rifle
		}
		traderLevels = [][]models.TraderLevel{
			{
				{Name: "Jaeger", Level: 1},
				{Name: "Prapor", Level: 1},
				{Name: "Skier", Level: 1},
				{Name: "Peacekeeper", Level: 1},
				{Name: "Mechanic", Level: 1},
			},
			{
				{Name: "Jaeger", Level: 2},
				{Name: "Prapor", Level: 2},
				{Name: "Skier", Level: 2},
				{Name: "Peacekeeper", Level: 2},
				{Name: "Mechanic", Level: 2},
			},
			{
				{Name: "Jaeger", Level: 3},
				{Name: "Prapor", Level: 3},
				{Name: "Skier", Level: 3},
				{Name: "Peacekeeper", Level: 3},
				{Name: "Mechanic", Level: 3},
			},
			{
				{Name: "Jaeger", Level: 4},
				{Name: "Prapor", Level: 4},
				{Name: "Skier", Level: 4},
				{Name: "Peacekeeper", Level: 4},
				{Name: "Mechanic", Level: 4},
			},
		}
	} else {
		log.Info().Msg("Fetching weapon IDs")
		weaponIds, err = models.GetAllWeaponIds(dbClient.Conn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get all weapon IDs")
		}
	}

	log.Info().Msgf("Queueing %d weapons for evaluation", len(weaponIds))

	queueBuilds(weaponIds, traderLevels, dbClient)

	log.Info().Msg("Batch evaluator done - all builds queued.")
}

// queueBuilds queues all weapon/trader level combinations for evaluation
func queueBuilds(weaponIds []string, traderLevels [][]models.TraderLevel, dbConn *db.Database) {
	queuedCount := 0
	errorCount := 0

	for i := 0; i < len(weaponIds); i++ {
		for j := 0; j < len(traderLevels); j++ {
			constraints := models.EvaluationConstraints{
				TraderLevels:     traderLevels[j],
				IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
				IgnoredItemIDs:   []string{},
			}

			log.Debug().Msgf("Queueing weapon %s with constraints %v", weaponIds[i], traderLevels[j])

			_, err := queue.CreateQueueEntry(dbConn.Conn, weaponIds[i], "recoil", constraints, queue.PriorityBatch)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to queue build for weapon %s", weaponIds[i])
				errorCount++
				continue
			}

			queuedCount++
		}
	}

	log.Info().Msgf("Successfully queued %d builds (%d errors)", queuedCount, errorCount)
}
