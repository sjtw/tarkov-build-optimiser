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
	GetTraderOffer(id string) ([]models.TraderOffer, error)
}

type WeaponTreeConstraints struct {
	ignoredSlotNames map[string]bool
}

type CandidateTree struct {
	Item        *Item
	dataService TreeDataProvider
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
	Constraints        models.EvaluationConstraints
}

func (wt *CandidateTree) AddItemConflicts(itemId string, conflicts []ConflictingItem) {
	if _, ok := wt.AllowedItemConflicts[itemId]; !ok {
		wt.AllowedItemConflicts[itemId] = map[string]bool{}
	}

	for _, c := range conflicts {
		wt.AllowedItemConflicts[itemId][c.ID] = true
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

func (wt *CandidateTree) SortAllowedItems(by string) {
	for _, slot := range wt.Item.Slots {
		slot.SortAllowedItems(by)
	}
}

func CreateItemCandidateTree(id string, constraints models.EvaluationConstraints, data TreeDataProvider) (*CandidateTree, error) {
	item, err := data.GetWeaponModById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon mod %s", id)
	}

	return constructCandidateTree(item.ID, item.Name, item.RecoilModifier, item.ErgonomicsModifier, constraints, data)

}

func CreateWeaponCandidateTree(id string, constraints models.EvaluationConstraints, data TreeDataProvider) (*CandidateTree, error) {
	w, err := data.GetWeaponById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}

	return constructCandidateTree(id, w.Name, w.RecoilModifier, w.ErgonomicsModifier, constraints, data)
}

func constructCandidateTree(id string, name string, recoilModifier int, ergoModifier int, constraints models.EvaluationConstraints, data TreeDataProvider) (*CandidateTree, error) {
	candidateTree := &CandidateTree{
		dataService:          data,
		AllowedItemConflicts: map[string]map[string]bool{},
		CandidateItems: map[string]bool{
			id: true,
		},
		Item:        nil,
		Constraints: constraints,
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

	candidateTree.Item = item

	err := item.PopulateSlots()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", id)
		return nil, err
	}

	item.CalculatePotentialValues()

	candidateTree.UpdateAllowedItems()
	candidateTree.updateAllowedItemsMap()
	candidateTree.UpdateAllowedItemSlots()
	candidateTree.updateAllowedItemSlotsMap()

	return candidateTree, nil
}
