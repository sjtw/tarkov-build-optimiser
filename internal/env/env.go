package env

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"tarkov-build-optimiser/internal/helpers"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Env struct {
	PgHost      string
	PgPort      string
	PgUser      string
	PgPassword  string
	PgName      string
	Environment string
	// cpu-core multiplier for the number of evaluator workers to create
	// e.g. given 4 cores & POOL_SIZE_MULTIPLIER=2, 8 evaluator workers will be created
	EvaluatorPoolSizeFactor int
	// EvaluatorFresh when true causes optimum builds to be purged before evaluation (same as --fresh)
	EvaluatorFresh bool
}

var (
	once   sync.Once
	envErr error
	env    = Env{}
)

func getInt(key string, def int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to convert env var %s to integer", key)
		return def
	}
	return value
}

func getBoolTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func load() {
	projectRoot, err := helpers.GetProjectRoot()
	if err != nil {
		envErr = err
	}
	envFilePath := filepath.Join(projectRoot, ".env")

	err = godotenv.Load(envFilePath)
	if err != nil {
		envErr = err
	}

	env = Env{
		PgHost:                  os.Getenv("POSTGRES_HOST"),
		PgPort:                  os.Getenv("POSTGRES_PORT"),
		PgUser:                  os.Getenv("POSTGRES_USER"),
		PgPassword:              os.Getenv("POSTGRES_PASSWORD"),
		PgName:                  os.Getenv("POSTGRES_DB"),
		Environment:             os.Getenv("ENVIRONMENT"),
		EvaluatorPoolSizeFactor: getInt("POOL_SIZE_MULTIPLIER", 2),
		EvaluatorFresh:          getBoolTruthy("EVALUATOR_FRESH"),
	}

	log.Debug().
		Str("PgHost", env.PgHost).
		Str("PgPort", env.PgPort).
		Str("PgName", env.PgName).
		Str("Environment", env.Environment).
		Msg("Environment variables loaded")
}

func Get() (Env, error) {
	once.Do(load)

	return env, envErr
}
