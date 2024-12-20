package main

import (
	"github.com/rs/zerolog/log"
	"sync"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
)

type EvaluationResult struct {
	Task   evaluator.Task
	Result models.ItemEvaluationResult
	Ok     bool
	Error  error
}

type WeaponPossibilityResult struct {
	Weapon *evaluator.WeaponTree
	Id     string
	Ok     bool
}

func processEvaluationTasks(dataProvider evaluator.EvaluationDataProvider, tasks []evaluator.Task, workerCount int) []EvaluationResult {
	ev := evaluator.CreateEvaluator(dataProvider)
	taskChan := make(chan evaluator.Task, len(tasks))
	wg := sync.WaitGroup{}

	resultsChan := make(chan EvaluationResult, len(tasks))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range taskChan {
				result, err := ev.EvaluateTask(task)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to generate weapon builds for %s", task.Weapon.Item.ID)
					resultsChan <- EvaluationResult{
						Task:   task,
						Result: models.ItemEvaluationResult{},
						Ok:     false,
						Error:  err,
					}
				}

				resultsChan <- EvaluationResult{
					Task:   task,
					Result: result,
					Ok:     true,
					Error:  nil,
				}
			}
		}()
	}

	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	wg.Wait()

	results := make([]EvaluationResult, len(tasks))
	close(resultsChan)
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

func createEvaluationTasks(weaponCandidateTrees []WeaponPossibilityResult, evaluationTypes []string) []evaluator.Task {
	tasks := make([]evaluator.Task, 0)

	traderLevelVariations := evaluator.GenerateTraderLevelVariations(models.TraderNames)

	for i := 0; i < len(weaponCandidateTrees); i++ {
		w := weaponCandidateTrees[i]

		if w.Ok == false || w.Weapon == nil {
			log.Debug().Msgf("Skipping [%s] [%s] - weapon possibility result is invalid.", w.Id, w.Weapon.Item.Name)
			continue
		}

		log.Debug().Msgf("Creating task variations for weapon %s", w.Id)

		for j := 0; j < len(traderLevelVariations); j++ {
			constraints := models.EvaluationConstraints{
				TraderLevels: traderLevelVariations[j],
			}

			for k := 0; k < len(evaluationTypes); k++ {
				task := evaluator.Task{
					Constraints:    constraints,
					Weapon:         *w.Weapon,
					EvaluationType: evaluationTypes[k],
				}
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

func createWeaponCandidateTrees(weaponIds []string, dataProvider evaluator.TreeDataProvider) []WeaponPossibilityResult {
	results := make([]WeaponPossibilityResult, 0)

	for i := 0; i < len(weaponIds); i++ {
		weapon, err := evaluator.ConstructWeaponTree(weaponIds[i], dataProvider)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create weapon tree for %s. Skipping", weaponIds[i])
			continue
		}

		results = append(results, WeaponPossibilityResult{
			Weapon: weapon,
			Id:     weaponIds[i],
			Ok:     err == nil,
		})
	}

	return results
}
