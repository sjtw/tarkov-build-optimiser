package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upItemCategories, downItemCategories)
}

func upItemCategories(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table weapon_mods
    	add column category_name varchar,
    	add column category_id varchar;`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to add category_name and category_id to weapon_mods")
	}
	return nil
}

func downItemCategories(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `alter table weapon_mods
		drop column category_name,
		drop column category_id;`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop category_name and category_id from weapon_mods")
	}

	return nil
}
