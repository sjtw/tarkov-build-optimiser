package main

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"runtime"
	"sync"
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

	tasks := make([]Task, 0)

	for i := 0; i < len(weaponIds); i++ {
		log.Info().Msgf("Creating task variations for weapon %s", weaponIds[i])

		traderLevelVariations := evaluator.GenerateTraderLevelVariations(models.TraderNames)

		for j := 0; j < len(traderLevelVariations); j++ {
			constraints := models.EvaluationConstraints{
				TraderLevels: traderLevelVariations[j],
			}

			task := Task{
				Constraints: constraints,
				WeaponID:    weaponIds[i],
			}
			log.Info().Msgf("Scheduling evaluation task: %v", task)
			tasks = append(tasks, task)
		}

	}

	log.Info().Msgf("Scheduled %d evaluation tasks", len(tasks))

	var wg sync.WaitGroup
	workerCount := runtime.NumCPU() * 3
	taskChannel := make(chan Task, len(tasks))

	for i := 0; i < workerCount; i++ {
		log.Info().Msgf("Creating worker %d", i)
		wg.Add(1)
		go worker(dbClient.Conn, taskChannel, &wg)
	}

	log.Info().Msgf("Queuing tasks")
	for i := 0; i < len(tasks); i++ {
		taskChannel <- tasks[i]
	}
	log.Info().Msgf("Queued %d tasks", len(taskChannel))

	close(taskChannel)
	wg.Wait()
	log.Info().Msg("Evaluator done.")
}

type Task struct {
	Constraints models.EvaluationConstraints
	WeaponID    string
}

func worker(db *sql.DB, tasks <-chan Task, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range tasks {
		log.Info().Msgf("Processing evaluation task: %v", task)
		err := evaluator.GenerateOptimumWeaponBuilds(db, task.WeaponID, task.Constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to generate weapon builds for %s", task.WeaponID)
		}
	}
}
