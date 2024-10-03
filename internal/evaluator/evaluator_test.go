package evaluator

import (
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/db"
	"testing"
)

func TestCalculateOptimumRecoil(t *testing.T) {
	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
		t.Fatal()
	}
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

	item1ChildSlot := ConstructSlot("child-slot1", "Child Slot1")
	item1ChildSlotItem := ConstructItem("child-item1", "Child Item1")
	item1ChildSlotItem.RecoilModifier = -4
	item1ChildSlotItem.ErgonomicsModifier = 4
	item1ChildSlot.AddChildItem(item1ChildSlotItem)
	item1.AddChildSlot(item1ChildSlot)

	slot1 := ConstructSlot("slot1", "Slot1")
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := ConstructSlot("slot2", "Slot2")
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	sum, build := findBestRecoilTree(weapon)

	assert.Equal(t, sum, -18)

	assert.Len(t, build.Slots, 2)
	assert.NotNil(t, build.Slots[0])

	assert.Equal(t, build.Slots[0].BestRecoilModifier, -5)

	assert.NotNil(t, build.Slots[0].BestRecoilItem)
	assert.Equal(t, build.Slots[0].BestRecoilItem.ID, item1.ID)
	assert.Len(t, build.Slots[0].BestRecoilItem.Slots, 1)
	assert.Equal(t, build.Slots[0].BestRecoilItem.Slots[0].ID, item1ChildSlot.ID)
	assert.Equal(t, build.Slots[0].BestRecoilItem.Slots[0].BestRecoilItem.ID, item1ChildSlotItem.ID)
	assert.Equal(t, build.Slots[0].BestRecoilItem.Slots[0].BestRecoilModifier, item1ChildSlotItem.RecoilModifier)

	assert.NotNil(t, build.Slots[1])
	assert.Equal(t, build.Slots[1].BestRecoilModifier, -3)
	assert.Equal(t, build.Slots[1].BestRecoilItem.ID, item3.ID)
	assert.Len(t, build.Slots[1].BestRecoilItem.Slots, 0)
}
