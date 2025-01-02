package candidate_tree

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

type ConflictingItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

type PotentialValues struct {
	MinRecoil     int `json:"min_recoil"`
	MaxRecoil     int `json:"max_recoil"`
	AvgRecoil     int `json:"avg_recoil"`
	MinErgonomics int `json:"min_ergonomics"`
	MaxErgonomics int `json:"max_ergonomics"`
	AvgErgonomics int `json:"avg_ergonomics"`
}

type Item struct {
	ID                 string            `json:"item_id" bson:"id"`
	Name               string            `json:"name" bson:"name"`
	RecoilModifier     int               `json:"recoil_modifier" bson:"recoil_modifier"`
	ErgonomicsModifier int               `json:"ergonomics_modifier" bson:"ergonomics_modifier"`
	Slots              []*ItemSlot       `json:"slots" bson:"slots"`
	Type               string            `json:"type" bson:"type"`
	ConflictingItems   []ConflictingItem `json:"conflicting_items" bson:"conflicting_items"`
	CategoryName       string            `json:"category_name"`
	CategoryID         string            `json:"category_id"`
	parentSlot         *ItemSlot
	RootItem           *Item
	Root               *CandidateTree
	PotentialValues    PotentialValues `json:"potential_values"`
}

func ConstructItem(id string, name string, rootWeaponTree *CandidateTree) *Item {
	return &Item{
		ID:               id,
		Name:             name,
		Slots:            make([]*ItemSlot, 0),
		Root:             rootWeaponTree,
		ConflictingItems: make([]ConflictingItem, 0),
		PotentialValues:  PotentialValues{},
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

func (item *Item) GetDescendantSlots() []*ItemSlot {
	if item.Slots == nil {
		return make([]*ItemSlot, 0)
	}
	descendants := make([]*ItemSlot, 0, len(item.Slots))
	descendants = append(descendants, item.Slots...)
	for _, childSlots := range item.Slots {
		items := childSlots.GetDescendantAllowedItems()
		for _, item := range items {
			if item == nil {
				fmt.Println("ops")
			}
			descendants = append(descendants, item.GetDescendantSlots()...)
		}
	}

	return descendants
}

func (item *Item) pruneUselessAllowedItems() {
	for _, slot := range item.Slots {
		slot.pruneUselessAllowedItems()
	}
}

func (item *Item) CalculatePotentialValues() {
	item.PotentialValues = PotentialValues{
		MinRecoil:     item.RecoilModifier,
		MaxRecoil:     item.RecoilModifier,
		MinErgonomics: item.ErgonomicsModifier,
		MaxErgonomics: item.ErgonomicsModifier,
	}

	if item.Slots != nil {
		for _, slot := range item.Slots {
			slot.CalculatePotentialValues()

			item.PotentialValues.MinRecoil += slot.PotentialValues.MinRecoil
			item.PotentialValues.MaxRecoil += slot.PotentialValues.MaxRecoil
			item.PotentialValues.MinErgonomics += slot.PotentialValues.MinErgonomics
			item.PotentialValues.MaxErgonomics += slot.PotentialValues.MaxErgonomics
		}
	}
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
	slots, err := item.Root.dataService.GetSlotsByItemID(item.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get slots for item %s", item.ID)
		return err
	}

	for i := 0; i < len(slots); i++ {

		s := slots[i]
		slot := ConstructSlot(s.ID, s.Name, item.Root)

		item.AddChildSlot(slot)

		ignored := false
		for _, name := range item.Root.Constraints.IgnoredSlotNames {
			//log.Info().Msgf("slot %s is ignored - not populating with allowed items", name)
			if slots[i].Name == name {
				ignored = true
				break
			}
		}

		if ignored {
			continue
		}

		err := slot.PopulateAllowedItems()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to populate slot %s", s.ID)
			return err
		}
	}

	return nil
}
