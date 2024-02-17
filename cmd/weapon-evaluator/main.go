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
		err := evaluator.GenerateOptimumWeaponBuilds(dbClient.Conn, weaponIds[i])
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to generate weapon builds for %s", weaponIds[i])
		}
	}
	log.Info().Msg("Evaluator done.")
}
