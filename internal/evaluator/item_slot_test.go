package evaluator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSlot_GetParentItem_NoParent(t *testing.T) {
	item := ConstructItem("item1", "Item1")
	parent := item.GetParentSlot()

	assert.Nil(t, parent)
}

func TestSlot_GetParentItem_WithParent(t *testing.T) {
	item := ConstructItem("item1", "Item1")
	slot := ConstructSlot("slot1", "Slot1")

	slot.SetParentItem(item)

	parentItem := slot.GetParentItem()
	assert.NotNil(t, parentItem)
	assert.Equal(t, parentItem, item)
}

func TestSlot_GetAncestors(t *testing.T) {
	item1 := ConstructItem("item1", "Item1")
	item2 := ConstructItem("item2", "Item2")
	item3 := ConstructItem("item3", "Item3")
	slot1 := ConstructSlot("slot1", "Slot1")
	slot2 := ConstructSlot("slot2", "Slot2")
	slot3 := ConstructSlot("slot3", "Slot3")

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
	item1 := ConstructItem("item1", "Item1")
	item2 := ConstructItem("item2", "Item2")
	item3 := ConstructItem("item3", "Item3")
	slot1 := ConstructSlot("slot1", "Slot1")
	slot2 := ConstructSlot("slot2", "Slot2")
	slot3 := ConstructSlot("slot3", "Slot3")

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
