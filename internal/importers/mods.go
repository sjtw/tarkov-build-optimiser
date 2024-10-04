package importers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/cache"
	cli "tarkov-build-optimiser/internal/cli"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

func ImportMods(db *db.Database, api *tarkovdev.Api) error {
	modCache := cache.NewJSONFileCache("./file-caches/mods-cache.json")
	if cli.GetFlags().PurgeCache {
		fmt.Println("--purge-cache provided - purging mods file cache")
		err := modCache.Purge()
		if err != nil {
			return err
		}
	}

	var mods []models.WeaponMod
	if cli.GetFlags().UseCache {
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

	if cli.GetFlags().UseCache {
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

func getModsFromTarkovDev(api *tarkovdev.Api) ([]models.WeaponMod, error) {
	res, err := tarkovdev.GetWeaponMods(context.Background(), api.Client)
	if err != nil {
		log.Error().Err(err).Msg("failed to get weapon mods")
		return nil, err
	}

	weaponMods := make([]models.WeaponMod, 0, len(res.Items))

	for i := 0; i < len(res.Items); i++ {
		log.Info().Msgf("Importing weapon mod %s", res.Items[i].Name)
		mod := res.Items[i]
		newMod := models.WeaponMod{
			ID:                 mod.Id,
			Name:               mod.Name,
			ErgonomicsModifier: int(mod.ErgonomicsModifier),
			RecoilModifier:     int(mod.RecoilModifier),
		}

		var slots []models.Slot
		switch properties := mod.Properties.(type) {
		case *tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesWeaponMod,
			*tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesScope,
			*tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesBarrel,
			*tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesMagazine:
			slots, err = convertPropertiesToSlots(properties)
			if err != nil {
				return nil, err
			}
		default:
			fmt.Printf("unsupported mod properties type: %T - skipping", mod.Properties)
		}

		newMod.Slots = slots
		weaponMods = append(weaponMods, newMod)
	}
	return weaponMods, nil
}

func getModsFromCache(cache cache.FileCache) ([]models.WeaponMod, error) {
	all, err := cache.All()
	if err != nil {
		log.Error().Err(err).Msg("failed to get weapon mod from cache")
		return nil, err
	}

	keys := all.Keys()

	if len(keys) == 0 {
		return nil, errors.New("no weapon mods to import found in cache")
	}

	weapons := make([]models.WeaponMod, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		mod := models.WeaponMod{}
		err := all.Get(key, &mod)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get weapon %s from cache", key)
			return nil, err
		}

		weapons = append(weapons, mod)
	}

	return weapons, nil
}

func updateModFileCache(modCache cache.FileCache, mods []models.WeaponMod) error {
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
