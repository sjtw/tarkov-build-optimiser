package evaluator

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type ItemSlot struct {
	ID                 string  `json:"id" bson:"id"`
	Name               string  `json:"name" bson:"name"`
	AllowedItems       []*Item `json:"-"`
	BestRecoilItem     *Item   `json:"best_recoil_item" bson:"best_recoil_item"`
	BestRecoilModifier int     `json:"best_recoil_modifier" bson:"best_recoil_modifier"`
	BestErgoModifier   int     `json:"best_ergo_modifier" bson:"best_ergo_modifier"`
	BestErgoItem       *Item   `json:"best_ergo_item" bson:"best_ergo_item"`
	parentItem         *Item
	// IDs of items which would potentially create a circular reference
	// we don't want to add these to the possibility tree for obvious reasons,
	// but we may still want to know what they are
	AllowedCircularReferenceItemIds []string `json:"allowed_circular_reference_item_ids"`
}

func ConstructSlot(id string, name string) *ItemSlot {
	return &ItemSlot{
		ID:           id,
		Name:         name,
		AllowedItems: make([]*Item, 0),
		parentItem:   nil,
	}
}

func (slot *ItemSlot) GetParentItem() *Item {
	if slot.AllowedItems == nil {
		return nil
	}

	return slot.parentItem
}

func (slot *ItemSlot) AddChildItem(item *Item) {
	item.SetParentSlot(slot)
	slot.AllowedItems = append(slot.AllowedItems, item)
}

func (slot *ItemSlot) SetParentItem(item *Item) {
	slot.parentItem = item
}

func (slot *ItemSlot) HasParentItem() bool {
	return slot.GetParentItem() != nil
}

// GetAncestorIds - returns all slot and item ancestors
func (slot *ItemSlot) GetAncestorIds() []string {
	parent := slot.GetParentItem()
	if parent == nil {
		return make([]string, 0)
	}

	ancestors := parent.GetAncestorIds()
	return append([]string{parent.ID}, ancestors...)
}

// GetAncestorItems - returns all ancestor AllowedItems only
func (slot *ItemSlot) GetAncestorItems() []*Item {
	if !slot.HasParentItem() {
		return make([]*Item, 0)
	}

	parentItem := slot.GetParentItem()
	if !parentItem.HasParentSlot() {
		return []*Item{parentItem}
	}

	ancestorItems := parentItem.GetParentSlot().GetAncestorItems()

	return append([]*Item{parentItem}, ancestorItems...)
}

func (slot *ItemSlot) PopulateAllowedItems(db *sql.DB) error {
	allowedItems, err := models.GetAllowedItemsBySlotID(db, slot.ID)
	if err != nil {
		return err
	}

	for i := 0; i < len(allowedItems); i++ {
		item := allowedItems[i]

		modProperties, err := models.GetWeaponModById(db, item.ID)
		if err != nil {
			return nil
		}

		allowedItem := &Item{
			ID:                 item.ID,
			Name:               item.Name,
			RecoilModifier:     modProperties.RecoilModifier,
			ErgonomicsModifier: modProperties.ErgonomicsModifier,
			Slots:              nil,
			parentSlot:         nil,
			Type:               "weapon_mod",
		}

		if slot.IsItemValidChild(allowedItem) {
			// must add first - add child maintains the parent relationship
			slot.AddChildItem(allowedItem)
			err := allowedItem.PopulateSlots(db)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to populate slot %s with item: %s", slot.ID, item.ID)
				return err
			}
		} else {
			log.Info().Msgf("Not adding item: %s to slot: %s - would result in recursion", allowedItem.ID, slot.ID)
		}
	}

	return nil
}

// IsItemValidChild
// returns true if the given item can can be added without a circular reference being created
// we want to avoid adding an item to this slot, if the same slot+item combination exists
// e.g. mount x -> sight y (with a rail) -> mount x -> sight y
//
//	or	mount x -> mount y -> mount x
func (slot *ItemSlot) IsItemValidChild(item *Item) bool {
	ancestorItems := slot.GetAncestorItems()
	for i := 0; i < len(ancestorItems); i++ {
		ancestor := ancestorItems[i]

		if !ancestor.HasParentSlot() {
			return true
		}

		if ancestor.ID == item.ID && ancestor.GetParentSlot().ID == slot.ID {
			slot.AllowedCircularReferenceItemIds = append(slot.AllowedCircularReferenceItemIds, item.ID)
			return false
		}
	}

	return true
}
