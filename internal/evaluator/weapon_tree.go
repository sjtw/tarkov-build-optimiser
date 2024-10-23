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
	// all itemIDs which conflict globally with other itemIDs
	AllowedItemConflicts map[string]map[string]bool
	// all candidate items for this weapon
	CandidateItems map[string]bool
}

func (wt *WeaponTree) AddItemConflicts(itemId string, conflictIDs []string) {
	if _, ok := wt.AllowedItemConflicts[itemId]; !ok {
		wt.AllowedItemConflicts[itemId] = map[string]bool{}
	}

	for _, conflictId := range conflictIDs {
		wt.AllowedItemConflicts[itemId][conflictId] = true
	}
}

func (wt *WeaponTree) AddCandidateItem(itemID string) {
	wt.CandidateItems[itemID] = true
}

func ConstructWeaponTree(id string, data TreeDataProvider) (*WeaponTree, error) {
	weaponTree := &WeaponTree{
		dataService: data,
		constraints: WeaponTreeConstraints{
			ignoredSlotNames: map[string]bool{"Scope": true, "Ubgl": true},
		},
		AllowedItemConflicts: map[string]map[string]bool{},
		CandidateItems:       map[string]bool{},
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
