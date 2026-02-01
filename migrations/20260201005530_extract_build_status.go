package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upExtractBuildStatus, downExtractBuildStatus)
}

func upExtractBuildStatus(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE optimal_build_status (
			status_id SERIAL PRIMARY KEY,
			build_id INTEGER UNIQUE NOT NULL REFERENCES optimum_builds(build_id) ON DELETE CASCADE,
			status VARCHAR NOT NULL CHECK (status IN ('Pending', 'InProgress', 'Completed', 'Failed')),
			evaluation_start TIMESTAMP WITH TIME ZONE,
			evaluation_end TIMESTAMP WITH TIME ZONE
		);

		-- Migrate existing status data
		INSERT INTO optimal_build_status (build_id, status, evaluation_start, evaluation_end)
		SELECT build_id, status, evaluation_start, evaluation_end
		FROM optimum_builds;

		-- Remove columns from optimum_builds
		ALTER TABLE optimum_builds
		DROP COLUMN status,
		DROP COLUMN evaluation_start,
		DROP COLUMN evaluation_end;
	`)
	return err
}

func downExtractBuildStatus(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE optimum_builds
		ADD COLUMN status VARCHAR CHECK (status IN ('Pending', 'InProgress', 'Completed', 'Failed')),
		ADD COLUMN evaluation_start TIMESTAMP WITH TIME ZONE,
		ADD COLUMN evaluation_end TIMESTAMP WITH TIME ZONE;

		-- Restore data
		UPDATE optimum_builds ob
		SET status = obs.status,
			evaluation_start = obs.evaluation_start,
			evaluation_end = obs.evaluation_end
		FROM optimal_build_status obs
		WHERE ob.build_id = obs.build_id;

		DROP TABLE IF EXISTS optimal_build_status;
	`)
	return err
}
