package evaluator

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type DataProvider interface {
	GetWeaponById(id string) (*models.Weapon, error)
	GetSlotsByItemID(id string) ([]models.Slot, error)
	GetWeaponModById(id string) (*models.WeaponMod, error)
	GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error)
}

type WeaponTree struct {
	Item        *Item
	db          *sql.DB
	dataService DataProvider
}

func ConstructWeaponTree(id string, data DataProvider) (*WeaponTree, error) {
	weaponTree := &WeaponTree{
		dataService: data,
	}

	w, err := data.GetWeaponById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}
	item := &Item{
		ID:                 id,
		Name:               w.Name,
		RecoilModifier:     w.RecoilModifier,
		ErgonomicsModifier: w.ErgonomicsModifier,
		Slots:              []*ItemSlot{},
		parentSlot:         nil,
		Type:               "weapon",
		RootWeaponTree:     weaponTree,
	}

	err = item.PopulateSlots([]string{"Sight", "Ubgl"})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", w.ID)
		return nil, err
	}

	weaponTree.Item = item

	return weaponTree, nil
}
