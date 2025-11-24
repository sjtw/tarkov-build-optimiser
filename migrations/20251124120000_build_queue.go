package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upBuildQueue, downBuildQueue)
}

func upBuildQueue(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		create table build_queue
		(
			queue_id serial primary key,
			item_id varchar not null,
			build_type varchar not null,
			jaeger_level int not null,
			prapor_level int not null,
			peacekeeper_level int not null,
			mechanic_level int not null,
			skier_level int not null,
			priority int not null default 10,
			status varchar not null check (status in ('Queued', 'Processing', 'Completed', 'Failed')),
			created_at timestamp with time zone not null default now(),
			started_at timestamp with time zone,
			completed_at timestamp with time zone,
			error_message text
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create build_queue table")
		return err
	}

	// Create index for efficient queue polling (highest priority first, oldest first)
	_, err = tx.ExecContext(ctx, `
		create index idx_build_queue_polling on build_queue(status, priority desc, created_at asc)
		where status in ('Queued', 'Processing');`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create index on build_queue table")
		return err
	}

	// Create index for checking if a build is already queued
	_, err = tx.ExecContext(ctx, `
		create index idx_build_queue_lookup on build_queue(item_id, build_type, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level, status);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create lookup index on build_queue table")
		return err
	}

	return nil
}

func downBuildQueue(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `drop table if exists build_queue;`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop build_queue table")
		return err
	}

	return nil
}
