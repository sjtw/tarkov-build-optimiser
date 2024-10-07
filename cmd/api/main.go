package main

import (
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/router"
)

func main() {
	environment, err := env.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get environment variables")
	}

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to db")
	}

	cfg := router.Config{DB: dbClient}
	r := router.NewRouter(cfg)

	err = r.Start(":8080")
	if err != nil {
		return
	}
}
