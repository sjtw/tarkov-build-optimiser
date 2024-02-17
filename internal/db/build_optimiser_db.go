package db

import (
	"tarkov-build-optimiser/internal/env"

	"github.com/rs/zerolog/log"
)

// CreateBuildOptimiserDBClient helper to create db connection to the pg db
// using the current env
func CreateBuildOptimiserDBClient() (*Database, error) {
	e, err := env.Get()
	if err != nil {
		return nil, err
	}

	log.Info().Msg("connecting to database")
	db, err := NewDatabase(Config{
		Host:     e.PgHost,
		Port:     e.PgPort,
		User:     e.PgUser,
		Password: e.PgPassword,
		Name:     e.PgName,
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}
