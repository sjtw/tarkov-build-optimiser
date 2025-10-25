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
        create table if not exists conflict_free_cache (
            item_id                varchar not null,
            focused_stat           varchar not null,
            jaeger_level           int not null,
            prapor_level           int not null,
            peacekeeper_level      int not null,
            mechanic_level         int not null,
            skier_level            int not null,
            recoil_sum             int not null,
            ergonomics_sum         int not null,
            created_at             timestamp default now(),
            updated_at             timestamp default now(),
            unique (item_id, focused_stat, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level)
        );`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create conflict_free_cache table")
	}

	// Index for fast lookups by item_id and focused_stat
	_, err = tx.ExecContext(ctx, `
        create index if not exists idx_conflict_free_cache_lookup on conflict_free_cache (item_id, focused_stat);
    `)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create index on conflict_free_cache")
	}

	// Index for trader level range queries (used in GetConflictFreeCache)
	_, err = tx.ExecContext(ctx, `
        create index if not exists idx_conflict_free_cache_trader_levels on conflict_free_cache (item_id, focused_stat, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level);
    `)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create trader levels index on conflict_free_cache")
	}

	return nil
}

func downComputedSubtrees(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "drop table if exists conflict_free_cache;")
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop conflict_free_cache table")
	}
	return nil
}
