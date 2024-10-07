package models

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
)

type WeaponMod struct {
	ID                 string `json:"item_id"`
	Name               string `json:"name"`
	ErgonomicsModifier int    `json:"ergonomics_modifier"`
	RecoilModifier     int    `json:"recoil_modifier"`
	Slots              []Slot `json:"slots"`
}

func UpsertMod(tx *sql.Tx, mod WeaponMod) error {
	query := `
	insert into weapon_mods (item_id,
                         name,
                         ergonomics_modifier,
                         recoil_modifier)
	values ($1, $2, $3, $4) on conflict (item_id) do update set
		name = $2,
		ergonomics_modifier = $3,
		recoil_modifier = $4
	;`
	_, err := tx.Exec(query, mod.ID, mod.Name, mod.ErgonomicsModifier, mod.RecoilModifier)
	if err != nil {
		return err
	}

	err = upsertManySlot(tx, mod.ID, mod.Slots)
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

func GetWeaponModById(db *sql.DB, id string) (*WeaponMod, error) {
	rows, err := db.Query(`select item_id, name, ergonomics_modifier, recoil_modifier from weapon_mods where item_id = $1;`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weaponMods := make([]*WeaponMod, 0)
	for rows.Next() {
		weaponMod := &WeaponMod{}
		err := rows.Scan(&weaponMod.ID, &weaponMod.Name, &weaponMod.ErgonomicsModifier, &weaponMod.RecoilModifier)
		if err != nil {
			return nil, err
		}
		weaponMods = append(weaponMods, weaponMod)
	}

	if len(weaponMods) == 0 {
		msg := fmt.Sprintf("No results for %s", id)
		return nil, errors.New(msg)
	}

	if len(weaponMods) > 1 {
		msg := fmt.Sprintf("Expected 1 result for weapon mod %s, got %d", id, len(weaponMods))
		return nil, errors.New(msg)
	}

	return weaponMods[0], nil
}

func GetAllWeaponMods(db *sql.DB) ([]*WeaponMod, error) {
	rows, err := db.Query(`select item_id, name, ergonomics_modifier, recoil_modifier from weapon_mods;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weaponMods := make([]*WeaponMod, 0)
	for rows.Next() {
		weaponMod := &WeaponMod{}
		err := rows.Scan(&weaponMod.ID, &weaponMod.Name, &weaponMod.ErgonomicsModifier, &weaponMod.RecoilModifier)
		if err != nil {
			return nil, err
		}
		weaponMods = append(weaponMods, weaponMod)
	}

	return weaponMods, nil
}
