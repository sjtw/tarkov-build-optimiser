package evaluator

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/db"
	"testing"
)

func TestCalculateOptimumRecoil(t *testing.T) {
	dbClient, err := db.CreateBuildOptimiserDBClient()
	id := "5a0ec13bfcdbcb00165aa685"
	weapon, err := CreateWeaponPossibilityTree(dbClient.Conn, id)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", id)
	}
	err = GenerateOptimumWeaponBuilds(dbClient.Conn, weapon.ID)
	assert.Nil(t, err)
}

func TestCreateWeaponPossibilityTree(t *testing.T) {
	// AKMN
	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
		return
	}

	id := "5a0ec13bfcdbcb00165aa685"
	weapon, err := CreateWeaponPossibilityTree(dbClient.Conn, id)
	assert.NoError(t, err)

	assert.IsType(t, &Item{}, weapon)
}

func TestFindBestRecoilTree(t *testing.T) {
	weapon := ConstructItem("test-item", "Test Item")
	weapon.RecoilModifier = -10
	weapon.ErgonomicsModifier = 10

	item1 := ConstructItem("item1", "Item1")
	item1.RecoilModifier = -1
	item1.ErgonomicsModifier = 1

	item2 := ConstructItem("item2", "Item2")
	item2.RecoilModifier = -2
	item2.ErgonomicsModifier = 2

	item3 := ConstructItem("item3", "Item3")
	item3.RecoilModifier = -3
	item3.ErgonomicsModifier = 3

	childSlot := ConstructSlot("child-slot1", "Child Slot1")
	childItem := ConstructItem("child-item1", "Child Item1")
	childItem.RecoilModifier = -4
	childItem.ErgonomicsModifier = 4
	childSlot.AddChildItem(childItem)
	item1.AddChildSlot(childSlot)

	slot1 := ConstructSlot("slot1", "Slot1")
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := ConstructSlot("slot2", "Slot2")
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	sum, build := findBestRecoilTree(weapon)
	fmt.Println(sum)
	fmt.Println(build)
}
