package models

import "database/sql"

type Slot struct {
	ID           string        `json:"slot_id"`
	Name         string        `json:"name"`
	AllowedItems []AllowedItem `json:"allowed_items"`
}

func upsertSlot(tx *sql.Tx, itemID string, slot Slot) error {
	query := `INSERT INTO slots (
			slot_id,
			item_id,
			name
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (slot_id) DO UPDATE SET
			name = $3
		;`
	_, err := tx.Exec(query, slot.ID, itemID, slot.Name)
	if err != nil {
		return err
	}

	err = upsertManyAllowedItem(tx, slot.ID, slot.AllowedItems)
	if err != nil {
		return err
	}

	return nil
}

func upsertManySlot(tx *sql.Tx, itemID string, slots []Slot) error {
	for i := 0; i < len(slots); i++ {
		slot := slots[i]
		err := upsertSlot(tx, itemID, slot)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetSlotsByItemID(db *sql.DB, itemID string) ([]Slot, error) {
	rows, err := db.Query(`select slot_id, name from slots where item_id = $1`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]Slot, 0)
	for rows.Next() {
		slot := Slot{}
		err := rows.Scan(&slot.ID, &slot.Name)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}

	return slots, nil
}
