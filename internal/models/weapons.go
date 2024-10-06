package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
)

type Weapon struct {
	ID                 string `json:"item_id"`
	Name               string `json:"name"`
	ErgonomicsModifier int    `json:"ergonomics_modifier"`
	RecoilModifier     int    `json:"recoil_modifier"`
	Slots              []Slot `json:"slots"`
}

func UpsertWeapon(tx *sql.Tx, weapon Weapon) error {
	query := `INSERT INTO weapons (
			item_id,
			name,
			recoil_modifier,
			ergonomics_modifier
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (item_id) DO UPDATE SET
			name = $2,
			recoil_modifier = $3,
			ergonomics_modifier = $4;`
	_, err := tx.Exec(query, weapon.ID, weapon.Name, weapon.RecoilModifier, weapon.ErgonomicsModifier)
	if err != nil {
		return err
	}

	err = upsertManySlot(tx, weapon.ID, weapon.Slots)
	if err != nil {
		return err
	}

	return nil
}

func UpsertManyWeapon(tx *sql.Tx, weapons []Weapon) error {
	for i := 0; i < len(weapons); i++ {
		err := UpsertWeapon(tx, weapons[i])
		if err != nil {
			return err
		}
		log.Debug().Msgf("Upserted weapon: ID: %s, Name: %s", weapons[i].ID, weapons[i].Name)
	}

	return nil
}

func GetWeaponById(db *sql.DB, id string) (*Weapon, error) {
	query := `
		select w.name,
					w.item_id             as id,
					w.recoil_modifier     as recoil_modifier,
					w.ergonomics_modifier as ergonomics_modifier,
					jsonb_agg(jsonb_build_object('slot_id', ws.slot_id, 'name', ws.name, 'allowed_items', (
							select jsonb_agg(jsonb_build_object('item_id', sai.item_id, 'name', sai.name))
							from slot_allowed_items sai
							where sai.slot_id = ws.slot_id
					)))                   as slots
		from weapons w
						join slots ws on w.item_id = ws.item_id
		where w.item_id = $1
		group by w.name, w.item_id, w.recoil_modifier, w.ergonomics_modifier;`

	rows, err := db.Query(query, id)

	defer rows.Close()

	if err != nil {
		return nil, err
	}

	weapons := make([]*Weapon, 0)
	for rows.Next() {
		weapon := &Weapon{}
		var slotsStr string
		err := rows.Scan(&weapon.Name, &weapon.ID, &weapon.RecoilModifier, &weapon.ErgonomicsModifier, &slotsStr)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(slotsStr), &weapon.Slots); err != nil {
			return nil, err
		}

		weapons = append(weapons, weapon)
	}

	if len(weapons) == 0 {
		msg := fmt.Sprintf("No results for %s", id)
		return nil, errors.New(msg)
	}

	if len(weapons) > 1 {
		msg := fmt.Sprintf("Multiple results for %s", id)
		return nil, errors.New(msg)
	}

	return weapons[0], nil
}

func GetAllWeaponIds(db *sql.DB) ([]string, error) {
	query := `select item_id from weapons;`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func GetWeapons(db *sql.DB) ([]Weapon, error) {
	query := `
		select w.name,
					w.item_id             as id,
					w.recoil_modifier     as recoil_modifier,
					w.ergonomics_modifier as ergonomics_modifier,
					jsonb_agg(jsonb_build_object('slot_id', ws.slot_id, 'name', ws.name, 'allowed_items', (
							select jsonb_agg(jsonb_build_object('item_id', sai.item_id, 'name', sai.name))
							from slot_allowed_items sai
							where sai.slot_id = ws.slot_id
					)))                   as slots
		from weapons w
						join slots ws on w.item_id = ws.item_id
		group by w.name, w.item_id, w.recoil_modifier, w.ergonomics_modifier;
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weapons []Weapon
	for rows.Next() {
		weapon := Weapon{}
		var slotsStr string
		err := rows.Scan(&weapon.Name, &weapon.ID, &weapon.RecoilModifier, &weapon.ErgonomicsModifier, &slotsStr)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(slotsStr), &weapon.Slots); err != nil {
			return nil, err
		}

		weapons = append(weapons, weapon)
	}

	return weapons, nil
}

func Purge(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM weapons;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM slots;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM slot_allowed_items;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM weapon_mods;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM trader_offers;`)
	if err != nil {
		return err
	}

	return nil
}
