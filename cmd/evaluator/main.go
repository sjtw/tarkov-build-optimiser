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

	log.Info().Msg("Creating evaluator status entry")

	log.Info().Msg("Purging optimum builds.")
	err = models.PurgeOptimumBuilds(dbClient.Conn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to purge optimum builds.")
		return
	}

	log.Info().Msg("Models purged.")

	traderLevels := evaluator.GenerateTraderLevelVariations(models.TraderNames)
	traderLevels = evaluator.SortTraderLevelsByMax(traderLevels)

	var weaponIds []string
	if flags.TestRun {
		log.Info().Msg("Using test weapon IDs")
		weaponIds = []string{
			//"54491c4f4bdc2db1078b4568",
			//"5448bd6b4bdc2dfc2f8b4569",
			"5447a9cd4bdc2dbd208b4567",
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

	workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	dataService := candidate_tree.CreateDataService(dbClient.Conn)
	evaluate(weaponIds, dataService, workerCount, traderLevels, dbClient.Conn)

	log.Info().Msg("Evaluator done.")
}

type EvaluationResult struct {
	Result         *evaluator.Build
	Weapon         *candidate_tree.CandidateTree
	EvaluationType string
	BuildID        int
}

type Candidateinput struct {
	weaponID    string
	constraints models.EvaluationConstraints
	BuildID     int
}

func evaluate(weaponIds []string, dataProvider candidate_tree.TreeDataProvider, workerCount int, traderLevels [][]models.TraderLevel, db *sql.DB) []EvaluationResult {
	inputChan := make(chan Candidateinput, workerCount*2)
	resultsChan := make(chan EvaluationResult, len(weaponIds)*len(traderLevels))
	wg := sync.WaitGroup{}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for input := range inputChan {
				log.Info().Msgf("Processing input for weapon %s", input.weaponID)

				err := models.SetBuildInProgress(db, input.BuildID)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to update evaluator status to inprogress for weapon %s", input.weaponID)
					err2 := models.SetBuildFailed(db, input.BuildID)
					if err2 != nil {
						log.Error().Err(err2).Msgf("Failed to set build failed for build %d", input.BuildID)
					}
					continue
				}

				weapon, err := candidate_tree.CreateWeaponCandidateTree(input.weaponID, "recoil", input.constraints, dataProvider)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create weapon tree for %s. Skipping", input.weaponID)
					err2 := models.SetBuildFailed(db, input.BuildID)
					if err2 != nil {
						log.Error().Err(err2).Msgf("Failed to set build failed for build %d", input.BuildID)
					}
					continue
				}

				weapon.SortAllowedItems("recoil-min")

				log.Info().Msgf("Generated weapon candidate tree for %s with constraints %v", input.weaponID, input.constraints)

				build := evaluator.FindBestBuild(weapon, "recoil", map[string]bool{})

				log.Info().Msgf("Evaluation complete - weapon %s with constraints %v", input.weaponID, input.constraints)

				evaledWeapon, err := build.ToEvaluatedWeapon()
				if err != nil {
					log.Error().Err(err).Msgf("Failed to convert result to evaluated weapon for weapon %s with constraints %v", input.weaponID, input.constraints)
					err2 := models.SetBuildFailed(db, input.BuildID)
					if err2 != nil {
						log.Error().Err(err2).Msgf("Failed to set build failed for build %d", input.BuildID)
					}
					continue
				}

				evaluationResult := evaledWeapon.ToItemEvaluationResult()

				err = models.SetBuildCompleted(db, input.BuildID, &evaluationResult)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to save build for weapon %s with constraints %v", input.weaponID, input.constraints)
					err2 := models.SetBuildFailed(db, input.BuildID)
					if err2 != nil {
						log.Error().Err(err2).Msgf("Failed to set build failed for build %d", input.BuildID)
					}
					continue
				}

				log.Info().Msgf("Saved build for weapon %s with constraints %v", input.weaponID, input.constraints)

				resultsChan <- EvaluationResult{
					BuildID:        input.BuildID,
					EvaluationType: "recoil",
					Weapon:         weapon,
					Result:         build,
				}
			}
		}()
	}

	// Send work to the input channel
	for i := 0; i < len(weaponIds); i++ {
		for j := 0; j < len(traderLevels); j++ {
			log.Info().Msgf("Sending work for weapon %s with constraints %v", weaponIds[i], traderLevels[j])

			constraints := models.EvaluationConstraints{
				TraderLevels:     traderLevels[j],
				IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
				IgnoredItemIDs:   []string{},
			}

			buildID, err := models.CreatePendingOptimumBuild(db, weaponIds[i], "recoil", constraints)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to create evaluator status for weapon %s", weaponIds[i])
				return nil
			}

			log.Info().Msgf("Sending to input %s, %v", weaponIds[i], constraints)
			inputChan <- Candidateinput{
				weaponID:    weaponIds[i],
				constraints: constraints,
				BuildID:     buildID,
			}
		}
	}

	close(inputChan)
	wg.Wait()
	close(resultsChan)

	results := make([]EvaluationResult, 0)
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}
