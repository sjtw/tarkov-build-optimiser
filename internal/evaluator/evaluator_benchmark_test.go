package evaluator

import (
	"fmt"
	"tarkov-build-optimiser/internal/candidate_tree"
	"testing"
)

// sink avoids compiler eliminating results in benchmarks
var sink *Build

// buildSyntheticTree constructs a synthetic candidate tree with repeated subproblems:
// - root has topSlotCount slots
// - each top-level slot has itemsPerTopSlot allowed items
// - each allowed item exposes the same shared child slot, which has childItemsPerSlot allowed items
func buildSyntheticTree(topSlotCount int, itemsPerTopSlot int, childItemsPerSlot int) *candidate_tree.CandidateTree {
	// Shared child slot reused by every allowed item across top-level slots
	commonChild := &candidate_tree.ItemSlot{
		Name:            "child-shared",
		ID:              "slot-child-shared",
		AllowedItems:    make([]*candidate_tree.Item, 0, childItemsPerSlot),
		PotentialValues: candidate_tree.PotentialValues{},
	}
	for i := 0; i < childItemsPerSlot; i++ {
		commonChild.AllowedItems = append(commonChild.AllowedItems, &candidate_tree.Item{
			Name:               fmt.Sprintf("child-item-%d", i),
			ID:                 fmt.Sprintf("item-child-%d", i),
			RecoilModifier:     -1 - (i % 3), // ensure negative potential to avoid skip
			ErgonomicsModifier: 0,
			ConflictingItems:   []candidate_tree.ConflictingItem{},
			Slots:              []*candidate_tree.ItemSlot{},
		})
	}

	// Top-level slots
	topSlots := make([]*candidate_tree.ItemSlot, 0, topSlotCount)
	for s := 0; s < topSlotCount; s++ {
		slot := &candidate_tree.ItemSlot{
			Name:         fmt.Sprintf("slot-top-%d", s),
			ID:           fmt.Sprintf("slot-top-%d", s),
			AllowedItems: make([]*candidate_tree.Item, 0, itemsPerTopSlot),
		}
		for i := 0; i < itemsPerTopSlot; i++ {
			slot.AllowedItems = append(slot.AllowedItems, &candidate_tree.Item{
				Name:               fmt.Sprintf("top-%d-item-%d", s, i),
				ID:                 fmt.Sprintf("item-top-%d-%d", s, i),
				RecoilModifier:     -1 - (i % 2), // negative to explore
				ErgonomicsModifier: 0,
				ConflictingItems:   []candidate_tree.ConflictingItem{},
				Slots:              []*candidate_tree.ItemSlot{commonChild}, // shared subtree → repeated subproblems
			})
		}
		topSlots = append(topSlots, slot)
	}

	rootItem := &candidate_tree.Item{
		Name:               "SyntheticWeapon",
		ID:                 "item-synthetic-weapon",
		RecoilModifier:     0,
		ErgonomicsModifier: 0,
		ConflictingItems:   []candidate_tree.ConflictingItem{},
		Slots:              topSlots,
	}
	weapon := &candidate_tree.CandidateTree{Item: rootItem}
	weapon.Item.CalculatePotentialValues()
	return weapon
}

// Benchmark with a cold memo (new map every iteration) which forces full evaluation work each time.
func BenchmarkProcessSlots_ColdMemo(b *testing.B) {
	weapon := buildSyntheticTree(4, 8, 12) // moderately large branching
	weapon.UpdateAllowedItemSlots()
	weapon.UpdateAllowedItems()
	desc := precomputeSlotDescendantItemIDs(weapon)

	slots := weapon.Item.Slots
	chosen := []OptimalItem{}
	excluded := map[string]bool{}
	visited := map[string]bool{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memo := map[string]*Build{}
		sink = processSlots(weapon, slots, chosen, "recoil", 0, 0, excluded, visited, memo, desc)
	}
}

// Benchmark with a warm memo, pre-seeded for the top-level subproblem, so the function returns immediately.
func BenchmarkProcessSlots_WarmMemo(b *testing.B) {
	weapon := buildSyntheticTree(4, 8, 12)
	weapon.UpdateAllowedItemSlots()
	weapon.UpdateAllowedItems()
	desc := precomputeSlotDescendantItemIDs(weapon)

	slots := weapon.Item.Slots
	chosen := []OptimalItem{}
	excluded := map[string]bool{}
	visited := map[string]bool{}

	// Seed memo with a result for the top-level subproblem so we hit it immediately
	memo := map[string]*Build{}
	topKey := makeMemoKey("recoil", slots[0].ID, slots[1:], excluded, desc)
	memo[topKey] = &Build{EvaluationType: "recoil", OptimalItems: []OptimalItem{{ID: "memo-top"}}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = processSlots(weapon, slots, chosen, "recoil", 0, 0, excluded, visited, memo, desc)
	}
}
