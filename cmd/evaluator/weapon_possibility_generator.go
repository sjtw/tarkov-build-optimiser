package main

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/evaluator"
)

type WeaponPossibilityResult struct {
	Item *evaluator.Item
	Id   string
	Ok   bool
}

func generateWeaponPossibilities(db *sql.DB, weaponIds []string, workerCount int) (weapons []WeaponPossibilityResult) {
	weaponCount := len(weaponIds)

	idChan := make(chan string, weaponCount)
	resultChan := make(chan WeaponPossibilityResult, weaponCount)
	doneChan := make(chan struct{})

	dataService := evaluator.CreateTreeDataService(db)

	for i := 0; i < workerCount; i++ {
		log.Debug().Msgf("Creating possibility generation worker %d", i)
		go weaponPossibilityWorker(dataService, idChan, resultChan, doneChan, i)
	}

	log.Debug().Msgf("Queuing weapons for possibility generation")
	for i := 0; i < weaponCount; i++ {
		idChan <- weaponIds[i]
	}
	log.Debug().Msgf("Queued %d weaponIds.", weaponCount)

	close(idChan)

	go func() {
		for i := 0; i < workerCount; i++ {
			<-doneChan
		}
		close(resultChan)
		close(doneChan)
	}()

	results := make([]WeaponPossibilityResult, 0, weaponCount)
	log.Debug().Msg("Collecting possibility generation results.")
	count := 0
	for result := range resultChan {
		if count%10 == 0 {
			log.Info().Msgf("Generated %d weapon possibilities, %d remaining.", count, weaponCount-count)
		}

		results = append(results, result)
		count++
	}

	return results
}

func weaponPossibilityWorker(data evaluator.DataProvider, weaponIds <-chan string, resultsChan chan<- WeaponPossibilityResult, doneChan chan<- struct{}, id int) {
	for weaponId := range weaponIds {
		log.Debug().Msgf("[Worker %d] Creating possibility tree for %s", id, weaponId)
		weapon, err := evaluator.ConstructWeaponTree(weaponId, data)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to generate weapon builds for %s", weaponId)
			resultsChan <- WeaponPossibilityResult{
				Item: nil,
				Id:   weaponId,
				Ok:   false,
			}
			continue
		}

		log.Debug().Msgf("[Worker %d] Finished possibility tree for %s", id, weaponId)

		resultsChan <- WeaponPossibilityResult{
			Id:   weaponId,
			Item: weapon.Item,
			Ok:   true,
		}
	}
	doneChan <- struct{}{}
}
