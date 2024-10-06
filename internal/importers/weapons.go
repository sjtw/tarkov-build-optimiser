package importers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

func ImportWeapons(db *db.Database, api *tarkovdev.Api) error {
	weaponsCache, err := cache.NewJSONFileCache("./file-caches/weapons-cache.json")
	if err != nil {
		return err
	}

	if cli.GetFlags().PurgeCache {
		fmt.Println("--purge-cache provided - purging weapons file cache")
		err := weaponsCache.Purge()
		if err != nil {
			return err
		}
	}

	var weapons []models.Weapon
	if cli.GetFlags().UseCache {
		log.Debug().Msg("--use-cache provided - pulling weapons from file cache.")
		data, err := getWeaponsFromCache(weaponsCache)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get weapons from file cache.")
			return err
		}
		log.Debug().Msg("Retrieved weapons from file cache.")
		weapons = data
	} else {
		log.Debug().Msg("Fetching weapons from Tarkov.dev API.")
		res, err := getWeaponsFromTarkovDev(api)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get weapons from Tarkov.dev API.")
			return err
		}
		log.Debug().Msg("Retrieved weapons from Tarkov.dev API..")
		weapons = res

		log.Debug().Msg("Updating weapon file cache with weapons from Tarkov.dev.")
		err = updateWeaponFileCache(weaponsCache, weapons)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update weapon cache with weapons from Tarkov.dev.")
			return err
		}
		log.Debug().Msgf("Added weapon %d to file cache.", len(weapons))
	}

	if cli.GetFlags().CacheOnly {
		log.Debug().Msg("--cache-only was provided - Not persisting weapons in db.")
		return nil
	}

	log.Debug().Msg("Beginning transaction.")
	tx, err := db.Conn.Begin()
	if err != nil {
		log.Error().Err(err).Msg("Failed to begin transaction.")
		return err
	}

	err = models.UpsertManyWeapon(tx, weapons)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func getWeaponsFromCache(cache cache.FileCache) ([]models.Weapon, error) {
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
		log.Debug().Msgf("Importing weapon %s", res.Items[i].Name)

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
				log.Debug().Msgf("unsupported weapon mod properties type: %T - skipping", weapon.Properties)
			}

			newWeapon.Slots = slots
		}

		weapons = append(weapons, newWeapon)
	}

	return weapons, nil
}

func updateWeaponFileCache(weaponsCache cache.FileCache, weapons []models.Weapon) error {
	for i := 0; i < len(weapons); i++ {
		log.Debug().Msgf("Storing weapon %s in file cache", weapons[i].ID)
		err := weaponsCache.Store(weapons[i].ID, weapons[i])
		if err != nil {
			return err
		}
	}

	return nil
}
