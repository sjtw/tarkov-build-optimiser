package candidate_tree

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"tarkov-build-optimiser/internal/models"

	"github.com/rs/zerolog/log"
)

type DataService struct {
	db                       *sql.DB
	weaponModCache           map[string]*models.WeaponMod
	slotCache                map[string][]models.Slot
	modMu                    sync.Mutex
	allowedItemBySlotIDCache map[string][]*models.AllowedItem
	allowedItemBySlotIDMu    sync.Mutex
	priceCache               map[string]int
	priceCacheMu             sync.RWMutex
}

func CreateDataService(db *sql.DB) *DataService {
	return &DataService{
		db:                       db,
		weaponModCache:           make(map[string]*models.WeaponMod),
		allowedItemBySlotIDCache: make(map[string][]*models.AllowedItem),
		slotCache:                make(map[string][]models.Slot),
		priceCache:               make(map[string]int),
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

func (tds *DataService) IsWeapon(id string) (bool, error) {
	return models.IsWeapon(tds.db, id)
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

func (tds *DataService) GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	ctx := context.Background()
	subtree, err := models.GetEvaluatedSubtree(ctx, tds.db, itemId, buildType, constraints)
	if err != nil {
		return nil, err
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

func (tds *DataService) GetItemPrice(ctx context.Context, itemID string, traderLevels []models.TraderLevel) (int, bool, error) {
	key := fmt.Sprintf("%s|%s", itemID, models.SerialiseLevels(traderLevels))

	tds.priceCacheMu.RLock()
	if price, ok := tds.priceCache[key]; ok {
		tds.priceCacheMu.RUnlock()
		return price, true, nil
	}
	tds.priceCacheMu.RUnlock()

	price, ok, err := models.GetLowestPriceForItem(ctx, tds.db, itemID, traderLevels)
	if err != nil {
		return 0, false, err
	}
	if ok {
		tds.priceCacheMu.Lock()
		tds.priceCache[key] = price
		tds.priceCacheMu.Unlock()
	}

	return price, ok, nil
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
