package main

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"sync"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
)

func processEvaluationTasks(db *sql.DB, tasks []Task, workerCount int) {
	var wg sync.WaitGroup
	taskChannel := make(chan Task, len(tasks))

	for i := 0; i < workerCount; i++ {
		log.Info().Msgf("Creating worker %d", i)
		wg.Add(1)
		go calculateBuilds(db, taskChannel, &wg, i)
	}

	log.Info().Msgf("Queuing tasks")
	for i := 0; i < len(tasks); i++ {
		taskChannel <- tasks[i]
	}
	log.Info().Msgf("Queued %d tasks", len(taskChannel))

	close(taskChannel)
	wg.Wait()
}

type Task struct {
	Constraints models.EvaluationConstraints
	Weapon      evaluator.Item
}

func calculateBuilds(db *sql.DB, tasks <-chan Task, wg *sync.WaitGroup, workerId int) {
	defer wg.Done()

	for task := range tasks {
		log.Info().Msgf("[Worker %d] Processing evaluation task: %v", workerId, task)
		err := evaluator.GenerateOptimumWeaponBuilds(db, task.Weapon, task.Constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to generate weapon builds for %s", task.Weapon.ID)
		}
	}
}

func createEvaluationTasks(weaponPossibilities []WeaponPossibilityResult) []Task {
	tasks := make([]Task, 0)

	traderLevelVariations := evaluator.GenerateTraderLevelVariations(models.TraderNames)

	evaluatedWeapons := make([]*evaluator.Item, 0)
	for i := 0; i < len(weaponPossibilities); i++ {
		w := weaponPossibilities[i]

		if w.Ok == false || w.Item == nil {
			log.Info().Msgf("Skipping %v - weapon possibility result is invalid.", w.Id)
			continue
		}

		log.Info().Msgf("Creating task variations for weapon %s", w.Id)

		evaluatedWeapons = append(evaluatedWeapons, weaponPossibilities[i].Item)

		for j := 0; j < len(traderLevelVariations); j++ {
			constraints := models.EvaluationConstraints{
				TraderLevels: traderLevelVariations[j],
			}

			task := Task{
				Constraints: constraints,
				Weapon:      *w.Item,
			}
			tasks = append(tasks, task)
		}
	}

	return tasks
}
