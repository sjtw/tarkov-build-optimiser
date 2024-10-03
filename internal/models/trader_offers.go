package models

import (
	"database/sql"
	"github.com/rs/zerolog/log"
)

type TraderOffer struct {
	ID             string
	Name           string
	Trader         string
	MinTraderLevel int
	PriceRub       int
}

func UpsertTraderOffer(tx *sql.Tx, offer TraderOffer) error {
	query := `
		insert into trader_offers (
			item_id,
		   	name,
		   	trader,
			min_trader_level,
		   	price_rub
		)
		values ($1, $2, $3, $4, $5) on conflict (item_id) do update set
			name = $2,
			trader = $2,
			min_trader_level = $3,
			price_rub = $4
		;`
	_, err := tx.Exec(query, offer.ID, offer.Name, offer.Trader, offer.MinTraderLevel, offer.PriceRub)
	if err != nil {
		return err
	}
	return nil
}

func UpsertManyTraderOffers(tx *sql.Tx, offers []TraderOffer) error {
	for i := 0; i < len(offers); i++ {
		err := UpsertTraderOffer(tx, offers[i])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to upsert mod: %v", offers[i])
			return err
		}
		log.Info().Msgf("Upserted mod: ID: %s, Name: %s", offers[i].ID, offers[i].Name)
	}
	return nil
}
