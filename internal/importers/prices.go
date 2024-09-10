package importers

import (
	"context"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/tarkovdev"
)

type TraderItemOffer struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Trader         string `json:"trader"`
	MinTraderLevel int    `json:"min_trader_level"`
	PriceRub       int    `json:"price_rub"`
}

func getTraderOffers(api *tarkovdev.Api) ([]TraderItemOffer, error) {
	ctx := context.Background()
	res, err := tarkovdev.GetItemPrices(ctx, api.Client)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get item prices")
		return nil, err
	}

	items := make([]TraderItemOffer, 0, len(res.Items))

	for i := 0; i < len(res.Items); i++ {
		log.Info().Msgf("Processing Item %s price.", res.Items[i].Id)
		item := res.Items[i]

		newItem := TraderItemOffer{
			ID:   item.Id,
			Name: item.Name,
		}

		offers := item.GetBuyFor()
		for j := 0; j < len(offers); j++ {
			newItem.PriceRub = offers[j].PriceRUB

			switch trader := offers[j].Vendor.(type) {
			case *tarkovdev.GetItemPricesItemsItemBuyForItemPriceVendorTraderOffer:
				newItem.MinTraderLevel = trader.MinTraderLevel
			case *tarkovdev.GetItemPricesItemsItemBuyForItemPriceVendorFleaMarket:
				break
			}
		}

		items = append(items, newItem)
	}

	return items, nil
}
