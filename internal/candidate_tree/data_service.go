package candidate_tree

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
	"tarkov-build-optimiser/internal/models"
)

type DataService struct {
	db                       *sql.DB
	weaponModCache           map[string]*models.WeaponMod
	slotCache                map[string][]models.Slot
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
		slotCache:                make(map[string][]models.Slot),
	}
}

func (tds *DataService) GetWeaponById(id string) (*models.Weapon, error) {
	return models.GetWeaponById(tds.db, id)
}

func (tds *DataService) GetSlotsByItemID(id string) ([]models.Slot, error) {
	tds.modMu.Lock()
	slots, ok := tds.slotCache[id]
	tds.modMu.Unlock()
	if ok {
		return slots, nil
	}

	err := tds.loadAllSlots()
	if err != nil {
		return nil, err
	}

	tds.modMu.Lock()
	slots, ok = tds.slotCache[id]
	tds.modMu.Unlock()
	if !ok {
		return []models.Slot{}, nil
	}

	return slots, nil
}

func (tds *DataService) GetWeaponModById(id string) (*models.WeaponMod, error) {
	tds.modMu.Lock()
	mod, ok := tds.weaponModCache[id]
	tds.modMu.Unlock()
	if ok {
		return mod, nil
	}

	err := tds.loadAllWeaponMods()
	if err != nil {
		return nil, err
	}

	tds.modMu.Lock()
	mod, ok = tds.weaponModCache[id]
	tds.modMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("weapon mod with id %s not found", id)
	}

	return mod, nil
}

func (tds *DataService) GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error) {
	tds.allowedItemBySlotIDMu.Lock()
	allowedItems, ok := tds.allowedItemBySlotIDCache[id]
	tds.allowedItemBySlotIDMu.Unlock()
	if ok {
		return allowedItems, nil
	}

	allAllowedItems, err := models.GetAllAllowedItems(tds.db)
	if err != nil {
		log.Err(err).Msg("Failed to get all allowed items")
		return nil, err
	}

	tds.allowedItemBySlotIDMu.Lock()
	tds.allowedItemBySlotIDCache = allAllowedItems
	tds.allowedItemBySlotIDMu.Unlock()

	allowedItems, ok = tds.allowedItemBySlotIDCache[id]
	if !ok {
		return []*models.AllowedItem{}, nil
	}

	return allowedItems, nil
}

func serialiseBuildKey(itemId string, buildType string, constraints models.EvaluationConstraints) string {
	serialisedTraderConstraints := models.SerialiseLevels(constraints.TraderLevels)
	return fmt.Sprintf("%s-%s-%s", itemId, buildType, serialisedTraderConstraints)
}

//func (tds *DataService) SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) {
//	key := serialiseBuildKey(build.ID, build.EvaluationType, constraints)
//	tds.evalResultMu.Lock()
//	tds.evalResultCache[key] = build
//	tds.evalResultMu.Unlock()
//
//	// We don't want to block the main evaluation thread, so we'll save the build in a goroutine
//	// I'm sure it can be done more elegantly but rn if it fails it's not the end of the world
//	// we just have to calculate it again sometime...
//	go func() {
//		err := models.UpsertOptimumBuild(tds.db, build, constraints, )
//		if err != nil {
//			log.Warn().Err(err).Msgf("Failed to save build: %v", build)
//		}
//	}()
//}

func (tds *DataService) GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	key := serialiseBuildKey(itemId, buildType, constraints)
	tds.evalResultMu.Lock()
	build, ok := tds.evalResultCache[key]
	tds.evalResultMu.Unlock()
	if ok && build != nil {
		return build, nil
	}

	ctx := context.Background()
	subtree, err := models.GetEvaluatedSubtree(ctx, tds.db, itemId, buildType, constraints)
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

func (tds *DataService) loadAllWeaponMods() error {
	mods, err := models.GetAllWeaponMods(tds.db)
	if err != nil {
		return err
	}

	tds.modMu.Lock()
	defer tds.modMu.Unlock()
	for _, mod := range mods {
		tds.weaponModCache[mod.ID] = mod
	}

	return nil
}

func (tds *DataService) loadAllSlots() error {
	slots, err := models.GetAllSlots(tds.db)
	if err != nil {
		return err
	}

	tds.modMu.Lock()
	defer tds.modMu.Unlock()
	for itemID, itemSlots := range slots {
		tds.slotCache[itemID] = itemSlots
	}

	return nil
}
