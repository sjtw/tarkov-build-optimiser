package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upEvaluationStatus, downEvaluationStatus)
}

func upEvaluationStatus(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		alter table optimum_builds 
		    add column build_id serial primary key,
			add column evaluation_start timestamp with time zone,
			add column evaluation_end timestamp with time zone,
			add column status varchar check (status in ('Pending', 'InProgress', 'Completed', 'Failed'))
		;`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Msg("failed to add build_id, evaluation_start, evaluation_end, status to optimum_builds table")
		return err
	}

	return nil
}

func downEvaluationStatus(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		alter table optimum_builds
			drop column build_id,
			drop column evaluation_start,
			drop column evaluation_end,
			drop column status`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Msg("failed to drop build_id, evaluation_start, evaluation_end, status from  optimum_builds table")
		return err
	}

	return nil
}
