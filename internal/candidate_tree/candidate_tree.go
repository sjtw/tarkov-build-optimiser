package candidate_tree

import (
	"tarkov-build-optimiser/internal/models"

	"github.com/rs/zerolog/log"
)

type TreeDataProvider interface {
	GetWeaponById(id string) (*models.Weapon, error)
	GetSlotsByItemID(id string) ([]models.Slot, error)
	GetWeaponModById(id string) (*models.WeaponMod, error)
	GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error)
	GetTraderOffer(id string) ([]models.TraderOffer, error)
	IsWeapon(id string) (bool, error)
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

// GetPrecomputedProvider exposes a precomputed subtree provider if the underlying dataService implements it.
func (wt *CandidateTree) GetPrecomputedProvider() PrecomputedSubtreeProvider {
	if wt == nil {
		return nil
	}
	if provider, ok := any(wt.dataService).(PrecomputedSubtreeProvider); ok {
		return provider
	}
	return nil
}

// SetDataService sets the underlying data service for this candidate tree.
func (wt *CandidateTree) SetDataService(ds TreeDataProvider) {
	if wt == nil {
		return
	}
	wt.dataService = ds
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

// pruneUselessAlowedItems removes alloweditems which definitely have no potential value improvement
func (wt *CandidateTree) pruneUselessAllowedItems() {
	for _, slot := range wt.Item.Slots {
		slot.pruneUselessAllowedItems()
	}
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

func CreateWeaponCandidateTree(id string, focusedStat string, constraints models.EvaluationConstraints, data TreeDataProvider) (*CandidateTree, error) {
	w, err := data.GetWeaponById(id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}

	return constructCandidateTree(id, w.Name, w.RecoilModifier, w.ErgonomicsModifier, focusedStat, constraints, data)
}

func constructCandidateTree(id string, name string, recoilModifier int, ergoModifier int, focusedStat string, constraints models.EvaluationConstraints, data TreeDataProvider) (*CandidateTree, error) {
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
	candidateTree.SortAllowedItems(focusedStat)
	candidateTree.pruneUselessAllowedItems()

	// Hook: apply precomputed subtree pruning if dataService implements PrecomputedSubtreeProvider
	if provider, ok := any(candidateTree.dataService).(PrecomputedSubtreeProvider); ok {
		ApplyPrecomputedPruning(candidateTree, focusedStat, provider)
	}

	candidateTree.UpdateAllowedItems()
	candidateTree.updateAllowedItemsMap()
	candidateTree.UpdateAllowedItemSlots()
	candidateTree.updateAllowedItemSlotsMap()

	return candidateTree, nil
}
