package evaluator

import (
	"context"
	"sync"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
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
	GetItemPriceFunc   func(itemID string) (int, bool, error)
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

func (m *MockEvaluationDataProvider) GetItemPrice(ctx context.Context, itemID string, traderLevels []models.TraderLevel) (int, bool, error) {
	if m.GetItemPriceFunc != nil {
		return m.GetItemPriceFunc(itemID)
	}
	return 0, false, nil
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
	cache := NewMemoryCache()
	bestBuild := FindBestBuild(weapon, "recoil", initialExcluded, cache, nil)

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

func TestFindBestBuild_ErgonomicsFocus_SelectsHighestErgo(t *testing.T) {
	// One slot with two options: one is better for ergonomics, the other better for recoil.
	slot := &candidate_tree.ItemSlot{
		Name: "grip",
		ID:   "slot-grip",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "ergo-better",
				ID:                 "item-ergo-better",
				RecoilModifier:     -1, // small recoil benefit to avoid recoil-based skipping
				ErgonomicsModifier: 10, // best ergonomics
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "recoil-better",
				ID:                 "item-recoil-better",
				RecoilModifier:     -10, // better recoil
				ErgonomicsModifier: 1,   // worse ergonomics
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots:              []*candidate_tree.ItemSlot{slot},
	}

	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "ergonomics", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}

	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}

	if eval.Slots[0].Item == nil {
		t.Fatalf("expected grip slot to be filled")
	}
	if eval.Slots[0].Item.ID != "item-ergo-better" {
		t.Fatalf("expected ergonomics-focused build to pick ergo-better, got %s", eval.Slots[0].Item.ID)
	}
}

func TestFindBestBuild_RecoilFocus_TieBreaksOnErgonomics(t *testing.T) {
	// One slot with two items having equal recoil but different ergonomics
	slot := &candidate_tree.ItemSlot{
		Name: "slot-a",
		ID:   "slot-a",
		AllowedItems: []*candidate_tree.Item{
			{ // worse ergonomics
				Name:               "equal-recoil-low-ergo",
				ID:                 "item-low-ergo",
				RecoilModifier:     -5,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{ // better ergonomics
				Name:               "equal-recoil-high-ergo",
				ID:                 "item-high-ergo",
				RecoilModifier:     -5,
				ErgonomicsModifier: 5,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots:              []*candidate_tree.ItemSlot{slot},
	}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "recoil", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}

	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if eval.Slots[0].Item == nil || eval.Slots[0].Item.ID != "item-high-ergo" {
		t.Fatalf("expected high ergonomics item on recoil tie-break, got %+v", eval.Slots[0].Item)
	}
}

func TestFindBestBuild_RespectsExcludedItems(t *testing.T) {
	slot := &candidate_tree.ItemSlot{
		Name: "slot-a",
		ID:   "slot-a",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "best",
				ID:                 "item-best",
				RecoilModifier:     -10,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "second",
				ID:                 "item-second",
				RecoilModifier:     -9,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots:              []*candidate_tree.ItemSlot{slot},
	}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	excluded := map[string]bool{"item-best": true}
	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "recoil", excluded, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if eval.Slots[0].Item == nil || eval.Slots[0].Item.ID != "item-second" {
		t.Fatalf("expected excluded best item to be skipped, got %+v", eval.Slots[0].Item)
	}
}

func TestFindBestBuild_SkipsSlotIfBetterGlobal(t *testing.T) {
	// Two top-level slots. Slot S1 has an item that conflicts with a very good item in S2.
	// Best global choice is to leave S1 empty so S2 can pick the very good item.
	s1 := &candidate_tree.ItemSlot{
		Name: "S1",
		ID:   "slot-s1",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "conflicting",
				ID:                 "item-conflict",
				RecoilModifier:     -5,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{{ID: "item-very-good", Name: "very good", CategoryID: "x", CategoryName: "X"}},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}
	s2 := &candidate_tree.ItemSlot{
		Name: "S2",
		ID:   "slot-s2",
		AllowedItems: []*candidate_tree.Item{
			{ // very good but conflicts with S1's item
				Name:               "very good",
				ID:                 "item-very-good",
				RecoilModifier:     -50,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "ok",
				ID:                 "item-ok",
				RecoilModifier:     -10,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{
		Name:               "Weapon",
		ID:                 "item-weapon",
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots:              []*candidate_tree.ItemSlot{s1, s2},
	}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "recoil", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}

	// Expect S1 empty, S2 picks very good
	if eval.Slots[0].Item != nil {
		t.Fatalf("expected S1 to be empty, got %+v", eval.Slots[0].Item)
	}
	if eval.Slots[1].Item == nil || eval.Slots[1].Item.ID != "item-very-good" {
		t.Fatalf("expected S2 to pick very good, got %+v", eval.Slots[1].Item)
	}
}

func TestFindBestBuild_RecoilFocus_CrossSlotSynergy_KnownLimitation(t *testing.T) {
	// Now enabled: branch-and-bound replaces unsafe early break
	// S1 has three items: A (-10) conflicts with both C and C2; B (-8) conflicts with C2 only; E (-6) conflicts with none.
	// S2 has C (-40), C2 (-50), D (-5).
	// Global optimum is E + C2 (-56), but after evaluating B which yields -48, current code breaks early and misses E.

	s1 := &candidate_tree.ItemSlot{
		Name: "S1",
		ID:   "slot-s1",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "A",
				ID:                 "item-A",
				RecoilModifier:     -10,
				ErgonomicsModifier: 0,
				ConflictingItems: []candidate_tree.ConflictingItem{
					{ID: "item-C", Name: "C", CategoryID: "x", CategoryName: "X"},
					{ID: "item-C2", Name: "C2", CategoryID: "x", CategoryName: "X"},
				},
				Slots: []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "B",
				ID:                 "item-B",
				RecoilModifier:     -8,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{{ID: "item-C2", Name: "C2", CategoryID: "x", CategoryName: "X"}},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "E",
				ID:                 "item-E",
				RecoilModifier:     -6,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}
	s2 := &candidate_tree.ItemSlot{
		Name: "S2",
		ID:   "slot-s2",
		AllowedItems: []*candidate_tree.Item{
			{Name: "C", ID: "item-C", RecoilModifier: -40, ErgonomicsModifier: 0, ConflictingItems: []candidate_tree.ConflictingItem{}, Slots: []*candidate_tree.ItemSlot{}},
			{Name: "C2", ID: "item-C2", RecoilModifier: -50, ErgonomicsModifier: 0, ConflictingItems: []candidate_tree.ConflictingItem{}, Slots: []*candidate_tree.ItemSlot{}},
			{Name: "D", ID: "item-D", RecoilModifier: -5, ErgonomicsModifier: 0, ConflictingItems: []candidate_tree.ConflictingItem{}, Slots: []*candidate_tree.ItemSlot{}},
		},
	}
	rootItem := &candidate_tree.Item{Name: "Weapon", ID: "item-weapon", Slots: []*candidate_tree.ItemSlot{s1, s2}}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "recoil", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if eval.Slots[0].Item == nil || eval.Slots[0].Item.ID != "item-E" || eval.Slots[1].Item == nil || eval.Slots[1].Item.ID != "item-C2" {
		t.Fatalf("expected E + C2, got S1=%+v S2=%+v", eval.Slots[0].Item, eval.Slots[1].Item)
	}
}

func TestToEvaluatedWeapon_AggregatesConflicts(t *testing.T) {
	// Select an item that declares conflicts; ensure conflicts propagate to EvaluatedWeapon.Conflicts
	slot := &candidate_tree.ItemSlot{
		Name: "slot-a",
		ID:   "slot-a",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "with-conflict",
				ID:                 "item-conflict-declarer",
				RecoilModifier:     -1,
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{{ID: "item-some-conflict", Name: "some", CategoryID: "cat", CategoryName: "Cat"}},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}
	rootItem := &candidate_tree.Item{Name: "Weapon", ID: "item-weapon", Slots: []*candidate_tree.ItemSlot{slot}}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "recoil", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if len(eval.Conflicts) == 0 {
		t.Fatalf("expected conflicts aggregated on evaluated weapon; none found")
	}
	found := false
	for _, c := range eval.Conflicts {
		if c.ID == "item-some-conflict" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected specific conflict id to be present, got %+v", eval.Conflicts)
	}
}

func TestFindBestBuild_ErgonomicsFocus_AllowsRecoilWorsening_KnownBug(t *testing.T) {
	// Now enabled: ergonomics gating uses ergonomics potential instead of recoil
	// One slot: item E improves ergonomics but worsens recoil; item R improves recoil slightly but no ergonomics.
	slot := &candidate_tree.ItemSlot{
		Name: "slot-a",
		ID:   "slot-a",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "ergo-only",
				ID:                 "item-ergo-only",
				RecoilModifier:     5,  // worse recoil
				ErgonomicsModifier: 12, // better ergo
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "recoil-only",
				ID:                 "item-recoil-only",
				RecoilModifier:     -1, // better recoil
				ErgonomicsModifier: 0,  // no ergo improvement
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{Name: "Weapon", ID: "item-weapon", Slots: []*candidate_tree.ItemSlot{slot}}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "ergonomics", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if eval.Slots[0].Item == nil || eval.Slots[0].Item.ID != "item-ergo-only" {
		t.Fatalf("expected ergonomics-focused build to pick ergo-only despite recoil worsening; got %+v", eval.Slots[0].Item)
	}
}

func TestFindBestBuild_ErgonomicsFocus_TieBreaksOnRecoil(t *testing.T) {
	// One slot with two items having equal ergonomics but different recoil; ergonomics focus should
	// pick the one with lower recoil as a tie-break.
	slot := &candidate_tree.ItemSlot{
		Name: "slot-a",
		ID:   "slot-a",
		AllowedItems: []*candidate_tree.Item{
			{
				Name:               "ergo-5-recoil-10",
				ID:                 "item-ergo5-recoil10",
				RecoilModifier:     -10,
				ErgonomicsModifier: 5,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
			{
				Name:               "ergo-5-recoil-5",
				ID:                 "item-ergo5-recoil5",
				RecoilModifier:     -5,
				ErgonomicsModifier: 5,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{},
			},
		},
	}

	rootItem := &candidate_tree.Item{Name: "Weapon", ID: "item-weapon", Slots: []*candidate_tree.ItemSlot{slot}}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()

	cache := NewMemoryCache()
	best := FindBestBuild(weapon, "ergonomics", map[string]bool{}, cache, nil)
	if best == nil {
		t.Fatalf("expected build, got nil")
	}
	eval, err := best.ToEvaluatedWeapon()
	if err != nil {
		t.Fatalf("ToEvaluatedWeapon failed: %v", err)
	}
	if eval.Slots[0].Item == nil || eval.Slots[0].Item.ID != "item-ergo5-recoil10" {
		t.Fatalf("expected ergonomics tie-break to choose lower recoil, got %+v", eval.Slots[0].Item)
	}
}
