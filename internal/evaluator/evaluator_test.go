package evaluator

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/weapon_tree"
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

type MockEvaluationDataProvider struct {
	GetSubtreeFunc     func(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error)
	GetTraderOfferFunc func(itemID string) ([]models.TraderOffer, error)
	SaveBuildFunc      func(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints)
}

func (m *MockEvaluationDataProvider) GetSubtree(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error) {
	if m.GetSubtreeFunc != nil {
		return m.GetSubtreeFunc(itemId, buildType, constraints)
	}
	return nil, nil
}

func (m *MockEvaluationDataProvider) GetTraderOffer(itemID string) ([]models.TraderOffer, error) {
	if m.GetTraderOfferFunc != nil {
		return m.GetTraderOfferFunc(itemID)
	}
	return nil, nil
}

func (m *MockEvaluationDataProvider) SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) {
	if m.SaveBuildFunc != nil {
		m.SaveBuildFunc(build, constraints)
	}
}

func getTraders(_ string) ([]models.TraderOffer, error) {
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

type MockBuildSaver struct {
	SaveCallCount int        // Tracks how many times Save was called
	SaveArgs      []SaveArgs // Stores the arguments passed to Save
	mu            sync.Mutex
}

func (saver *MockBuildSaver) SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) {
	saver.mu.Lock()
	defer saver.mu.Unlock()

	saver.SaveCallCount++
	saver.SaveArgs = append(saver.SaveArgs, SaveArgs{
		Build:       build,
		Constraints: constraints,
	})
}

func TestFindBestRecoilTree(t *testing.T) {
	rootWeaponTree := &weapon_tree.WeaponTree{}
	weapon := weapon_tree.ConstructItem("weapon1", "Weapon1", rootWeaponTree)
	weapon.RecoilModifier = -10
	weapon.ErgonomicsModifier = 10

	item1 := weapon_tree.ConstructItem("item1", "Item1", rootWeaponTree)
	item1.RecoilModifier = -1
	item1.ErgonomicsModifier = 1

	item2 := weapon_tree.ConstructItem("item2", "Item2", rootWeaponTree)
	item2.RecoilModifier = -2
	item2.ErgonomicsModifier = 2

	item3 := weapon_tree.ConstructItem("item3", "Item3", rootWeaponTree)
	item3.RecoilModifier = -3
	item3.ErgonomicsModifier = 3

	item1ChildSlot := weapon_tree.ConstructSlot("child-slot1", "Child Slot1", rootWeaponTree)
	item1ChildSlotItem := weapon_tree.ConstructItem("child-item1", "Child Item1", rootWeaponTree)
	item1ChildSlotItem.RecoilModifier = -4
	item1ChildSlotItem.ErgonomicsModifier = 4
	item1ChildSlot.AddChildItem(item1ChildSlotItem)
	item1.AddChildSlot(item1ChildSlot)

	slot1 := weapon_tree.ConstructSlot("slot1", "Slot1", rootWeaponTree)
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := weapon_tree.ConstructSlot("slot2", "Slot2", rootWeaponTree)
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	constraints := createMockConstraints()

	buildSaver := &MockBuildSaver{}
	dataProvider := &MockEvaluationDataProvider{
		GetTraderOfferFunc: getTraders,
		SaveBuildFunc:      buildSaver.SaveBuild,
	}

	evaluator := CreateEvaluator(dataProvider)
	candidates := []string{"weapon1", "item1", "item2", "item3", "child-item1"}
	build, err := evaluator.evaluateWeapon(weapon, "recoil", constraints, candidates)
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
	//assert.Equal(t, 5, len(buildSaver.SaveArgs))

	//assert.Equal(t, item1ChildSlotItem.ID, buildSaver.SaveArgs[0].Build.ID)
	//assert.Equal(t, -4, buildSaver.SaveArgs[0].Build.RecoilSum)
	//assert.Equal(t, "recoil", buildSaver.SaveArgs[0].Build.EvaluationType)
	//assert.Equal(t, item1ChildSlotItem.Name, buildSaver.SaveArgs[0].Build.Name)
	//assert.Equal(t, constraints, buildSaver.SaveArgs[0].Constraints)
	//
	//assert.Equal(t, item1.ID, buildSaver.SaveArgs[1].Build.ID)
	//assert.Equal(t, item1ChildSlotItem.RecoilModifier+item1.RecoilModifier, buildSaver.SaveArgs[1].Build.RecoilSum)
	//assert.Equal(t, "recoil", buildSaver.SaveArgs[1].Build.EvaluationType)
	//assert.Equal(t, item1.Name, buildSaver.SaveArgs[1].Build.Name)
	//assert.Equal(t, constraints, buildSaver.SaveArgs[1].Constraints)
	//
	//assert.Equal(t, item2.ID, buildSaver.SaveArgs[2].Build.ID)
	//assert.Equal(t, item2.RecoilModifier, buildSaver.SaveArgs[2].Build.RecoilSum)
	//assert.Equal(t, "recoil", buildSaver.SaveArgs[2].Build.EvaluationType)
	//assert.Equal(t, item2.Name, buildSaver.SaveArgs[2].Build.Name)
	//assert.Equal(t, constraints, buildSaver.SaveArgs[2].Constraints)
}

func TestFindBestErgoTree(t *testing.T) {
	rootWeaponTree := &weapon_tree.WeaponTree{}
	weapon := weapon_tree.ConstructItem("weapon1", "Test Item", rootWeaponTree)
	weapon.RecoilModifier = -10
	weapon.ErgonomicsModifier = 10

	item1 := weapon_tree.ConstructItem("item1", "Item1", rootWeaponTree)
	item1.RecoilModifier = -1
	item1.ErgonomicsModifier = 1

	item2 := weapon_tree.ConstructItem("item2", "Item2", rootWeaponTree)
	item2.RecoilModifier = -2
	item2.ErgonomicsModifier = 2

	item3 := weapon_tree.ConstructItem("item3", "Item3", rootWeaponTree)
	item3.RecoilModifier = -3
	item3.ErgonomicsModifier = 3

	item1ChildSlot := weapon_tree.ConstructSlot("child-slot1", "Child Slot1", rootWeaponTree)
	item1ChildSlotItem := weapon_tree.ConstructItem("child-item1", "Child Item1", rootWeaponTree)
	item1ChildSlotItem.RecoilModifier = -4
	item1ChildSlotItem.ErgonomicsModifier = 4
	item1ChildSlot.AddChildItem(item1ChildSlotItem)
	item1.AddChildSlot(item1ChildSlot)

	slot1 := weapon_tree.ConstructSlot("slot1", "Slot1", rootWeaponTree)
	slot1.AddChildItem(item1)
	slot1.AddChildItem(item2)

	slot2 := weapon_tree.ConstructSlot("slot2", "Slot2", rootWeaponTree)
	slot2.AddChildItem(item3)

	weapon.AddChildSlot(slot1)
	weapon.AddChildSlot(slot2)

	constraints := createMockConstraints()
	buildSaver := &MockBuildSaver{}
	dataProvider := &MockEvaluationDataProvider{
		GetTraderOfferFunc: getTraders,
		SaveBuildFunc:      buildSaver.SaveBuild,
	}
	evaluator := CreateEvaluator(dataProvider)

	candidates := []string{"weapon1", "item1", "item2", "item3", "child-item1"}
	build, err := evaluator.evaluateWeapon(weapon, "recoil", constraints, candidates)

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

func TestFindBestBuild(t *testing.T) {
	// TODO - use construction functions and mock data provider to construct this
	rootItem := &weapon_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     0.0,
		ErgonomicsModifier: 0.0,
		ConflictingItems:   []string{},
		Slots: []*weapon_tree.ItemSlot{
			{
				Name: "stock",
				ID:   "slot-stock",
				AllowedItems: []*weapon_tree.Item{
					{
						Name:               "Buffer_Tube",
						ID:                 "item-buffer-tube",
						RecoilModifier:     5.0,
						ErgonomicsModifier: 5.0,
						ConflictingItems:   []string{"item-ADAR-stock"},
						Slots: []*weapon_tree.ItemSlot{
							{
								Name: "stock",
								ID:   "slot-buffer-stock",
								AllowedItems: []*weapon_tree.Item{
									{
										Name:               "good stock",
										ID:                 "item-good-stock",
										RecoilModifier:     22.0,
										ErgonomicsModifier: 22.0,
										ConflictingItems:   []string{"item-ADAR-stock"},
										Slots:              []*weapon_tree.ItemSlot{},
									},
								},
							},
						},
					},
					{
						Name:               "bad stock",
						ID:                 "item-bad-stock",
						RecoilModifier:     5.0,
						ErgonomicsModifier: 5.0,
						ConflictingItems:   []string{"item-ADAR-stock"},
						Slots:              []*weapon_tree.ItemSlot{},
					},
				},
			},
			{
				Name: "receiver",
				ID:   "slot-receiver",
				AllowedItems: []*weapon_tree.Item{
					{
						Name:               "Base_Receiver",
						ID:                 "item-base-receiver",
						RecoilModifier:     0.0,
						ErgonomicsModifier: 0.0,
						ConflictingItems:   []string{},
						Slots:              []*weapon_tree.ItemSlot{},
					},
				},
			},
			{
				Name: "pistol grip",
				ID:   "slot-pistol-grip",
				AllowedItems: []*weapon_tree.Item{
					{
						Name:               "ADAR_Stock",
						ID:                 "item-ADAR-stock",
						RecoilModifier:     20.0,
						ErgonomicsModifier: 20.0,
						ConflictingItems:   []string{"Buffer_Tube", "good stock"},
						Slots:              []*weapon_tree.ItemSlot{},
					},
					{
						Name:               "bad grip",
						ID:                 "item-bad-grip",
						RecoilModifier:     5.0,
						ErgonomicsModifier: 5.0,
						ConflictingItems:   []string{},
						Slots:              []*weapon_tree.ItemSlot{},
					},
				},
			},
		},
	}

	weapon := &weapon_tree.WeaponTree{
		Item: rootItem,
	}

	initialExcluded := make(map[string]bool)

	// Start traversal and get the best build
	bestBuild := findBestBuild(weapon, []OptimalItem{}, "recoil", 0.0, initialExcluded)

	assert.NotNil(t, bestBuild)

	for _, item := range bestBuild.OptimalItems {
		assert.NotEqual(t, "item-ADAR-stock", item.ID)
		assert.NotEqual(t, "item-bad-stock", item.ID)
	}

	evaluation := bestBuild.ToEvaluatedWeapon()
	assert.Equal(t, evaluation.ID, "item-weapon")
	assert.Len(t, evaluation.Slots, 3)
	assert.Equal(t, "item-buffer-tube", evaluation.Slots[0].Item.ID)
	assert.Len(t, evaluation.Slots[0].Item.Slots, 1)
	assert.Equal(t, "item-good-stock", evaluation.Slots[0].Item.Slots[0].Item.ID)
	assert.Equal(t, "item-base-receiver", evaluation.Slots[1].Item.ID)
	assert.Len(t, evaluation.Slots[1].Item.Slots, 0)
	assert.Equal(t, "item-bad-grip", evaluation.Slots[2].Item.ID)
	assert.Len(t, evaluation.Slots[2].Item.Slots, 0)
}
