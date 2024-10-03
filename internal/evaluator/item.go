package evaluator

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

type Item struct {
	ID                 string      `json:"item_id" bson:"id"`
	Name               string      `json:"name" bson:"name"`
	RecoilModifier     int         `json:"recoil_modifier" bson:"recoil_modifier"`
	ErgonomicsModifier int         `json:"ergonomics_modifier" bson:"ergonomics_modifier"`
	Slots              []*ItemSlot `json:"slots" bson:"slots"`
	Type               string      `json:"type" bson:"type"`
	parentSlot         *ItemSlot
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

func (item *Item) GetBestRecoilSum() int {
	sum := item.RecoilModifier
	for i := 0; i < len(item.Slots); i++ {
		sum += item.Slots[i].BestRecoilModifier
	}

	return sum
}

func (item *Item) GetBestErgoSum() int {
	sum := item.ErgonomicsModifier
	for i := 0; i < len(item.Slots); i++ {
		sum += item.Slots[i].BestErgoModifier
	}

	return sum
}

func (item *Item) Save(db *sql.DB) error {
	b, err := json.Marshal(item)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal build")
		return err
	}

	query := `INSERT INTO optimum_builds (
			item_id,
			build,
			build_type,
            modifier_sum,
            name
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (item_id, build_type) DO UPDATE SET
			build = $2,
			modifier_sum = $4,
			name = $5
		;`

	recoilSum := item.GetBestRecoilSum()
	_, err = db.Exec(query, item.ID, b, "recoil", recoilSum, item.Name)
	if err != nil {
		return err
	}

	ergoSum := item.GetBestErgoSum()
	_, err = db.Exec(query, item.ID, b, "ergonomics", ergoSum, item.Name)
	if err != nil {
		return err
	}

	return nil
}
