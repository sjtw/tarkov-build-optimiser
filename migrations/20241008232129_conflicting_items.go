package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upConflictingItems, downConflictingItems)
}

func upConflictingItems(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
	create table conflicting_items
	    (
	        item_id varchar,
	        conflicting_item_id varchar,
 			unique (item_id, conflicting_item_id)
	    );`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create conflicting_items table")
	}
	return nil
}

func downConflictingItems(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `drop table conflicting_items;`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop conflicting_items")
	}

	return nil
}
