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

// GenerateNonConflictingCandidateSets - generates all maximal, non-conflicting sets of candidate item IDs given
// the candidate list and conflict maps.
func GenerateNonConflictingCandidateSets(candidates map[string]bool, conflicts map[string]map[string]bool) [][]string {
	// Some items do not have symmetrical conflicts (for example pistol grips with integrated buttstocks conflict
	// with most stocks, however there is no conflict in the other direction. By removing all conflicts are symmetrical
	// up-front we never need to be concerned with the order items are checked/added to a build.
	symmetricConflicts := make(map[string]map[string]bool)
	for candidate, conflictSet := range conflicts {
		if _, exists := symmetricConflicts[candidate]; !exists {
			symmetricConflicts[candidate] = make(map[string]bool)
		}

		for conflict := range conflictSet {
			if _, exists := symmetricConflicts[conflict]; !exists {
				symmetricConflicts[conflict] = make(map[string]bool)
			}
			symmetricConflicts[candidate][conflict] = true
			symmetricConflicts[conflict][candidate] = true
		}
	}

	result := [][]string{}

	conflictsWithSet := func(candidate string, currentSet []string) bool {
		for _, member := range currentSet {
			if symmetricConflicts[candidate] != nil && symmetricConflicts[candidate][member] {
				return true
			}
		}
		return false
	}

	conflictFree := []string{}
	conflictingCandidates := []string{}
	for candidate := range candidates {
		if len(symmetricConflicts[candidate]) == 0 {
			conflictFree = append(conflictFree, candidate)
		} else {
			conflictingCandidates = append(conflictingCandidates, candidate)
		}
	}

	remaining := conflictingCandidates

	for len(remaining) > 0 {
		currentSet := []string{}
		newRemaining := []string{}

		for _, candidate := range remaining {
			if !conflictsWithSet(candidate, currentSet) {
				currentSet = append(currentSet, candidate)
			} else {
				newRemaining = append(newRemaining, candidate)
			}
		}

		for _, candidate := range conflictFree {
			if !conflictsWithSet(candidate, currentSet) {
				currentSet = append(currentSet, candidate)
			}
		}

		result = append(result, currentSet)
		remaining = newRemaining
	}

	return result
}
