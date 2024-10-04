package evaluator

import (
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/models"
	"testing"
)

func createMockConstraints() models.EvaluationConstraints {
	constraints := models.EvaluationConstraints{
		TraderLevels: []models.TraderLevel{},
	}

	for i := 0; i < len(models.TraderNames); i++ {
		trader := models.TraderLevel{Name: models.TraderNames[i], Level: 5}
		constraints.TraderLevels = append(constraints.TraderLevels, trader)
	}

	return constraints
}

type MockTraderOfferGetter struct {
}

func CreateMockTraderOfferGetter() TraderOfferGetter {
	return &MockTraderOfferGetter{}
}

func (to *MockTraderOfferGetter) Get(_ string) ([]models.TraderOffer, error) {
	offers := []models.TraderOffer{
		{ID: "test-item", Name: "1", Trader: "Prapor", MinTraderLevel: 1, PriceRub: 1},
		{ID: "item1", Name: "1", Trader: "Prapor", MinTraderLevel: 1, PriceRub: 1},
		{ID: "item2", Name: "1", Trader: "Prapor", MinTraderLevel: 1, PriceRub: 1},
		{ID: "item3", Name: "1", Trader: "Prapor", MinTraderLevel: 1, PriceRub: 1},
		{ID: "child-item1", Name: "1", Trader: "Prapor", MinTraderLevel: 1, PriceRub: 1},
	}
	return offers, nil
}

type MockBuildSaver struct{}

func (saver *MockBuildSaver) Save(_ string, _ string, _ string, _ int, _ *models.ItemEvaluationResult, _ string, _ models.EvaluationConstraints) error {
	return nil
}

func CreateMockBuildSaver() BuildSaver {
	return &MockBuildSaver{}
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

	constraints := createMockConstraints()
	offerGetter := CreateMockTraderOfferGetter()
	buildSaver := CreateMockBuildSaver()

	evaluator := CreateEvaluator(offerGetter, buildSaver)
	build, err := evaluator.evaluate(weapon, "recoil", constraints)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, build.RecoilSum, -18)

	assert.Len(t, build.Slots, 2)
	assert.NotNil(t, build.Slots[0])

	assert.Equal(t, build.Slots[0].Item.RecoilSum, -5)

	assert.NotNil(t, build.Slots[0].Item)
	assert.Equal(t, build.Slots[0].Item.ID, item1.ID)
	assert.Len(t, build.Slots[0].Item.Slots, 1)
	assert.Equal(t, build.Slots[0].Item.Slots[0].ID, item1ChildSlot.ID)
	assert.Equal(t, build.Slots[0].Item.Slots[0].Item.ID, item1ChildSlotItem.ID)
	assert.Equal(t, build.Slots[0].Item.Slots[0].Item.RecoilSum, item1ChildSlotItem.RecoilModifier)

	assert.NotNil(t, build.Slots[1])
	assert.Equal(t, build.Slots[1].Item.RecoilSum, -3)
	assert.Equal(t, build.Slots[1].Item.ID, item3.ID)
	assert.Len(t, build.Slots[1].Item.Slots, 0)
}

func TestFindBestErgoTree(t *testing.T) {
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

	constraints := createMockConstraints()
	offerGetter := CreateMockTraderOfferGetter()
	buildSaver := CreateMockBuildSaver()
	evaluator := CreateEvaluator(offerGetter, buildSaver)

	build, err := evaluator.evaluate(weapon, "recoil", constraints)

	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 18, build.ErgonomicsSum)

	assert.Len(t, build.Slots, 2)
	assert.NotNil(t, build.Slots[0])

	assert.Equal(t, build.Slots[0].Item.ErgonomicsSum, 5)

	assert.NotNil(t, build.Slots[0].Item)
	assert.Equal(t, build.Slots[0].Item.ID, item1.ID)
	assert.Len(t, build.Slots[0].Item.Slots, 1)
	assert.Equal(t, build.Slots[0].Item.Slots[0].ID, item1ChildSlot.ID)
	assert.Equal(t, build.Slots[0].Item.Slots[0].Item.ID, item1ChildSlotItem.ID)
	assert.Equal(t, build.Slots[0].Item.Slots[0].Item.ErgonomicsSum, item1ChildSlotItem.ErgonomicsModifier)

	assert.NotNil(t, build.Slots[1])
	assert.Equal(t, build.Slots[1].Item.ErgonomicsSum, 3)
	assert.Equal(t, build.Slots[1].Item.ID, item3.ID)
	assert.Len(t, build.Slots[1].Item.Slots, 0)
}
