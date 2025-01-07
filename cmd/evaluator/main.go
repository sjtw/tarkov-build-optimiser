package main

import (
	"database/sql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"runtime"
	"sync"
	"tarkov-build-optimiser/internal/candidate_tree"
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
			"5644bd2b4bdc2d3b4c8b4572",
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

	workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	dataService := candidate_tree.CreateDataService(dbClient.Conn)
	evaluate(weaponIds, dataService, workerCount, dbClient.Conn)

	log.Info().Msg("Evaluator done.")
}

type EvaluationResult struct {
	Task   Task
	Result *evaluator.Build
}

//type WeaponPossibilityResult struct {
//	Weapon *candidate_tree.CandidateTree
//	Id     string
//}

type Task struct {
	Weapon         *candidate_tree.CandidateTree
	EvaluationType string
}

//func processEvaluationTasks(tasks []Task, workerCount int) []EvaluationResult {
//	taskChan := make(chan Task, len(tasks))
//	resultsChan := make(chan EvaluationResult, len(tasks))
//	wg := sync.WaitGroup{}
//
//	for i := 0; i < workerCount; i++ {
//		wg.Add(1)
//
//		go func() {
//			defer wg.Done()
//
//			for task := range taskChan {
//				build := evaluator.FindBestBuild(task.Weapon, task.EvaluationType, map[string]bool{})
//				resultsChan <- EvaluationResult{
//					Task:   task,
//					Result: build,
//				}
//			}
//		}()
//	}
//
//	for _, task := range tasks {
//		taskChan <- task
//	}
//	close(taskChan)
//
//	wg.Wait()
//
//	results := make([]EvaluationResult, len(tasks))
//	close(resultsChan)
//	for result := range resultsChan {
//		results = append(results, result)
//	}
//
//	return results
//}

//func createEvaluationTasks(weaponCandidateTrees []WeaponPossibilityResult, evaluationTypes []string) []Task {
//	tasks := make([]Task, 0)
//
//	for i := 0; i < len(weaponCandidateTrees); i++ {
//		w := weaponCandidateTrees[i]
//		log.Debug().Msgf("Creating task variations for weapon %s", w.Id)
//
//		for k := 0; k < len(evaluationTypes); k++ {
//			task := Task{
//				Weapon:         w.Weapon,
//				EvaluationType: evaluationTypes[k],
//			}
//			tasks = append(tasks, task)
//		}
//	}
//
//	return tasks
//}

type Candidateinput struct {
	weaponID    string
	constraints models.EvaluationConstraints
}

func evaluate(weaponIds []string, dataProvider candidate_tree.TreeDataProvider, workerCount int, db *sql.DB) []EvaluationResult {
	inputChan := make(chan Candidateinput, len(weaponIds)*len(models.TraderNames))
	resultsChan := make(chan EvaluationResult, len(weaponIds)*len(models.TraderNames))
	wg := sync.WaitGroup{}

	traderLevels := evaluator.GenerateTraderLevelVariations(models.TraderNames)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for input := range inputChan {
				weapon, err := candidate_tree.CreateWeaponCandidateTree(input.weaponID, input.constraints, dataProvider)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create weapon tree for %s. Skipping", weaponIds[i])
					continue
				}

				weapon.SortAllowedItems("recoil-min")

				log.Info().Msgf("Generated weapon candidate tree for %s with constraints %v", weaponIds[i], input.constraints)

				task := Task{
					Weapon:         weapon,
					EvaluationType: "recoil",
				}
				build := evaluator.FindBestBuild(task.Weapon, task.EvaluationType, map[string]bool{})

				log.Info().Msgf("Evaluation complete - weapon %s with constraints %v", weaponIds[i], input.constraints)

				evaledWeapon, err := build.ToEvaluatedWeapon()
				if err != nil {
					log.Error().Err(err).Msgf("Failed to convert result to evaluated weapon for weapon %s with constraints %v", weaponIds[i], input.constraints)
					continue
				}

				evaluationResult := evaledWeapon.ToItemEvaluationResult()

				err = models.UpsertOptimumBuild(db, &evaluationResult, build.WeaponTree.Constraints)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to save build for weapon %s with constraints %v", weaponIds[i], input.constraints)
					continue
				}

				resultsChan <- EvaluationResult{
					Task:   task,
					Result: build,
				}
			}

		}()
	}

	for i := 0; i < len(weaponIds); i++ {
		for j := 0; j < len(traderLevels); j++ {
			constraints := models.EvaluationConstraints{
				TraderLevels:     traderLevels[j],
				IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
				IgnoredItemIDs:   []string{},
			}

			inputChan <- Candidateinput{
				weaponID:    weaponIds[i],
				constraints: constraints,
			}
		}
	}

	close(inputChan)
	wg.Wait()

	results := make([]EvaluationResult, 0)
	close(resultsChan)
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}
