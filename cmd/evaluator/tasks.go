package main

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
)

type EvaluationResult struct {
	Task  Task
	Ok    bool
	Error error
}

func processEvaluationTasks(db *sql.DB, tasks []Task, workerCount int) []EvaluationResult {
	taskCount := len(tasks)

	taskChan := make(chan Task, len(tasks))
	resultChan := make(chan EvaluationResult, taskCount)
	doneChan := make(chan struct{})

	for i := 0; i < workerCount; i++ {
		log.Debug().Msgf("Creating worker %d", i)
		go calculateBuilds(db, taskChan, resultChan, doneChan, i)
	}

	log.Debug().Msgf("Queuing tasks")
	for i := 0; i < len(tasks); i++ {
		taskChan <- tasks[i]
	}
	log.Debug().Msgf("Queued %d tasks", len(tasks))

	close(taskChan)

	go func() {
		for i := 0; i < workerCount; i++ {
			<-doneChan
		}
		close(resultChan)
		close(doneChan)
	}()

	log.Debug().Msg("Collecting results")
	count := 0
	results := make([]EvaluationResult, 0, taskCount)
	for result := range resultChan {
		if count%1000 == 0 {
			log.Info().Msgf("Generated %d weapon builds, %d remaining.", count, taskCount-count)
		}

		results = append(results, result)
		count++
	}

	return results
}

type Task struct {
	Constraints models.EvaluationConstraints
	Weapon      evaluator.Item
}

func calculateBuilds(db *sql.DB, tasks <-chan Task, resultChan chan<- EvaluationResult, doneChan chan<- struct{}, workerId int) {
	for task := range tasks {
		log.Debug().Msgf("[Worker %d] Processing evaluation task: %v", workerId, task)
		err := evaluator.GenerateOptimumWeaponBuilds(db, task.Weapon, task.Constraints)
		if err != nil {
			log.Error().Err(err).Msgf("[Worker %d] Failed to generate weapon builds for %s", workerId, task.Weapon.ID)
			resultChan <- EvaluationResult{
				Task:  task,
				Ok:    false,
				Error: err,
			}
			continue
		}

		log.Debug().Msgf("[Worker %d] Finished possibility tree for %s, %v", workerId, task.Weapon.ID, task.Constraints)

		resultChan <- EvaluationResult{
			Task:  task,
			Ok:    true,
			Error: nil,
		}
	}

	doneChan <- struct{}{}
}

func createEvaluationTasks(weaponPossibilities []WeaponPossibilityResult) []Task {
	tasks := make([]Task, 0)

	traderLevelVariations := evaluator.GenerateTraderLevelVariations(models.TraderNames)
	//testLevels := []models.TraderLevel{
	//	{Name: "Prapor", Level: 4},
	//	{Name: "Peacekeeper", Level: 4},
	//	{Name: "Jaeger", Level: 4},
	//	{Name: "Mechanic", Level: 4},
	//	{Name: "Skier", Level: 4},
	//}
	//traderLevelVariations := [][]models.TraderLevel{testLevels}

	evaluatedWeapons := make([]*evaluator.Item, 0)
	for i := 0; i < len(weaponPossibilities); i++ {
		w := weaponPossibilities[i]

		if w.Ok == false || w.Item == nil {
			log.Debug().Msgf("Skipping %v - weapon possibility result is invalid.", w.Id)
			continue
		}

		log.Debug().Msgf("Creating task variations for weapon %s", w.Id)

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
