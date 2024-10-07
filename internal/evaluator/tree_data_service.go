package evaluator

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"
)

type TreeDataService struct {
	db *sql.DB
}

func CreateTreeDataService(db *sql.DB) *TreeDataService {
	return &TreeDataService{
		db: db,
	}
}

func (tds *TreeDataService) GetWeaponById(id string) (*models.Weapon, error) {
	return models.GetWeaponById(tds.db, id)
}

func (tds *TreeDataService) GetSlotsByItemID(id string) ([]models.Slot, error) {
	return models.GetSlotsByItemID(tds.db, id)
}

func (tds *TreeDataService) GetWeaponModById(id string) (*models.WeaponMod, error) {
	return models.GetWeaponModById(tds.db, id)
}

func (tds *TreeDataService) GetAllowedItemsBySlotID(id string) ([]*models.AllowedItem, error) {
	return models.GetAllowedItemsBySlotID(tds.db, id)
}
