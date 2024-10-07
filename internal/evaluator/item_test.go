package evaluator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestItem_GetParentSlot_NoParent(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item := ConstructItem("item1", "Item1", rootWeapon)
	parentSlot := item.GetParentSlot()

	assert.Nil(t, parentSlot)
}

func TestItem_GetParentSlot_WithParent(t *testing.T) {
	rootWeapon := &WeaponTree{}
	item := ConstructItem("item1", "Item1", rootWeapon)
	slot := ConstructSlot("slot1", "Slot1", rootWeapon)
	item.SetParentSlot(slot)

	parentSlot := item.GetParentSlot()
	assert.NotNil(t, parentSlot)
	assert.Equal(t, parentSlot, slot)
}
