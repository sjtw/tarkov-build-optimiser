package models_test

import (
	"database/sql"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
	"testing"

	"github.com/rs/zerolog/log"
)

func createMockWeapon(tx *sql.Tx) models.Weapon {
	weapon := models.Weapon{
		ID:                 "mock_weapon",
		Name:               "M4A1",
		ErgonomicsModifier: 12,
		RecoilModifier:     50,
		Slots: []models.Slot{
			{ID: "mock_slot_1", Name: "Muzzle"},
			{ID: "mock_slot_2", Name: "Stock"},
		},
	}

	err := models.UpsertWeapon(tx, weapon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create mock weapon")
	}

	return weapon
}

func reset(db *sql.DB) {
	err := models.Purge(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to purge all")
	}
}

// TODO: Model tests should use a separate DB instance
func TestCreateAndGetWeapon(t *testing.T) {
	environment, err := env.Get()
	if err != nil {
		t.Fatal(err)
	}
	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	conn := dbClient.Conn

	reset(conn)
	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("failed to begin transaction")
	}
	createdWeapon := createMockWeapon(tx)
	err = tx.Commit()
	if err != nil {
		t.Fatal("failed to commit transaction")
	}

	weapon, err := models.GetWeaponById(conn, createdWeapon.ID)
	if err != nil {
		t.Errorf("TestGetWeapon failed: %v", err)
	}

	if weapon.ID != createdWeapon.ID {
		t.Errorf("TestGetWeapon failed: expected %s, got %s", createdWeapon.ID, weapon.ID)
	}
	if weapon.Name != createdWeapon.Name {
		t.Errorf("TestGetWeapon failed: expected %s, got %s", createdWeapon.Name, weapon.Name)
	}
	if weapon.ErgonomicsModifier != createdWeapon.ErgonomicsModifier {
		t.Errorf("TestGetWeapon failed: expected %d, got %d", createdWeapon.ErgonomicsModifier, weapon.ErgonomicsModifier)
	}
	if weapon.RecoilModifier != createdWeapon.RecoilModifier {
		t.Errorf("TestGetWeapon failed: expected %d, got %d", createdWeapon.RecoilModifier, weapon.RecoilModifier)
	}
	if weapon.Slots == nil {
		t.Errorf("TestGetWeapon failed: expected slots to be populated")
	}
	if len(weapon.Slots) != 2 {
		t.Errorf("TestGetWeapon failed: expected 2 slots, got %d", len(weapon.Slots))
	}
	if !helpers.ContainsSlot(weapon.Slots, createdWeapon.Slots[0]) {
		t.Errorf("TestGetWeapon failed: expected %s, got %s", createdWeapon.Slots[0], weapon.Slots[0])
	}
	if !helpers.ContainsSlot(weapon.Slots, createdWeapon.Slots[1]) {
		t.Errorf("TestGetWeapon failed: expected %s, got %s", createdWeapon.Slots[1], weapon.Slots[1])
	}
}
