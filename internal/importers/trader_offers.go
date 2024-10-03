package importers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

type TraderItemOffer struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Trader         string `json:"trader"`
	MinTraderLevel int    `json:"min_trader_level"`
	PriceRub       int    `json:"price_rub"`
}

func ImportTraderOffers(db *db.Database, api *tarkovdev.Api) error {
	traderOfferCache := cache.NewJSONFileCache("./file-caches/trader-offers-cache.json")
	if helpers.ContainsStr(os.Args, "--purge-cache") {
		fmt.Println("--purge-cache provided - purging trader offers file cache")
		err := traderOfferCache.Purge()
		if err != nil {
			return err
		}
	}

	var offers []models.TraderOffer

	if helpers.ContainsStr(os.Args, "--use-cache") {
		log.Info().Msg("--use-cache provided - pulling trader offers from file cache.")
		data, err := getTraderOffersFromCache(traderOfferCache)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get trader offers from file cache.")
			return err
		}
		offers = data
		log.Info().Msg("Retrieved trader offers from file cache.")
	} else {
		log.Info().Msg("Fetching trader offers from Tarkov.dev API.")
		res, err := getTraderOffersFromTarkovDev(api)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get trader offers from Tarkov.dev API.")
			return err
		}
		offers = res
		log.Info().Msg("Retrieved trader offers from Tarkov.dev API..")

		log.Info().Msgf("Storing %d trader offers in file cache.", len(offers))
		err = updateTraderOffercache(traderOfferCache, offers)
		if err != nil {
			log.Error().Err(err).Msg("Failed to store trader offers in file cache.")
			return err
		}
		log.Info().Msgf("Added all %d trader offers to file cache.", len(offers))
	}

	if helpers.ContainsStr(os.Args, "--cache-only") {
		log.Info().Msg("--cache-only was provided - Not persisting trader offers in db.")
		return nil
	}

	log.Info().Msg("Beginning db transaction.")
	tx, err := db.Conn.Begin()
	if err != nil {
		log.Info().Err(err).Msg("Failed to begin db transaction.")
		return err
	}

	log.Info().Msgf("Importing %d trader offers", len(offers))
	err = models.UpsertManyTraderOffers(tx, offers)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Error().Err(err).Msg("failed to roll back transaction.")
			return rollbackErr
		}
		log.Error().Err(err).Msg("Failed to upsert trader offers.")
		return err
	}

	log.Info().Msg("All trader offers imported, committing transaction.")
	err = tx.Commit()
	if err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction.")
		return err
	}

	return nil
}

func getTraderOffersFromTarkovDev(api *tarkovdev.Api) ([]models.TraderOffer, error) {
	ctx := context.Background()
	res, err := tarkovdev.GetItemPrices(ctx, api.Client)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get item prices")
		return nil, err
	}

	items := make([]models.TraderOffer, 0, len(res.Items))

	for i := 0; i < len(res.Items); i++ {
		log.Info().Msgf("Processing Item %s price.", res.Items[i].Id)
		item := res.Items[i]

		newItem := models.TraderOffer{
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

func getTraderOffersFromCache(cache *cache.JSONFileCache) ([]models.TraderOffer, error) {
	fmt.Println("--use-cache provided - using trader offer file cache")
	all, err := cache.All()
	if err != nil {
		log.Error().Err(err).Msg("failed to get trader offer from cache")
		return nil, err
	}

	keys := all.Keys()

	if len(keys) == 0 {
		return nil, errors.New("no trader offers to import found in cache")
	}

	traderOffers := make([]models.TraderOffer, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		mod := models.TraderOffer{}
		err := all.Get(key, &mod)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get trader offer %s from cache", key)
			return nil, err
		}

		traderOffers = append(traderOffers, mod)
	}

	return traderOffers, nil
}

func updateTraderOffercache(traderOfferCache *cache.JSONFileCache, traderOffers []models.TraderOffer) error {
	for i := 0; i < len(traderOffers); i++ {
		err := traderOfferCache.Store(traderOffers[i].ID, traderOffers[i])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to store trader offer %v in file cache", traderOffers[i])
			return err
		}
		log.Info().Msgf("Mod stored in file cache: %v", traderOffers[i])
	}
	return nil
}
