package models

import "database/sql"

type ConflictingItem struct {
	ItemID            string `json:"item_id"`
	ConflictingItemID string `json:"conflicting_item_id"`
}

func upsertConflictingItem(tx *sql.Tx, itemID string, conflictingItemID string) error {
	query := `INSERT INTO conflicting_items (
	   item_id,
	   conflicting_item_id
   ) VALUES ($1, $2)
     ON CONFLICT (item_id, conflicting_item_id) DO NOTHING;`
	_, err := tx.Exec(
		query,
		itemID,
		conflictingItemID,
	)
	if err != nil {
		return err
	}

	// bit of a hack to ensure all item conflicts are bidirectional as some aren't for some reason
	_, err = tx.Exec(
		query,
		conflictingItemID,
		itemID,
	)
	if err != nil {
		return err
	}

	return nil
}

func upsertManyConflictingItems(tx *sql.Tx, itemID string, conflictingItemIDs []string) error {
	for i := 0; i < len(conflictingItemIDs); i++ {
		err := upsertConflictingItem(tx, itemID, conflictingItemIDs[i])
		if err != nil {
			return err
		}
	}
	return nil
}
