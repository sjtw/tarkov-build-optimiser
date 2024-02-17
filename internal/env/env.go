package env

import (
	"os"
	"path/filepath"
	"tarkov-build-optimiser/internal/helpers"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Env struct {
	PgHost     string
	PgPort     string
	PgUser     string
	PgPassword string
	PgName     string
}

func Get() (Env, error) {
	projectRoot, err := helpers.GetProjectRoot()
	if err != nil {
		return Env{}, err
	}
	envFilePath := filepath.Join(projectRoot, ".env")

	err = godotenv.Load(envFilePath)
	if err != nil {
		return Env{}, err
	}

	env := Env{
		PgHost:     os.Getenv("POSTGRES_HOST"),
		PgPort:     os.Getenv("POSTGRES_PORT"),
		PgUser:     os.Getenv("POSTGRES_USER"),
		PgPassword: os.Getenv("POSTGRES_PASSWORD"),
		PgName:     os.Getenv("POSTGRES_DB"),
	}

	log.Info().Interface("env", env).Msg("Environment variables")

	return env, nil
}
