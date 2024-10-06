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
		values ($1, $2, $3, $4, $5)`
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
			log.Error().Err(err).Msgf("Failed to upsert trader offer: %v", offers[i])
			return err
		}
		log.Debug().Msgf("Upserted trader offer: item_id: %s, Name: %s", offers[i].ID, offers[i].Name)
	}
	return nil
}

func GetTraderOffersByItemID(db *sql.DB, itemID string) ([]TraderOffer, error) {
	query := `select item_id, name, trader, min_trader_level, price_rub from trader_offers where item_id = $1`

	rows, err := db.Query(query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	offers := make([]TraderOffer, 0)
	for rows.Next() {
		offer := &TraderOffer{}
		err := rows.Scan(&offer.ID, &offer.Name, &offer.Trader, &offer.MinTraderLevel, &offer.PriceRub)
		if err != nil {
			return nil, err
		}
		offers = append(offers, *offer)
	}

	return offers, nil
}
