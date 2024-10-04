package evaluator

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"
)

type BuildSaver interface {
	Save(itemId string, buildType string, itemType string, sum int, build *models.ItemEvaluationResult, name string, constraints models.EvaluationConstraints) error
}

type PgBuildSaver struct {
	db *sql.DB
}

func (bs *PgBuildSaver) Save(itemId string, buildType string, itemType string, sum int, build *models.ItemEvaluationResult, name string, constraints models.EvaluationConstraints) error {
	err := models.UpsertOptimumBuild(bs.db, itemId, buildType, itemType, sum, build, name, constraints)
	if err != nil {
		return err
	}

	return nil
}

func CreatePgBuildSaver(db *sql.DB) BuildSaver {
	return &PgBuildSaver{
		db: db,
	}
}
