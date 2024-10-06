package evaluator

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"
)

type PgBuildSaver struct {
	db *sql.DB
}

func (bs *PgBuildSaver) Save(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints, isSubtree bool) error {
	err := models.UpsertOptimumBuild(bs.db, build, constraints, isSubtree)
	if err != nil {
		return err
	}

	return nil
}

func CreatePgBuildSaver(db *sql.DB) *PgBuildSaver {
	return &PgBuildSaver{
		db: db,
	}
}
