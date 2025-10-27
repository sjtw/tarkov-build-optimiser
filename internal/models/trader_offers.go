package models

import (
	"context"
	"database/sql"
	"strings"

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
		   	price_rub,
		    trader_lowercase
		)
		values ($1, $2, $3, $4, $5, $6)`
	_, err := tx.Exec(query, offer.ID, offer.Name, offer.Trader, offer.MinTraderLevel, offer.PriceRub, strings.ToLower(offer.Trader))
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

// GetLowestPriceForItem returns the lowest trader offer price for itemID that satisfies provided trader levels.
// ok == false indicates no eligible offer exists.
func GetLowestPriceForItem(ctx context.Context, db *sql.DB, itemID string, traderLevels []TraderLevel) (price int, ok bool, err error) {
	levelMap := map[string]int{}
	for _, tl := range traderLevels {
		levelMap[strings.ToLower(tl.Name)] = tl.Level
	}

	levelValue := func(name string) int {
		if v, exists := levelMap[name]; exists {
			return v
		}
		return 0
	}

	query := `
		select min(price_rub)
		from trader_offers
		where item_id = $1
		  and trader_lowercase is not null
		  and trader_lowercase in ('jaeger', 'prapor', 'peacekeeper', 'mechanic', 'skier')
		  and min_trader_level <= case trader_lowercase
			  when 'jaeger' then $2
			  when 'prapor' then $3
			  when 'peacekeeper' then $4
			  when 'mechanic' then $5
			  when 'skier' then $6
			  else 0
		  end
	`

	var lowest sql.NullInt64
	err = db.QueryRowContext(ctx, query, itemID,
		levelValue("jaeger"),
		levelValue("prapor"),
		levelValue("peacekeeper"),
		levelValue("mechanic"),
		levelValue("skier"),
	).Scan(&lowest)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}

	if !lowest.Valid {
		return 0, false, nil
	}

	return int(lowest.Int64), true, nil
}
