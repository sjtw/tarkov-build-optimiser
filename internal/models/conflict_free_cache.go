package models

import (
	"context"
	"database/sql"
)

type ConflictFreeCache struct {
	ItemID           string `json:"item_id"`
	FocusedStat      string `json:"focused_stat"`
	JaegerLevel      int    `json:"jaeger_level"`
	PraporLevel      int    `json:"prapor_level"`
	PeacekeeperLevel int    `json:"peacekeeper_level"`
	MechanicLevel    int    `json:"mechanic_level"`
	SkierLevel       int    `json:"skier_level"`
	RecoilSum        int    `json:"recoil_sum"`
	ErgonomicsSum    int    `json:"ergonomics_sum"`
}

// UpsertConflictFreeCache stores or updates a conflict-free cache entry
func UpsertConflictFreeCache(db *sql.DB, entry *ConflictFreeCache) error {
	query := `
        insert into conflict_free_cache (
            item_id, focused_stat, jaeger_level, prapor_level, peacekeeper_level, 
            mechanic_level, skier_level, recoil_sum, ergonomics_sum
        ) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        on conflict (item_id, focused_stat, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level)
        do update set
            recoil_sum = excluded.recoil_sum,
            ergonomics_sum = excluded.ergonomics_sum,
            updated_at = now();
    `

	_, err := db.Exec(query,
		entry.ItemID, entry.FocusedStat, entry.JaegerLevel, entry.PraporLevel, entry.PeacekeeperLevel,
		entry.MechanicLevel, entry.SkierLevel, entry.RecoilSum, entry.ErgonomicsSum,
	)
	return err
}

// GetConflictFreeCache retrieves a conflict-free cache entry
func GetConflictFreeCache(ctx context.Context, db *sql.DB, itemID string, focusedStat string, levels []TraderLevel) (*ConflictFreeCache, error) {
	query := `
        select item_id, focused_stat, jaeger_level, prapor_level, peacekeeper_level, 
               mechanic_level, skier_level, recoil_sum, ergonomics_sum
        from conflict_free_cache
        where item_id = $1 and focused_stat = $2
          and jaeger_level = $3 and prapor_level = $4 and peacekeeper_level = $5 
          and mechanic_level = $6 and skier_level = $7
        limit 1;
    `

	row := db.QueryRowContext(ctx, query, itemID, focusedStat,
		levelsByName(levels, "Jaeger"), levelsByName(levels, "Prapor"), levelsByName(levels, "Peacekeeper"),
		levelsByName(levels, "Mechanic"), levelsByName(levels, "Skier"))

	var entry ConflictFreeCache
	err := row.Scan(&entry.ItemID, &entry.FocusedStat, &entry.JaegerLevel, &entry.PraporLevel, &entry.PeacekeeperLevel,
		&entry.MechanicLevel, &entry.SkierLevel, &entry.RecoilSum, &entry.ErgonomicsSum)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// PurgeConflictFreeCache removes all entries from the conflict-free cache
func PurgeConflictFreeCache(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM conflict_free_cache;")
	return err
}
