package evaluator

import (
	"database/sql"
	"fmt"
	"sync"
	"tarkov-build-optimiser/internal/models"
)

type DataService struct {
	db                       *sql.DB
	weaponModCache           map[string]*models.WeaponMod
	modMu                    sync.Mutex
	evalResultCache          map[string]*models.ItemEvaluationResult
	evalResultMu             sync.Mutex
	allowedItemBySlotIDCache map[string][]*models.AllowedItem
	allowedItemBySlotIDMu    sync.Mutex
}

func CreateDataService(db *sql.DB) *DataService {
	return &DataService{
		db:                       db,
		weaponModCache:           make(map[string]*models.WeaponMod),
		evalResultCache:          make(map[string]*models.ItemEvaluationResult),
		allowedItemBySlotIDCache: make(map[string][]*models.AllowedItem),
	}
}

func (tds *DataService) GetWeaponById(id string) (*models.Weapon, error) {
	return models.GetWeaponById(tds.db, id)
}

func (tds *DataService) GetSlotsByItemID(id string) ([]models.Slot, error) {
	return models.GetSlotsByItemID(tds.db, id)
}

func (tds *DataService) GetWeaponModById(id string) (*models.WeaponMod, error) {
	tds.modMu.Lock()
	mod, ok := tds.weaponModCache[id]
	tds.modMu.Unlock()
	if ok && mod != nil {
		return mod, nil
	}

	mods, err := models.GetAllWeaponMods(tds.db)
	if err != nil {
		return nil, err
	}

	tds.modMu.Lock()
	for _, m := range mods {
		if m.ID == id {
			mod = m
		}
		tds.weaponModCache[m.ID] = m
	}
	tds.modMu.Unlock()

	return mod, nil
}

func (tds *DataService) GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error) {
	tds.allowedItemBySlotIDMu.Lock()
	items, ok := tds.allowedItemBySlotIDCache[id]
	tds.allowedItemBySlotIDMu.Unlock()
	if ok && items != nil {
		return items, nil
	}

	allItems, err := models.GetAllAllowedItems(tds.db)
	if err != nil {
		return nil, err
	}

	tds.allowedItemBySlotIDMu.Lock()
	tds.allowedItemBySlotIDCache = allItems
	tds.allowedItemBySlotIDMu.Unlock()

	return allItems[id], nil
}

func serialiseBuildKey(itemId string, buildType string, constraints models.EvaluationConstraints) string {
	serialisedTraderConstraints := models.SerialiseLevels(constraints.TraderLevels)
	return fmt.Sprintf("%s-%s-%s", itemId, buildType, serialisedTraderConstraints)
}

func (tds *DataService) SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) error {
	key := serialiseBuildKey(build.ID, build.EvaluationType, constraints)
	tds.evalResultMu.Lock()
	tds.evalResultCache[key] = build
	tds.evalResultMu.Unlock()

	return models.UpsertOptimumBuild(tds.db, build, constraints)
}

func (tds *DataService) GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	key := serialiseBuildKey(itemId, buildType, constraints)
	tds.evalResultMu.Lock()
	build, ok := tds.evalResultCache[key]
	tds.evalResultMu.Unlock()
	if ok && build != nil {
		return build, nil
	}

	subtree, err := models.GetEvaluatedSubtree(tds.db, itemId, buildType, constraints)
	if err != nil {
		return nil, err
	}

	if subtree != nil {
		tds.evalResultMu.Lock()
		tds.evalResultCache[key] = subtree
		tds.evalResultMu.Unlock()
	}

	return subtree, nil
}

func (tds *DataService) GetTraderOffer(itemID string) ([]models.TraderOffer, error) {
	offers, err := models.GetTraderOffersByItemID(tds.db, itemID)
	if err != nil {
		return nil, err
	}
	return offers, nil
}
