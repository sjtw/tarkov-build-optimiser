package models

import "database/sql"

type AllowedItem struct {
	ID   string `json:"item_id"`
	Name string `json:"name"`
}

func upsertAllowedItem(tx *sql.Tx, slotID string, allowedItem AllowedItem) error {
	query := `
		INSERT INTO slot_allowed_items (
			slot_id,
			item_id,
			name
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (slot_id, item_id) DO UPDATE SET
			name = $3
		;`
	_, err := tx.Exec(query, slotID, allowedItem.ID, allowedItem.Name)
	if err != nil {
		return err
	}

	return nil
}

func upsertManyAllowedItem(tx *sql.Tx, slotId string, allowedItems []AllowedItem) error {
	for i := 0; i < len(allowedItems); i++ {
		err := upsertAllowedItem(tx, slotId, allowedItems[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func GetAllowedItemsBySlotID(db *sql.DB, slotID string) ([]*AllowedItem, error) {
	rows, err := db.Query(`select item_id, name FROM slot_allowed_items where slot_id = $1;`, slotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allowedItems := make([]*AllowedItem, 0)
	for rows.Next() {
		slot := &AllowedItem{}
		err := rows.Scan(&slot.ID, &slot.Name)
		if err != nil {
			return nil, err
		}
		allowedItems = append(allowedItems, slot)
	}

	return allowedItems, nil
}
