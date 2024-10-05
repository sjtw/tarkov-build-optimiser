package main

import (
	"github.com/rs/zerolog/log"
	"runtime"
	"tarkov-build-optimiser/internal/db"
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

	//weaponIds := []string{"5bf3e03b0db834001d2c4a9c"}

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	workerCount := runtime.NumCPU() * 10
	weaponPossibilities := generateWeaponPossibilities(dbClient.Conn, weaponIds, workerCount)

	tasks := createEvaluationTasks(weaponPossibilities)
	log.Info().Msgf("Scheduled %d evaluation tasks", len(tasks))

	processEvaluationTasks(dbClient.Conn, tasks, workerCount)

	log.Info().Msg("Evaluator done.")
}
