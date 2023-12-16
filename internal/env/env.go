package env

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Env struct {
	PgHost     string
	PgPort     string
	PgUser     string
	PgPassword string
	PgName     string
}

func Get() (Env, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return Env{}, err
	}

	envFilePath := filepath.Join(projectRoot, "../.env")
	err = godotenv.Load(envFilePath)
	if err != nil {
		return Env{}, err
	}

	return Env{
		PgHost:     os.Getenv("POSTGRES_HOST"),
		PgPort:     os.Getenv("POSTGRES_PORT"),
		PgUser:     os.Getenv("POSTGRES_USER"),
		PgPassword: os.Getenv("POSTGRES_PASSWORD"),
		PgName:     os.Getenv("POSTGRES_DB"),
	}, nil
}
