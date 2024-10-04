package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	goose.AddMigrationContext(upInitial, downInitial)
}

func upInitial(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		create table weapons
		(
				item_id           varchar primary key,
				name                varchar,
				recoil_modifier     int,
				ergonomics_modifier int
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create weapons table")
	}

	_, err = tx.ExecContext(ctx, `
		create table slots
		(
				slot_id 		   varchar primary key,
				item_id      	 varchar,
				name           varchar,
				unique (slot_id, item_id)
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create slots table")
	}

	_, err = tx.ExecContext(ctx, `
		create table slot_allowed_items
		(
			item_id varchar,
			slot_id         varchar,
			name            varchar,
			unique (slot_id, item_id)
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create slot_allowed_items table")
	}

	_, err = tx.ExecContext(ctx, `
		create table weapon_mods
		(
				item_id       varchar primary key,
				name                varchar,
				ergonomics_modifier int,
				recoil_modifier     int
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create weapon_mods table")
	}

	_, err = tx.ExecContext(ctx, `
		create table optimum_builds
		(
			item_id varchar,
			build jsonb,
			build_type varchar,
			modifier_sum integer,
			name varchar,
			jaeger_level int,
			prapor_level int,
			peacekeeper_level int,
			mechanic_level int,
			skier_level int
-- 			unique (item_id, build_type, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level)
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create optimum_builds table")
	}

	_, err = tx.ExecContext(ctx, `
		create table trader_offers
		(
		    item_id varchar,
		    name varchar,
		    trader varchar,
		    min_trader_level integer,
		    price_rub integer
		);`)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to create trader_offers table")
	}

	return nil
}

func downInitial(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "drop table if exists weapons;")
	if err != nil {
		_ = tx.Rollback()
		log.Fatal().Err(err).Msg("failed to drop weapons table")
	}

	_, err = tx.ExecContext(ctx, "drop table if exists slots;")
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		log.Fatal().Err(err).Msg("failed to drop slots table")
	}

	_, err = tx.ExecContext(ctx, "drop table if exists slot_allowed_items;")
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		log.Fatal().Err(err).Msg("failed to drop slot_allowed_items table")
	}

	_, err = tx.ExecContext(ctx, "drop table if exists weapon_mods;")
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		log.Fatal().Err(err).Msg("failed to drop weapon_mods table")
	}

	_, err = tx.ExecContext(ctx, "drop table if exists optimum_builds;")
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		log.Fatal().Err(err).Msg("failed to drop optimum_builds table")
	}

	_, err = tx.ExecContext(ctx, "drop table if exists trader_offers;")
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		log.Fatal().Err(err).Msg("failed to drop optimum_builds table")
	}

	return nil
}
