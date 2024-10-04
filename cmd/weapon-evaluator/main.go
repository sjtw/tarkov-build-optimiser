package main

import (
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
)

func main() {
	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db")
	}

	log.Info().Msg("Fetching weapon IDs")
	weaponIds, err := models.GetAllWeaponIds(dbClient.Conn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get all weapon IDs")
	}

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))
	for i := 0; i < len(weaponIds); i++ {
		log.Info().Msgf("Building weapon %s", weaponIds[i])

		traderLevelVariations := evaluator.GenerateTraderLevelVariations(evaluator.TraderNames)

		for j := 0; j < len(traderLevelVariations); j++ {
			log.Info().Msgf("Trader level constraints: %v", traderLevelVariations[j])
			constraints := evaluator.EvaluationConstraints{
				TraderLevels: traderLevelVariations[j],
			}

			err := evaluator.GenerateOptimumWeaponBuilds(dbClient.Conn, weaponIds[i], constraints)
			if err != nil {
				log.Fatal().Err(err).Msgf("Failed to generate weapon builds for %s", weaponIds[i])
			}
		}

	}
	log.Info().Msg("Evaluator done.")
}
