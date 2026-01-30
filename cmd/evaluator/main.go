package main

import (
	"database/sql"
	"runtime"
	"sync"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"

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

	// Create cache (choose between memory or database)
	var cache evaluator.Cache
	if flags.UseDatabaseCache {
		cache = evaluator.NewDatabaseCache(dbClient.Conn)
		log.Info().Msg("Using DATABASE cache for conflict-free items")
	} else {
		cache = evaluator.NewMemoryCache()
		log.Info().Msg("Using MEMORY cache for conflict-free items")
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

	workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor

	log.Info().Msgf("Evaluating %d weapons", len(weaponIds))

	dataService := candidate_tree.CreateDataService(dbClient.Conn)
	evaluate(weaponIds, dataService, workerCount, traderLevels, dbClient.Conn, cache)

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

func evaluate(weaponIds []string, dataProvider candidate_tree.TreeDataProvider, workerCount int, traderLevels [][]models.TraderLevel, db *sql.DB, cache evaluator.Cache) {
	inputChan := make(chan Candidateinput, workerCount*2)
	resultsChan := make(chan EvaluationResult, workerCount*2)
	wg := sync.WaitGroup{}

	// Start a goroutine to continuously flush results (prevents memory accumulation)
	// This allows results to be processed for side effects without keeping them in memory
	resultsWg := sync.WaitGroup{}
	resultsWg.Add(1)
	go func() {
		defer resultsWg.Done()
		for result := range resultsChan {
			// Results are flushed immediately after being received
			// Add any side effects processing here in the future
			_ = result // Currently just discarded to prevent accumulation
		}
	}()

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
				build := evaluator.FindBestBuild(weapon, "recoil", map[string]bool{}, cache)

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
			log.Debug().Msgf("Sending work for weapon %s with constraints %v", weaponIds[i], traderLevels[j])

			constraints := models.EvaluationConstraints{
				TraderLevels:     traderLevels[j],
				IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
				IgnoredItemIDs:   []string{},
			}

			buildID, err := models.CreatePendingOptimumBuild(db, weaponIds[i], "recoil", constraints)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to create evaluator status for weapon %s", weaponIds[i])
				return
			}

			log.Debug().Msgf("Sending to input %s, %v", weaponIds[i], constraints)
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
	resultsWg.Wait()
}
