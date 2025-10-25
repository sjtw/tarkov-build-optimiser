package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upComputedSubtrees, downComputedSubtrees)
}

func upComputedSubtrees(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
        create table if not exists computed_subtrees (
            root_item_id           varchar not null,
            build_type             varchar not null,
            jaeger_level           int not null,
            prapor_level           int not null,
            peacekeeper_level      int not null,
            mechanic_level         int not null,
            skier_level            int not null,
            depth_evaluated        int not null,
            game_data_version      varchar not null,
            recoil_sum             int not null,
            ergonomics_sum         int not null,
            chosen_assignments     jsonb not null,  -- array of {slot_id, item_id}
            chosen_item_ids        text[] not null,
            conflicts_item_ids     text[] not null,
            potential_min_recoil   int not null,
            potential_max_recoil   int not null,
            potential_min_ergonomics int not null,
            potential_max_ergonomics int not null,
            created_at             timestamp default now(),
            updated_at             timestamp default now(),
            unique (root_item_id, build_type, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level, depth_evaluated, game_data_version)
        );`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create computed_subtrees table")
	}

	// Useful GIN index for membership tests on chosen_item_ids
	_, err = tx.ExecContext(ctx, `
        create index if not exists idx_computed_subtrees_chosen_item_ids on computed_subtrees using gin (chosen_item_ids);
    `)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create GIN index on computed_subtrees.chosen_item_ids")
	}

	return nil
}

func downComputedSubtrees(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "drop table if exists computed_subtrees;")
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop computed_subtrees table")
	}
	return nil
}
