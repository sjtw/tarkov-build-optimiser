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
	idChan := make(chan string, len(weaponIds))
	resultChan := make(chan WeaponPossibilityResult, len(weaponIds))
	doneChan := make(chan struct{})

	for i := 0; i < workerCount; i++ {
		log.Info().Msgf("Creating possibility generation worker %d", i)
		go weaponPossibilityWorker(db, idChan, resultChan, doneChan, i)
	}

	log.Info().Msgf("Queuing weapons for possibility generation")
	for i := 0; i < len(weaponIds); i++ {
		idChan <- weaponIds[i]
	}
	log.Info().Msgf("Queued %d weaponIds.", len(weaponIds))

	close(idChan)

	go func() {
		for i := 0; i < workerCount; i++ {
			<-doneChan
		}
		close(resultChan)
	}()

	results := make([]WeaponPossibilityResult, len(weaponIds))
	log.Info().Msg("Collecting possibility generation results.")
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

func weaponPossibilityWorker(db *sql.DB, weaponIds <-chan string, resultsChan chan<- WeaponPossibilityResult, doneChan chan<- struct{}, id int) {
	for weaponId := range weaponIds {
		log.Info().Msgf("[Worker %d] Creating possibility tree for %s", id, weaponId)
		weapon, err := evaluator.CreateWeaponPossibilityTree(db, weaponId)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to generate weapon builds for %s", weaponId)
			resultsChan <- WeaponPossibilityResult{
				Item: nil,
				Id:   weaponId,
				Ok:   false,
			}
			continue
		}

		log.Info().Msgf("[Worker %d] Finished possibility tree for %s", id, weaponId)

		resultsChan <- WeaponPossibilityResult{
			Id:   weaponId,
			Item: weapon,
			Ok:   true,
		}
	}
	doneChan <- struct{}{}
}
