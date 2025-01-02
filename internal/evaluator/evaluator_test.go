package evaluator

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"tarkov-build-optimiser/internal/candidate_tree"
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

type MockEvaluationDataProvider struct {
	GetSubtreeFunc     func(itemId string, buildType string, constraints models.EvaluationConstraints) (*models.ItemEvaluationResult, error)
	GetTraderOfferFunc func(itemID string) ([]models.TraderOffer, error)
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

//func (m *MockEvaluationDataProvider) SaveBuild(build *models.ItemEvaluationResult, constraints models.EvaluationConstraints) {
//	if m.SaveBuildFunc != nil {
//		m.SaveBuildFunc(build, constraints)
//	}
//}

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

func TestFindBestBuild(t *testing.T) {
	// TODO - use construction functions and mock data provider to construct this
	rootItem := &candidate_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     -0,
		ErgonomicsModifier: -0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots: []*candidate_tree.ItemSlot{
			{
				Name: "pistol grip",
				ID:   "slot-pistol-grip",
				AllowedItems: []*candidate_tree.Item{
					{
						Name:               "ADAR_Stock",
						ID:                 "item-ADAR-stock",
						RecoilModifier:     -20,
						ErgonomicsModifier: -20,
						ConflictingItems: []candidate_tree.ConflictingItem{
							{ID: "item-ADAR-stock", Name: "ADAR_Stock", CategoryID: "stock", CategoryName: "Stock"},
							{ID: "item-good-stock", Name: "Good Stock", CategoryID: "stock", CategoryName: "Stock"},
						},
						Slots: []*candidate_tree.ItemSlot{},
					},
					{
						Name:               "bad grip",
						ID:                 "item-bad-grip",
						RecoilModifier:     -5,
						ErgonomicsModifier: 1,
						ConflictingItems:   []candidate_tree.ConflictingItem{},
						Slots:              []*candidate_tree.ItemSlot{},
					},
					//{
					//	Name:               "bad grip 2",
					//	ID:                 "item-bad-grip-2",
					//	RecoilModifier:     -5,
					//	ErgonomicsModifier: 2,
					//	ConflictingItems:   []candidate_tree.ConflictingItem{},
					//	Slots:              []*candidate_tree.ItemSlot{},
					//},
				},
			},
			{
				Name: "stock",
				ID:   "slot-stock",
				AllowedItems: []*candidate_tree.Item{
					{
						Name:               "Buffer_Tube",
						ID:                 "item-buffer-tube",
						RecoilModifier:     -5,
						ErgonomicsModifier: -5,
						ConflictingItems: []candidate_tree.ConflictingItem{
							{ID: "item-ADAR-stock", Name: "ADAR_Stock", CategoryID: "stock", CategoryName: "Stock"},
						},
						Slots: []*candidate_tree.ItemSlot{
							{
								Name: "stock",
								ID:   "slot-buffer-stock",
								AllowedItems: []*candidate_tree.Item{
									{
										Name:               "good stock",
										ID:                 "item-good-stock",
										RecoilModifier:     -22,
										ErgonomicsModifier: 10,
										ConflictingItems: []candidate_tree.ConflictingItem{
											{ID: "item-ADAR-stock", Name: "ADAR_Stock", CategoryID: "stock", CategoryName: "Stock"},
											{ID: "item-useless-mount2", Name: "item-useless-mount2", CategoryID: "stock", CategoryName: "Stock"},
											{ID: "non-existent-item", Name: "Non Existent Item", CategoryID: "stock", CategoryName: "Stock"},
										},
										Slots: []*candidate_tree.ItemSlot{},
									},
								},
							},
						},
					},
					{
						Name:               "bad stock",
						ID:                 "item-bad-stock",
						RecoilModifier:     -5,
						ErgonomicsModifier: 5,
						ConflictingItems: []candidate_tree.ConflictingItem{
							{ID: "item-ADAR-stock", Name: "ADAR_Stock", CategoryID: "stock", CategoryName: "Stock"},
						},
						Slots: []*candidate_tree.ItemSlot{},
					},
					{
						Name:               "great stock - not allowed",
						ID:                 "item-great-stock",
						RecoilModifier:     -25,
						ErgonomicsModifier: 8,
						ConflictingItems: []candidate_tree.ConflictingItem{
							{ID: "item-ADAR-stock", Name: "ADAR_Stock", CategoryID: "stock", CategoryName: "Stock"},
						},
						Slots: []*candidate_tree.ItemSlot{},
					},
				},
			},
			{
				Name: "receiver",
				ID:   "slot-receiver",
				AllowedItems: []*candidate_tree.Item{
					{
						Name:               "Base_Receiver",
						ID:                 "item-base-receiver",
						RecoilModifier:     -1,
						ErgonomicsModifier: 0,
						ConflictingItems:   []candidate_tree.ConflictingItem{},
						Slots: []*candidate_tree.ItemSlot{
							{
								Name: "Scope",
								ID:   "slot-receiver-scope",
								AllowedItems: []*candidate_tree.Item{
									{
										Name:               "ACOG",
										ID:                 "item-acog",
										RecoilModifier:     -100,
										ErgonomicsModifier: 100,
										ConflictingItems:   []candidate_tree.ConflictingItem{},
										Slots:              []*candidate_tree.ItemSlot{},
									},
								},
							},
							{
								Name: "Foregrip",
								ID:   "slot-receiver-foregrip",
								AllowedItems: []*candidate_tree.Item{
									{
										Name:               "bad-foregrip",
										ID:                 "item-bad-foregrip",
										RecoilModifier:     -1,
										ErgonomicsModifier: 1,
										ConflictingItems:   []candidate_tree.ConflictingItem{},
										Slots:              []*candidate_tree.ItemSlot{},
									},
								},
							},
							{
								Name: "Mount",
								ID:   "slot-receiver-mount",
								AllowedItems: []*candidate_tree.Item{
									{
										Name:               "Useless Mount",
										ID:                 "item-useless-mount",
										RecoilModifier:     -5,
										ErgonomicsModifier: 0,
										ConflictingItems:   []candidate_tree.ConflictingItem{},
										Slots:              []*candidate_tree.ItemSlot{},
									},
									{
										Name:               "Useless Mount2",
										ID:                 "item-useless-mount2",
										RecoilModifier:     -5,
										ErgonomicsModifier: 0,
										ConflictingItems:   []candidate_tree.ConflictingItem{},
										Slots:              []*candidate_tree.ItemSlot{},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	weapon := &candidate_tree.CandidateTree{
		Item: rootItem,
	}

	initialExcluded := make(map[string]bool)
	initialExcluded["item-acog"] = true

	weapon.Item.CalculatePotentialValues()

	// Start traversal and get the best build
	bestBuild := FindBestBuild(weapon, "recoil", initialExcluded)

	assert.NotNil(t, bestBuild)

	for _, item := range bestBuild.OptimalItems {
		assert.NotEqual(t, "item-ADAR-stock", item.ID)
		assert.NotEqual(t, "item-bad-stock", item.ID)
	}

	evaluation, err := bestBuild.ToEvaluatedWeapon()
	if err != nil {
		t.Fatal(err, "Failed to convert best build to evaluated weapon")
	}
	assert.Equal(t, evaluation.ID, "item-weapon")
	assert.Len(t, evaluation.Slots, 3)

	// stock -> buffer tube
	assert.Equal(t, "item-buffer-tube", evaluation.Slots[1].Item.ID)
	assert.Equal(t, -5, evaluation.Slots[1].Item.RecoilModifier)
	assert.Equal(t, -5, evaluation.Slots[1].Item.ErgonomicsModifier)
	assert.Len(t, evaluation.Slots[1].Item.Slots, 1)

	// stock  -> buffer tube -> stock
	assert.NotNil(t, evaluation.Slots[1].Item.Slots[0].Item)
	assert.Equal(t, "item-good-stock", evaluation.Slots[1].Item.Slots[0].Item.ID)
	assert.Equal(t, -22, evaluation.Slots[1].Item.Slots[0].Item.RecoilModifier)
	assert.Equal(t, 10, evaluation.Slots[1].Item.Slots[0].Item.ErgonomicsModifier)

	// receiver
	assert.NotNil(t, evaluation.Slots[2].Item)
	assert.Equal(t, "item-base-receiver", evaluation.Slots[2].Item.ID)
	assert.Equal(t, -1, evaluation.Slots[2].Item.RecoilModifier)
	assert.Equal(t, 0, evaluation.Slots[2].Item.ErgonomicsModifier)

	assert.Len(t, evaluation.Slots[2].Item.Slots, 3)

	// receiver scope
	assert.Nil(t, evaluation.Slots[2].Item.Slots[0].Item)

	// receiver foregrip
	assert.NotNil(t, evaluation.Slots[2].Item.Slots[1].Item)
	assert.Equal(t, "item-bad-foregrip", evaluation.Slots[2].Item.Slots[1].Item.ID)
	assert.Equal(t, -1, evaluation.Slots[2].Item.Slots[1].Item.RecoilModifier)
	assert.Equal(t, 1, evaluation.Slots[2].Item.Slots[1].Item.ErgonomicsModifier)

	// receiver mount
	//assert.Nil(t, evaluation.Slots[2].Item.Slots[2].Item)

	// pistol grip
	assert.NotNil(t, evaluation.Slots[0].Item)
	assert.Equal(t, "item-bad-grip", evaluation.Slots[0].Item.ID)

	assert.Equal(t, -5, evaluation.Slots[0].Item.RecoilModifier)
	assert.Equal(t, 1, evaluation.Slots[0].Item.ErgonomicsModifier)
	assert.Len(t, evaluation.Slots[0].Item.Slots, 0)
}
