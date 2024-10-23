package evaluator

import (
	"github.com/rs/zerolog/log"
)

type Item struct {
	ID                 string      `json:"item_id" bson:"id"`
	Name               string      `json:"name" bson:"name"`
	RecoilModifier     int         `json:"recoil_modifier" bson:"recoil_modifier"`
	ErgonomicsModifier int         `json:"ergonomics_modifier" bson:"ergonomics_modifier"`
	Slots              []*ItemSlot `json:"slots" bson:"slots"`
	Type               string      `json:"type" bson:"type"`
	ConflictingItems   []string    `json:"conflicting_items" bson:"conflicting_items"`
	parentSlot         *ItemSlot
	RootItem           *Item
	RootWeaponTree     *WeaponTree
}

func ConstructItem(id string, name string, rootWeaponTree *WeaponTree) *Item {
	return &Item{
		ID:               id,
		Name:             name,
		Slots:            make([]*ItemSlot, 0),
		RootWeaponTree:   rootWeaponTree,
		ConflictingItems: make([]string, 0),
	}
}

func (item *Item) HasParentSlot() bool {
	return item.parentSlot != nil
}

func (item *Item) GetParentSlot() *ItemSlot {
	if item.parentSlot == nil {
		return nil
	}

	return item.parentSlot
}

func (item *Item) AddChildSlot(slot *ItemSlot) {
	slot.SetParentItem(item)
	item.Slots = append(item.Slots, slot)
}

func (item *Item) SetParentSlot(slot *ItemSlot) {
	item.parentSlot = slot
}

func (item *Item) GetAncestorIds() []string {
	parent := item.GetParentSlot()
	if parent == nil {
		return make([]string, 0)
	}

	ancestors := parent.GetAncestorIds()
	return append([]string{parent.ID}, ancestors...)
}

func (item *Item) PopulateSlots() error {
	slots, err := item.RootWeaponTree.dataService.GetSlotsByItemID(item.ID)
	if err != nil {
		return err
	}

	for i := 0; i < len(slots); i++ {
		s := slots[i]
		slot := ConstructSlot(s.ID, s.Name, item.RootWeaponTree)

		item.AddChildSlot(slot)

		err := slot.PopulateAllowedItems()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to populate slot %s", s.ID)
			return err
		}
	}

	return nil
}
