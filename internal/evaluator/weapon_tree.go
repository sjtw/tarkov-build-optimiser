package evaluator

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type TreeDataProvider interface {
	GetWeaponById(id string) (*models.Weapon, error)
	GetSlotsByItemID(id string) ([]models.Slot, error)
	GetWeaponModById(id string) (*models.WeaponMod, error)
	GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error)
}

type WeaponTreeConstraints struct {
	ignoredSlotNames map[string]bool
}

type WeaponTree struct {
	Item        *Item
	db          *sql.DB
	dataService TreeDataProvider
	constraints WeaponTreeConstraints
}

func ConstructWeaponTree(id string, data TreeDataProvider) (*WeaponTree, error) {
	weaponTree := &WeaponTree{
		dataService: data,
		constraints: WeaponTreeConstraints{
			ignoredSlotNames: map[string]bool{"Scope": true, "Ubgl": true},
		},
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

	err = item.PopulateSlots()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", w.ID)
		return nil, err
	}

	weaponTree.Item = item

	return weaponTree, nil
}
