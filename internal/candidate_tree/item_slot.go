package candidate_tree

import (
	"errors"
	"github.com/rs/zerolog/log"
	"slices"
)

type ItemSlot struct {
	ID              string  `json:"id" bson:"id"`
	Name            string  `json:"name" bson:"name"`
	AllowedItems    []*Item `json:"-"`
	parentItem      *Item
	PotentialValues PotentialValues `json:"potential_values"`

	// IDs of items which would potentially create a circular reference
	// we don't want to add these to the possibility tree for obvious reasons,
	// but we may still want to know what they are
	AllowedCircularReferenceItemIds []string `json:"allowed_circular_reference_item_ids"`
	RootWeaponTree                  *CandidateTree
}

func ConstructSlot(id string, name string, rootWeaponTree *CandidateTree) *ItemSlot {
	return &ItemSlot{
		ID:             id,
		Name:           name,
		RootWeaponTree: rootWeaponTree,
	}
}

func (slot *ItemSlot) GetParentItem() *Item {
	if slot.parentItem == nil {
		return nil
	}

	return slot.parentItem
}

func (slot *ItemSlot) GetDescendantAllowedItems() []*Item {
	descendants := make([]*Item, 0, len(slot.AllowedItems))
	descendants = append(descendants, slot.AllowedItems...)
	if slot.AllowedItems == nil {
		return make([]*Item, 0)
	}
	for _, ai := range slot.AllowedItems {
		aiSlots := ai.GetDescendantSlots()
		for _, s := range aiSlots {
			descendants = append(descendants, s.GetDescendantAllowedItems()...)
		}
	}

	return descendants
}

func (slot *ItemSlot) AddAllowedItem(item *Item) {
	item.SetParentSlot(slot)
	slot.AllowedItems = append(slot.AllowedItems, item)
}

func (slot *ItemSlot) SortAllowedItems(by string) {
	if slot.AllowedItems == nil || len(slot.AllowedItems) == 0 {
		return
	}

	for _, item := range slot.AllowedItems {
		if item.Slots == nil || len(item.Slots) == 0 {
			continue
		}

		for _, slot := range item.Slots {
			slot.SortAllowedItems(by)
		}
	}

	slices.SortFunc(slot.AllowedItems, func(i, j *Item) int {
		switch by {
		case "recoil-min":
			if i.PotentialValues.MinRecoil < j.PotentialValues.MinRecoil {
				return -1
			} else if i.PotentialValues.MinRecoil > j.PotentialValues.MinRecoil {
				return 1
			}
		case "recoil-max":
			if i.PotentialValues.MaxRecoil < j.PotentialValues.MaxRecoil {
				return -1
			} else if i.PotentialValues.MaxRecoil > j.PotentialValues.MaxRecoil {
				return 1
			}
		case "ergonomics-min":
			if i.PotentialValues.MinErgonomics < j.PotentialValues.MinErgonomics {
				return -1
			} else if i.PotentialValues.MinErgonomics > j.PotentialValues.MinErgonomics {
				return 1
			}
		case "ergonomics-max":
			if i.PotentialValues.MaxErgonomics < j.PotentialValues.MaxErgonomics {
				return -1
			} else if i.PotentialValues.MaxErgonomics > j.PotentialValues.MaxErgonomics {
				return 1
			}
		}
		return 0
	})
}

func (slot *ItemSlot) CalculatePotentialValues() {
	slot.PotentialValues = PotentialValues{}

	if slot.AllowedItems == nil || len(slot.AllowedItems) == 0 {
		return
	}

	for _, item := range slot.AllowedItems {
		item.CalculatePotentialValues()

		if item.PotentialValues.MinRecoil < slot.PotentialValues.MinRecoil {
			slot.PotentialValues.MinRecoil = item.PotentialValues.MinRecoil
		}
		if item.PotentialValues.MaxRecoil > slot.PotentialValues.MaxRecoil {
			slot.PotentialValues.MaxRecoil = item.PotentialValues.MaxRecoil
		}
		if item.PotentialValues.MinErgonomics < slot.PotentialValues.MinErgonomics {
			slot.PotentialValues.MinErgonomics = item.PotentialValues.MinErgonomics
		}
		if item.PotentialValues.MaxErgonomics > slot.PotentialValues.MaxErgonomics {
			slot.PotentialValues.MaxErgonomics = item.PotentialValues.MaxErgonomics
		}
	}
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

// pruneUselessAllowedItems - removes allowed items which definitely have no potential value improvement
// we're assuming potential values have already been calculated and have been sorted by value
//
// we basically see if any non-conflicting allowed item has the best value. If it does we can always use it so can
// just drop everything else
func (slot *ItemSlot) pruneUselessAllowedItems() {
	if slot.Name == "Rear Sight" {
		log.Debug().Msgf("Pruning useless allowed items for slot: %s", slot.Name)
	}

	if len(slot.AllowedItems) == 0 {
		return
	}

	conflictingItems := make([]*Item, 0)
	bestNonConflictingValue := 0
	var bestNonConflicting *Item

	for _, item := range slot.AllowedItems {
		item.pruneUselessAllowedItems()

		// hardcode for recoil for now.
		// TODO: use the roots constraints to determine which stat to use
		if item.ConflictingItems == nil || len(item.ConflictingItems) == 0 {
			// has no conflicts
			if item.PotentialValues.MinRecoil < bestNonConflictingValue {
				bestNonConflictingValue = item.PotentialValues.MinRecoil
				bestNonConflicting = item
			}
		} else {
			// has conflicts
			if item.PotentialValues.MinRecoil < 0 {
				conflictingItems = append(conflictingItems, item)
			}
		}
	}

	if len(conflictingItems) > 0 {
		if bestNonConflicting != nil && len(conflictingItems) > 0 {
			if bestNonConflicting.PotentialValues.MinRecoil < conflictingItems[0].PotentialValues.MinRecoil {
				// best non conflicting item is better than any conflicting item
				slot.AllowedItems = []*Item{bestNonConflicting}
			} else {
				//	it's not the best, but it might be better than some
				newAllowedItems := make([]*Item, 0)
				for _, conflictingItem := range conflictingItems {
					if conflictingItem.PotentialValues.MinRecoil < bestNonConflictingValue {
						newAllowedItems = append(newAllowedItems, conflictingItem)
					} else {
						newAllowedItems = append(newAllowedItems, bestNonConflicting)
						// everything onwards is worse than the best non-conflicting item
						// we can prune everything else.
						break
					}
				}
			}
		} else {
			// no non-conflicting items, just keep the best conflicting item
			slot.AllowedItems = conflictingItems
		}
	} else if bestNonConflicting != nil {
		// no conflicting items, just keep the best non-conflicting item
		slot.AllowedItems = []*Item{bestNonConflicting}
	} else {
		// no conflicting or non-conflicting items
		slot.AllowedItems = make([]*Item, 0)
	}

	// now we're done, ensure the best item is at the front of allowed items, incase we changed the ordering
	slot.SortAllowedItems("recoil-min")
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

func (slot *ItemSlot) PopulateAllowedItems() error {
	allowedItems, err := slot.RootWeaponTree.dataService.GetAllowedItemsBySlotID(slot.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get allowed items for slot %s", slot.ID)
		return err
	}

	for i := 0; i < len(allowedItems); i++ {
		allowedItem := allowedItems[i]

		traderOfferValid := false
		offer, err := slot.RootWeaponTree.dataService.GetTraderOffer(allowedItem.ID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get trader offer for item %s", allowedItem.ID)
			return err
		}
		for _, traderConstraint := range slot.RootWeaponTree.Constraints.TraderLevels {
			for _, t := range offer {
				if traderConstraint.Name == t.Trader {
					if traderConstraint.Level >= t.MinTraderLevel {
						traderOfferValid = true
						break
					}
				}
			}
			if traderOfferValid {
				break
			}
		}

		if !traderOfferValid {
			//log.Info().Msgf("item %s does not meet trader level constraints - not adding", allowedItem.ID)
			continue
		}

		ignored := false
		for _, id := range slot.RootWeaponTree.Constraints.IgnoredItemIDs {
			if id == allowedItem.ID {
				//log.Info().Msgf("item %s is ignored - not adding as allowed item", id)
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		modProperties, err := slot.RootWeaponTree.dataService.GetWeaponModById(allowedItem.ID)
		if err != nil {
			return nil
		}

		if modProperties == nil {
			continue
		}

		item := ConstructItem(allowedItem.ID, allowedItem.Name, slot.RootWeaponTree)
		item.RecoilModifier = modProperties.RecoilModifier
		item.ErgonomicsModifier = modProperties.ErgonomicsModifier
		item.CategoryID = modProperties.CategoryID
		item.CategoryName = modProperties.CategoryName
		item.Type = "weapon_mod"
		item.ConflictingItems = make([]ConflictingItem, 0)

		if len(modProperties.ConflictingItems) > 0 {
			for _, id := range modProperties.ConflictingItems {
				conflictingItem, err := slot.RootWeaponTree.dataService.GetWeaponModById(id)
				if err != nil || conflictingItem == nil {
					isWeapon, err := slot.RootWeaponTree.dataService.IsWeapon(id)
					if err != nil {
						log.Error().Err(err).Msgf("Failed to resolve conflicting item: %s. IsWeapon also failed.", id)
						return err
					}

					if !isWeapon {
						log.Error().Msgf("Failed to resolve conflicting item, Conflicting item %s is not a weapon either.", id)
						return errors.New("Failed to resolve conflicting item.")
					}

					if id == slot.RootWeaponTree.Item.ID {
						// the conflicting item is the weapon itself - skip it
						continue
					}

					// the conflicting item is a weapon, and isn't this one - all good
					continue
				}
				item.ConflictingItems = append(item.ConflictingItems, ConflictingItem{
					ID:           conflictingItem.ID,
					Name:         conflictingItem.Name,
					CategoryID:   conflictingItem.CategoryID,
					CategoryName: conflictingItem.CategoryName,
				})
			}
		}

		if slot.IsItemValidChild(item) {
			// must add first - add child maintains the parent relationship
			slot.AddAllowedItem(item)

			if len(item.ConflictingItems) > 0 {
				slot.RootWeaponTree.AddItemConflicts(item.ID, item.ConflictingItems)
			}
			slot.RootWeaponTree.AddCandidateItem(item.ID)

			err := item.PopulateSlots()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to populate slot %s with item: %s", slot.ID, item.ID)
				return err
			}
		} else {
			log.Debug().Msgf("Not adding item: %s to slot: %s - would result in recursion", allowedItem.ID, slot.ID)
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
