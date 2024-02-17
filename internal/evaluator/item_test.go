package evaluator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestItem_GetParentSlot_NoParent(t *testing.T) {
	item := ConstructItem("item1", "Item1")
	parentSlot := item.GetParentSlot()

	assert.Nil(t, parentSlot)
}

func TestItem_GetParentSlot_WithParent(t *testing.T) {
	item := ConstructItem("item1", "Item1")
	slot := ConstructSlot("slot1", "Slot1")
	item.SetParentSlot(slot)

	parentSlot := item.GetParentSlot()
	assert.NotNil(t, parentSlot)
	assert.Equal(t, parentSlot, slot)
}
