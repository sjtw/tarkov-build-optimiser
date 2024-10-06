package env

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"tarkov-build-optimiser/internal/helpers"

	"github.com/joho/godotenv"
)

type Env struct {
	PgHost      string
	PgPort      string
	PgUser      string
	PgPassword  string
	PgName      string
	Environment string
}

func Get() (Env, error) {
	if os.Getenv("CI") != "true" {
		projectRoot, err := helpers.GetProjectRoot()
		if err != nil {
			return Env{}, err
		}
		envFilePath := filepath.Join(projectRoot, ".env")

		err = godotenv.Load(envFilePath)
		if err != nil {
			return Env{}, err
		}
	}

	env := Env{
		PgHost:      os.Getenv("POSTGRES_HOST"),
		PgPort:      os.Getenv("POSTGRES_PORT"),
		PgUser:      os.Getenv("POSTGRES_USER"),
		PgPassword:  os.Getenv("POSTGRES_PASSWORD"),
		PgName:      os.Getenv("POSTGRES_DB"),
		Environment: os.Getenv("ENVIRONMENT"),
	}

	log.Debug().
		Str("PgHost", env.PgHost).
		Str("PgPort", env.PgPort).
		Str("PgName", env.PgName).
		Str("Environment", env.Environment).
		Msg("Environment variables loaded")

	return env, nil
}
