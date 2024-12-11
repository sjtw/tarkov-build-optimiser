package main

import (
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/importers"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"

	"github.com/rs/zerolog/log"
)

// imports from json file by default into postgres
// use `go run main.go purge-cache` to clear the cache
func main() {
	e, err := env.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get environment variables")
	}

	log.Debug().Msg("connecting to database")
	dbClient, err := db.NewDatabase(db.Config{
		Host:     e.PgHost,
		Port:     e.PgPort,
		User:     e.PgUser,
		Password: e.PgPassword,
		Name:     e.PgName,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	defer func(dbClient *db.Database) {
		_ = dbClient.Close()
	}(dbClient)

	api := tarkovdev.New()

	log.Debug().Msg("Purging all models.")
	err = models.Purge(dbClient.Conn)
	if err != nil {
		log.Error().Err(err).Msg("failed to purge all models.")
	}
	log.Debug().Msg("Models purged.")

	log.Debug().Msg("Importing weapons.")
	err = importers.ImportWeapons(dbClient, api)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to import weapons.")
	}
	log.Debug().Msg("All weapons imported OK.")

	log.Debug().Msg("Importing mods.")
	err = importers.ImportMods(dbClient, api)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to import mods.")
	}
	log.Debug().Msg("All mods imported OK.")

	log.Debug().Msg("Importing trader offers.")
	err = importers.ImportTraderOffers(dbClient, api)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to import trader offers.")
	}
	log.Debug().Msg("All trader offers imported OK.")

	err = models.PurgeOptimumBuilds(dbClient.Conn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to purge optimal builds after importing.")
	}
}
