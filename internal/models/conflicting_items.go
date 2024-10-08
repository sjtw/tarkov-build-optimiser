package models

import "database/sql"

type ConflictingItem struct {
	ItemID            string `json:"item_id"`
	ConflictingItemID string `json:"conflicting_item_id"`
}

func UpsertConflictingItems(tx *sql.Tx, itemID string, conflictingItemID string) error {
	query := `INSERT INTO conflicting_items (item_id, conflicting_item_id) values ($1, $2);`
	_, err := tx.Exec(query, itemID, conflictingItemID)
	if err != nil {
		return err
	}
	return nil
}
