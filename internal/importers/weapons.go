package importers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

func getWeaponsFromCache(cache *cache.JSONFileCache) ([]models.Weapon, error) {
	fmt.Println("--use-cache provided - using weapons file cache")
	all, err := cache.All()
	if err != nil {
		log.Error().Err(err).Msg("failed to get weapons from cache")
		return nil, err
	}

	keys := all.Keys()

	if len(keys) == 0 {
		return nil, errors.New("no weapons to import found in cache")
	}

	weapons := make([]models.Weapon, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		weapon := models.Weapon{}
		err := all.Get(key, &weapon)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get weapon %s from cache", key)
			return nil, err
		}

		weapons = append(weapons, weapon)
	}

	return weapons, nil
}

func getWeaponsFromTarkovDev(api *tarkovdev.Api) ([]models.Weapon, error) {
	fmt.Println("--use-cache not provided - fetching weapons from tarkov.dev")
	res, err := tarkovdev.GetWeapons(context.Background(), api.Client)
	if err != nil {
		log.Error().Err(err).Msg("failed to get weapons")
		return nil, err
	}

	if res == nil {
		return nil, errors.New("no weapons in response from tarkovdev")
	}

	if len(res.Items) == 0 {
		return nil, errors.New("no weapons to import")
	}

	weapons := make([]models.Weapon, 0, len(res.Items))
	for i := 0; i < len(res.Items); i++ {
		log.Info().Msgf("Importing weapon %s", res.Items[i].Name)

		weapon := res.Items[i]
		newWeapon := models.Weapon{
			ID:                 weapon.Id,
			Name:               weapon.Name,
			ErgonomicsModifier: int(weapon.ErgonomicsModifier),
			RecoilModifier:     int(weapon.RecoilModifier),
			Slots:              []models.Slot{},
		}

		var types []string
		for i := 0; i < len(weapon.Types); i++ {
			types = append(types, string(weapon.Types[i]))
		}
		if !helpers.ContainsStr(types, "preset") {
			var slots []models.Slot
			switch properties := weapon.Properties.(type) {
			case *tarkovdev.GetWeaponsItemsItemPropertiesItemPropertiesWeapon:
				slots, err = convertPropertiesToSlots(properties)
				if err != nil {
					return nil, err
				}
			default:
				fmt.Printf("unsupported weapon mod properties type: %T - skipping", weapon.Properties)
			}

			newWeapon.Slots = slots
		}

		weapons = append(weapons, newWeapon)
	}

	return weapons, nil
}
