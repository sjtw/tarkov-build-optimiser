package evaluator

import (
	"github.com/stretchr/testify/assert"
	"sync"
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

type SaveArgs struct {
	Build       *models.ItemEvaluationResult
	Constraints models.EvaluationConstraints
	IsSubtree   bool
}

// MockBuildSaver allows inspection of arguments in calls to the savers Save method
type MockBuildSaver struct {
	SaveCallCount int        // Tracks how many times Save was called
	SaveArgs      []SaveArgs // Stores the arguments passed to Save
	mu            sync.Mutex
}

func (saver *MockBuildSaver) Save(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints, isSubtree bool) error {
	saver.mu.Lock()
	defer saver.mu.Unlock()

	saver.SaveCallCount++
	saver.SaveArgs = append(saver.SaveArgs, SaveArgs{
		Build:       build,
		Constraints: constraints,
		IsSubtree:   isSubtree,
	})

	return nil
}

func CreateMockBuildSaver() *MockBuildSaver {
	return &MockBuildSaver{
		SaveCallCount: 0,
		SaveArgs:      []SaveArgs{},
		mu:            sync.Mutex{},
	}
}

type MockSubtreeGetter struct{}

func (sg *MockSubtreeGetter) Get(_ string, _ string, _ models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	return nil, nil
}

func CreateMockSubtreeGetter() SubtreeGetter {
	return &MockSubtreeGetter{}
}

func TestFindBestRecoilTree(t *testing.T) {
	rootWeaponTree := &WeaponTree{}
	weapon := ConstructItem("weapon1", "Weapon1", rootWeaponTree)
	weapon.RecoilModifier = -10
	weapon.ErgonomicsModifier = 10

	item1 := ConstructItem("item1", "Item1", rootWeaponTree)
	item1.RecoilModifier = -1
	item1.ErgonomicsModifier = 1

	item2 := ConstructItem("item2", "Item2", rootWeaponTree)
	item2.RecoilModifier = -2
	item2.ErgonomicsModifier = 2

	item3 := ConstructItem("item3", "Item3", rootWeaponTree)
	item3.RecoilModifier = -3
	item3.ErgonomicsModifier = 3

	item1ChildSlot := ConstructSlot("child-slot1", "Child Slot1", rootWeaponTree)
	item1ChildSlotItem := ConstructItem("child-item1", "Child Item1", rootWeaponTree)
	item1ChildSlotItem.RecoilModifier = -4
	item1ChildSlotItem.ErgonomicsModifier = 4
	item1ChildSlot.AddChildItem(item1ChildSlotItem)
	item1.AddChildSlot(item1ChildSlot)

	slot1 := ConstructSlot("slot1", "Slot1", rootWeaponTree)
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := ConstructSlot("slot2", "Slot2", rootWeaponTree)
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	constraints := createMockConstraints()
	offerGetter := CreateMockTraderOfferGetter()
	buildSaver := CreateMockBuildSaver()
	subtreeGetter := CreateMockSubtreeGetter()

	evaluator := CreateEvaluator(offerGetter, buildSaver, subtreeGetter)
	build, err := evaluator.evaluate(weapon, "recoil", constraints)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, build.RecoilSum, -18)

	assert.Len(t, build.Slots, 2)
	assert.NotNil(t, build.Slots[0])

	assert.Equal(t, build.Slots[0].Item.RecoilSum, -5)

	assert.NotNil(t, build.Slots[0].Item)
	assert.Equal(t, item1.ID, build.Slots[0].Item.ID)
	assert.Len(t, build.Slots[0].Item.Slots, 1)
	assert.Equal(t, item1ChildSlot.ID, build.Slots[0].Item.Slots[0].ID)
	assert.Equal(t, item1ChildSlotItem.ID, build.Slots[0].Item.Slots[0].Item.ID)
	assert.Equal(t, item1ChildSlotItem.RecoilModifier, build.Slots[0].Item.Slots[0].Item.RecoilSum)

	assert.NotNil(t, build.Slots[1])
	assert.Equal(t, -3, build.Slots[1].Item.RecoilSum)
	assert.Equal(t, item3.ID, build.Slots[1].Item.ID)
	assert.Len(t, build.Slots[1].Item.Slots, 0)

	// after each possible subtree is evaluated these are saved by buildSaver
	// assert they are ordered correctly & contain the expected states
	assert.Equal(t, 5, len(buildSaver.SaveArgs))

	assert.Equal(t, item1ChildSlotItem.ID, buildSaver.SaveArgs[0].Build.ID)
	assert.Equal(t, -4, buildSaver.SaveArgs[0].Build.RecoilSum)
	assert.Equal(t, "recoil", buildSaver.SaveArgs[0].Build.EvaluationType)
	assert.Equal(t, item1ChildSlotItem.Name, buildSaver.SaveArgs[0].Build.Name)
	assert.Equal(t, constraints, buildSaver.SaveArgs[0].Constraints)
	assert.Equal(t, constraints, buildSaver.SaveArgs[0].Constraints)

	assert.Equal(t, item1.ID, buildSaver.SaveArgs[1].Build.ID)
	assert.Equal(t, item1ChildSlotItem.RecoilModifier+item1.RecoilModifier, buildSaver.SaveArgs[1].Build.RecoilSum)
	assert.Equal(t, "recoil", buildSaver.SaveArgs[1].Build.EvaluationType)
	assert.Equal(t, item1.Name, buildSaver.SaveArgs[1].Build.Name)
	assert.Equal(t, constraints, buildSaver.SaveArgs[1].Constraints)
	assert.Equal(t, constraints, buildSaver.SaveArgs[1].Constraints)

	assert.Equal(t, item2.ID, buildSaver.SaveArgs[2].Build.ID)
	assert.Equal(t, item2.RecoilModifier, buildSaver.SaveArgs[2].Build.RecoilSum)
	assert.Equal(t, "recoil", buildSaver.SaveArgs[2].Build.EvaluationType)
	assert.Equal(t, item2.Name, buildSaver.SaveArgs[2].Build.Name)
	assert.Equal(t, constraints, buildSaver.SaveArgs[2].Constraints)
	assert.Equal(t, constraints, buildSaver.SaveArgs[2].Constraints)

}

func TestFindBestErgoTree(t *testing.T) {
	rootWeaponTree := &WeaponTree{}
	weapon := ConstructItem("test-item", "Test Item", rootWeaponTree)
	weapon.RecoilModifier = -10
	weapon.ErgonomicsModifier = 10

	item1 := ConstructItem("item1", "Item1", rootWeaponTree)
	item1.RecoilModifier = -1
	item1.ErgonomicsModifier = 1

	item2 := ConstructItem("item2", "Item2", rootWeaponTree)
	item2.RecoilModifier = -2
	item2.ErgonomicsModifier = 2

	item3 := ConstructItem("item3", "Item3", rootWeaponTree)
	item3.RecoilModifier = -3
	item3.ErgonomicsModifier = 3

	item1ChildSlot := ConstructSlot("child-slot1", "Child Slot1", rootWeaponTree)
	item1ChildSlotItem := ConstructItem("child-item1", "Child Item1", rootWeaponTree)
	item1ChildSlotItem.RecoilModifier = -4
	item1ChildSlotItem.ErgonomicsModifier = 4
	item1ChildSlot.AddChildItem(item1ChildSlotItem)
	item1.AddChildSlot(item1ChildSlot)

	slot1 := ConstructSlot("slot1", "Slot1", rootWeaponTree)
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := ConstructSlot("slot2", "Slot2", rootWeaponTree)
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	constraints := createMockConstraints()
	offerGetter := CreateMockTraderOfferGetter()
	buildSaver := CreateMockBuildSaver()
	subtreeGetter := CreateMockSubtreeGetter()
	evaluator := CreateEvaluator(offerGetter, buildSaver, subtreeGetter)

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
