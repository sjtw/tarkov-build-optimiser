package candidate_tree

import (
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

type CandidateTree struct {
	Item        *Item
	dataService TreeDataProvider
	constraints WeaponTreeConstraints
	// all itemIDs which conflict globally with other itemIDs
	AllowedItemConflicts map[string]map[string]bool
	// all candidate items for this weapon
	CandidateItems map[string]bool
	// all allowed items in the tree
	allowedItems   []*Item
	allowedItemMap map[string]*Item
	// all slots of all allowed items in the tree
	allowedItemSlots   []*ItemSlot
	allowedItemSlotMap map[string]*ItemSlot
}

func (wt *CandidateTree) AddItemConflicts(itemId string, conflictIDs []string) {
	if _, ok := wt.AllowedItemConflicts[itemId]; !ok {
		wt.AllowedItemConflicts[itemId] = map[string]bool{}
	}

	for _, conflictId := range conflictIDs {
		wt.AllowedItemConflicts[itemId][conflictId] = true
	}
}

func (wt *CandidateTree) AddCandidateItem(itemID string) {
	wt.CandidateItems[itemID] = true
}

// UpdateAllowedItemSlots updates allowedItemSlots with the current state of the tree
func (wt *CandidateTree) UpdateAllowedItemSlots() {
	slots := wt.Item.GetDescendantSlots()
	wt.allowedItemSlots = slots
	wt.updateAllowedItemSlotsMap()
}

func (wt *CandidateTree) updateAllowedItemSlotsMap() {
	slotMap := map[string]*ItemSlot{}
	for _, slot := range wt.allowedItemSlots {
		slotMap[slot.ID] = slot
	}
	wt.allowedItemSlotMap = slotMap
}

// UpdateAllowedItems updates allowedItems with the current state of the tree
func (wt *CandidateTree) UpdateAllowedItems() {
	allowedItems := make([]*Item, 0)
	for _, slot := range wt.Item.Slots {
		items := slot.GetDescendantAllowedItems()
		allowedItems = append(allowedItems, items...)
	}
	wt.allowedItems = allowedItems
	wt.updateAllowedItemsMap()
}

func (wt *CandidateTree) GetAllowedItem(id string) *Item {
	return wt.allowedItemMap[id]
}

func (wt *CandidateTree) updateAllowedItemsMap() {
	itemMap := map[string]*Item{}
	for _, item := range wt.allowedItems {
		itemMap[item.ID] = item
	}
	wt.allowedItemMap = itemMap
}

func (wt *CandidateTree) GetAllowedItemSlot(id string) *ItemSlot {
	return wt.allowedItemSlotMap[id]
}

func CreateItemCandidateTree(id string, data TreeDataProvider) (*CandidateTree, error) {
	item, err := data.GetWeaponModById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon mod %s", id)
	}

	return constructCandidateTree(item.ID, item.Name, item.RecoilModifier, item.ErgonomicsModifier, data)
}

func CreateWeaponCandidateTree(id string, data TreeDataProvider) (*CandidateTree, error) {
	w, err := data.GetWeaponById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}

	return constructCandidateTree(id, w.Name, w.RecoilModifier, w.ErgonomicsModifier, data)
}

func constructCandidateTree(id string, name string, recoilModifier int, ergoModifier int, data TreeDataProvider) (*CandidateTree, error) {
	candidateTree := &CandidateTree{
		dataService:          data,
		AllowedItemConflicts: map[string]map[string]bool{},
		CandidateItems: map[string]bool{
			id: true,
		},
		Item: nil,
	}

	item := &Item{
		ID:                 id,
		Name:               name,
		RecoilModifier:     recoilModifier,
		ErgonomicsModifier: ergoModifier,
		Slots:              []*ItemSlot{},
		parentSlot:         nil,
		Type:               "weapon",
		Root:               candidateTree,
	}

	err := item.PopulateSlots()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", id)
		return nil, err
	}

	candidateTree.Item = item

	return candidateTree, nil
}
