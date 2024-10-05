package evaluator

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"
)

type TraderOfferGetter interface {
	Get(itemID string) ([]models.TraderOffer, error)
}

type PgTraderOfferGetter struct {
	db *sql.DB
}

func CreatePgTraderOfferGetter(db *sql.DB) TraderOfferGetter {
	return &PgTraderOfferGetter{
		db: db,
	}
}

func (to *PgTraderOfferGetter) Get(itemID string) ([]models.TraderOffer, error) {
	offers, err := models.GetTraderOffersByItemID(to.db, itemID)
	if err != nil {
		return nil, err
	}
	return offers, nil
}