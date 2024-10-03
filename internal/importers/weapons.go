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

func ImportWeapons(db *db.Database, api *tarkovdev.Api) error {
	useCache := helpers.ContainsStr(os.Args, "--use-cache")

	weaponsCache := cache.NewJSONFileCache("./file-caches/weapons-cache.json")
	if helpers.ContainsStr(os.Args, "--purge-cache") {
		fmt.Println("--purge-cache provided - purging weapons file cache")
		err := weaponsCache.Purge()
		if err != nil {
			return err
		}
	}

	var weapons []models.Weapon
	if useCache {
		log.Info().Msg("--use-cache provided - pulling weapons from file cache.")
		data, err := getWeaponsFromCache(weaponsCache)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get weapons from file cache.")
			return err
		}
		log.Info().Msg("Retrieved weapons from file cache.")
		weapons = data
	} else {
		log.Info().Msg("Fetching weapons from Tarkov.dev API.")
		res, err := getWeaponsFromTarkovDev(api)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get weapons from Tarkov.dev API.")
			return err
		}
		log.Info().Msg("Retrieved weapons from Tarkov.dev API..")
		weapons = res

		log.Info().Msg("Updating weapon file cache with weapons from Tarkov.dev.")
		err = updateWeaponFileCache(weaponsCache, weapons)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update weapon cache with weapons from Tarkov.dev.")
			return err
		}
		log.Info().Msgf("Added weapon %d to file cache.", len(weapons))
	}

	if helpers.ContainsStr(os.Args, "--cache-only") {
		log.Info().Msg("--cache-only was provided - Not persisting weapons in db.")
		return nil
	}

	log.Info().Msg("Beginning transaction.")
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

func updateWeaponFileCache(weaponsCache *cache.JSONFileCache, weapons []models.Weapon) error {
	for i := 0; i < len(weapons); i++ {
		log.Info().Msgf("Storing weapon %s in file cache", weapons[i].ID)
		err := weaponsCache.Store(weapons[i].ID, weapons[i])
		if err != nil {
			return err
		}
	}

	return nil
}
