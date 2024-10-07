package evaluator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSlot_GetParentItem_NoParent(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item := ConstructItem("item1", "Item1", rootWeapon)
	parent := item.GetParentSlot()

	assert.Nil(t, parent)
}

func TestSlot_GetParentItem_WithParent(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item := ConstructItem("item1", "Item1", rootWeapon)
	slot := ConstructSlot("slot1", "Slot1", rootWeapon)

	slot.SetParentItem(item)

	parentItem := slot.GetParentItem()
	assert.NotNil(t, parentItem)
	assert.Equal(t, parentItem, item)
}

func TestSlot_GetAncestors(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item1 := ConstructItem("item1", "Item1", rootWeapon)
	item2 := ConstructItem("item2", "Item2", rootWeapon)
	item3 := ConstructItem("item3", "Item3", rootWeapon)
	slot1 := ConstructSlot("slot1", "Slot1", rootWeapon)
	slot2 := ConstructSlot("slot2", "Slot2", rootWeapon)
	slot3 := ConstructSlot("slot3", "Slot3", rootWeapon)

	item1.AddChildSlot(slot1)
	slot1.AddChildItem(item2)
	item2.AddChildSlot(slot2)
	slot2.AddChildItem(item3)
	item3.AddChildSlot(slot3)

	ancestors := slot3.GetAncestorIds()

	assert.Equal(t, ancestors[0], item3.ID)
	assert.Equal(t, ancestors[1], slot2.ID)
	assert.Equal(t, ancestors[2], item2.ID)
	assert.Equal(t, ancestors[3], slot1.ID)
	assert.Equal(t, ancestors[4], item1.ID)
	assert.Equal(t, len(ancestors), 5)
}

func TestSlot_GetAncestorItems(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item1 := ConstructItem("item1", "Item1", rootWeapon)
	item2 := ConstructItem("item2", "Item2", rootWeapon)
	item3 := ConstructItem("item3", "Item3", rootWeapon)
	slot1 := ConstructSlot("slot1", "Slot1", rootWeapon)
	slot2 := ConstructSlot("slot2", "Slot2", rootWeapon)
	slot3 := ConstructSlot("slot3", "Slot3", rootWeapon)

	item1.AddChildSlot(slot1)
	slot1.AddChildItem(item2)
	item2.AddChildSlot(slot2)
	slot2.AddChildItem(item3)
	item3.AddChildSlot(slot3)

	ancestors := slot3.GetAncestorItems()

	assert.Equal(t, ancestors[0], item3)
	assert.Equal(t, ancestors[1], item2)
	assert.Equal(t, ancestors[2], item1)
	assert.Equal(t, len(ancestors), 3)
}
