package evaluator

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type Item struct {
	ID                 string      `json:"item_id" bson:"id"`
	Name               string      `json:"name" bson:"name"`
	RecoilModifier     int         `json:"recoil_modifier" bson:"recoil_modifier"`
	ErgonomicsModifier int         `json:"ergonomics_modifier" bson:"ergonomics_modifier"`
	Slots              []*ItemSlot `json:"slots" bson:"slots"`
	parentSlot         *ItemSlot
	Type               string `json:"type" bson:"type"`
}

func ConstructItem(id string, name string) *Item {
	return &Item{
		ID:                 id,
		Name:               name,
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		Slots:              make([]*ItemSlot, 0),
	}
}

func (item *Item) SetType(t string) {
	switch t {
	case "weapon":
		item.Type = "weapon"
	case "weapon_mod":
		item.Type = "weapon_mod"
	default:
		log.Error().Str("type", t).Msg("invalid item type")
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

func (item *Item) PopulateSlots(db *sql.DB) error {
	slots, err := models.GetSlotsByItemID(db, item.ID)
	if err != nil {
		return err
	}

	for i := 0; i < len(slots); i++ {
		s := slots[i]
		slot := &ItemSlot{
			ID:           s.ID,
			Name:         s.Name,
			AllowedItems: nil,
			parentItem:   nil,
		}
		item.AddChildSlot(slot)

		err := slot.PopulateAllowedItems(db)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to populate slot %s", s.ID)
			return err
		}
	}

	return nil
}
