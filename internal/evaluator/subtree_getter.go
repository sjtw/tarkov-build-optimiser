package evaluator

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"
)

type PgSubtreeGetter struct {
	db *sql.DB
}

func (sg *PgSubtreeGetter) Get(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	subtree, err := models.GetEvaluatedSubtree(sg.db, itemId, buildType, constraints)
	if err != nil {
		return nil, err
	}

	return subtree, nil
}

func CreatePgSubtreeGetter(db *sql.DB) *PgSubtreeGetter {
	return &PgSubtreeGetter{
		db: db,
	}
}
