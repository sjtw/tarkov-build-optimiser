package importers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/tarkovdev"
)

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

func getModsFromCache(cache *cache.JSONFileCache) ([]models.WeaponMod, error) {
	fmt.Println("--use-cache provided - using weapon mod file cache")
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
