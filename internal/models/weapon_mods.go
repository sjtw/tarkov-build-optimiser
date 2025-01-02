package models

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type WeaponMod struct {
	ID                 string   `json:"item_id"`
	Name               string   `json:"name"`
	ErgonomicsModifier int      `json:"ergonomics_modifier"`
	RecoilModifier     int      `json:"recoil_modifier"`
	CategoryName       string   `json:"category_name"`
	CategoryID         string   `json:"category_id"`
	Slots              []Slot   `json:"slots"`
	ConflictingItems   []string `json:"conflicting_items"`
}

func UpsertMod(tx *sql.Tx, mod WeaponMod) error {
	query := `
        insert into weapon_mods (item_id,
                                 name,
                                 ergonomics_modifier,
                                 recoil_modifier,
                                 category_name,
                                 category_id)
        values ($1, $2, $3, $4, $5, $6)
        on conflict (item_id) do update set name                = $2,
                                            ergonomics_modifier = $3,
                                            recoil_modifier     = $4,
                                            category_name       = $5,
                                            category_id         = $6;`

	_, err := tx.Exec(query, mod.ID, mod.Name, mod.ErgonomicsModifier, mod.RecoilModifier, mod.CategoryName, mod.CategoryID)
	if err != nil {
		return err
	}

	err = upsertManySlot(tx, mod.ID, mod.Slots)
	if err != nil {
		return err
	}

	err = upsertManyConflictingItems(tx, mod.ID, mod.ConflictingItems)
	if err != nil {
		return err
	}

	return nil
}

func UpsertManyMod(tx *sql.Tx, mods []WeaponMod) error {
	for i := 0; i < len(mods); i++ {
		err := UpsertMod(tx, mods[i])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to upsert mod: %v", mods[i])
			return err
		}
		log.Debug().Msgf("Upserted mod: ID: %s, Name: %s", mods[i].ID, mods[i].Name)
	}
	return nil
}

func GetAllWeaponMods(db *sql.DB) ([]*WeaponMod, error) {
	rows, err := db.Query(`
		select wm.item_id,
			   wm.name,
			   wm.ergonomics_modifier,
			   wm.recoil_modifier,
			   COALESCE(array_agg(ci.conflicting_item_id) FILTER (WHERE ci.conflicting_item_id IS NOT NULL),
						'{}') AS conflicting_items
		from weapon_mods wm
				 left join conflicting_items ci on wm.item_id = ci.item_id
		group by wm.item_id, wm.name, wm.ergonomics_modifier, wm.recoil_modifier;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weaponMods := make([]*WeaponMod, 0)
	for rows.Next() {
		weaponMod := &WeaponMod{}
		err := rows.Scan(&weaponMod.ID, &weaponMod.Name, &weaponMod.ErgonomicsModifier, &weaponMod.RecoilModifier, pq.Array(&weaponMod.ConflictingItems))
		if err != nil {
			return nil, err
		}

		weaponMods = append(weaponMods, weaponMod)
	}

	return weaponMods, nil
}

func GetWeaponModById(db *sql.DB, id string) (*WeaponMod, error) {
	rows, err := db.Query(`
        select wm.item_id,
               wm.name,
               wm.ergonomics_modifier,
               wm.recoil_modifier,
               wm.category_id,
               wm.category_name,
               COALESCE(array_agg(ci.conflicting_item_id) FILTER (WHERE ci.conflicting_item_id IS NOT NULL),
                        '{}') AS conflicting_items
        from weapon_mods wm
                 left join conflicting_items ci on wm.item_id = ci.item_id
        where wm.item_id = $1
        group by wm.item_id, wm.name, wm.ergonomics_modifier, wm.recoil_modifier;`, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	weaponMod := &WeaponMod{}
	for rows.Next() {
		err := rows.Scan(&weaponMod.ID, &weaponMod.Name, &weaponMod.ErgonomicsModifier, &weaponMod.RecoilModifier, &weaponMod.CategoryID, &weaponMod.CategoryName, pq.Array(&weaponMod.ConflictingItems))
		if err != nil {
			return nil, err
		}
	}

	return weaponMod, nil
}
