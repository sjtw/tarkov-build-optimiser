package importers

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

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

func updateModFileCache(modCache *cache.JSONFileCache, mods []models.WeaponMod) error {
	for i := 0; i < len(mods); i++ {
		err := modCache.Store(mods[i].ID, mods[i])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to store mod in file cache %v", mods[i])
			return err
		}
		log.Info().Msgf("Mod stored in file cache: %v", mods[i])
	}
	return nil
}

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

func ImportMods(db *db.Database, api *tarkovdev.Api) error {
	modCache := cache.NewJSONFileCache("./file-caches/mods-cache.json")
	if helpers.ContainsStr(os.Args, "--purge-cache") {
		fmt.Println("--purge-cache provided - purging mods file cache")
		err := modCache.Purge()
		if err != nil {
			return err
		}
	}

	var mods []models.WeaponMod
	if helpers.ContainsStr(os.Args, "--use-cache") {
		log.Info().Msg("--use-cache provided - pulling mods from file cache.")
		data, err := getModsFromCache(modCache)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get mods from file cache.")
			return err
		}
		mods = data
		log.Info().Msg("Retrieved mods from file cache.")
	} else {
		log.Info().Msg("Fetching mods from Tarkov.dev API.")
		res, err := getModsFromTarkovDev(api)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get mods from Tarkov.dev API.")
			return err
		}
		mods = res
		log.Info().Msg("Retrieved mods from Tarkov.dev API..")

		log.Info().Msgf("Storing %d mods in file cache.", len(mods))
		err = updateModFileCache(modCache, mods)
		if err != nil {
			log.Error().Err(err).Msg("Failed to store mods in file cache.")
			return err
		}
		log.Info().Msgf("Added all %d mods to file cache.", len(mods))
	}

	if helpers.ContainsStr(os.Args, "--cache-only") {
		log.Info().Msg("--cache-only was provided - Not persisting mods in db.")
		return nil
	}

	log.Info().Msg("Beginning db transaction.")
	tx, err := db.Conn.Begin()
	if err != nil {
		log.Info().Err(err).Msg("Failed to begin db transaction.")
		return err
	}

	log.Info().Msgf("Importing %d mods", len(mods))
	err = models.UpsertManyMod(tx, mods)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Error().Err(err).Msg("failed to roll back transaction.")
			return rollbackErr
		}
		log.Error().Err(err).Msg("Failed to upsert mods.")
		return err
	}

	log.Info().Msg("All mods imported, committing transaction.")
	err = tx.Commit()
	if err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction.")
		return err
	}

	return nil
}
