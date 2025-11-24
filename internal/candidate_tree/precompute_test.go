package candidate_tree

import (
	"tarkov-build-optimiser/internal/models"
	"testing"
)

// test fake provider must be declared at package level; methods with receivers
// cannot be declared inside a function in Go.
type fakeProvider struct{}

func (f fakeProvider) GetPrecomputedSubtree(itemID string, focusedStat string, constraints models.EvaluationConstraints) (PrecomputedSubtreeInfo, bool) {
	if itemID == "item-A" {
		return PrecomputedSubtreeInfo{RootItemID: itemID, EvaluationType: focusedStat, RecoilSum: -100, ErgonomicsSum: 0, IsDefinitive: true}, true
	}
	return PrecomputedSubtreeInfo{}, false
}

// Failing TDD test: when a precomputed subtree exists for an allowed item and
// is compatible, the candidate tree preprocessing should prune sibling allowed items
// under that slot and keep only the precomputed root.
// This will fail until precomputed subtree reuse is integrated into CandidateTree construction.
func TestCandidateTree_PrunesAllowedItemsWhenPrecomputedSubtreeIsOptimal(t *testing.T) {
	// Build a minimal item with one slot S that has two allowed items A and B.
	// Assume precomputed subtree for A is available and optimal for this slot.
	slot := &ItemSlot{
		Name:         "S",
		ID:           "slot-S",
		AllowedItems: []*Item{{ID: "item-A", Name: "A"}, {ID: "item-B", Name: "B"}},
	}
	root := &Item{ID: "W", Name: "W", Slots: []*ItemSlot{slot}}
	tree := &CandidateTree{Item: root}
	ApplyPrecomputedPruning(tree, "recoil", fakeProvider{})

	// In the desired implementation, CandidateTree construction would consult a
	// precomputed-subtree store and prune B if A's cached subtree is optimal.
	// For now, assert the expected end-state (and fail): only A remains.
	if len(tree.Item.Slots[0].AllowedItems) != 1 || tree.Item.Slots[0].AllowedItems[0].ID != "item-A" {
		t.Fatalf("expected candidate tree to keep only precomputed item A; got %+v", tree.Item.Slots[0].AllowedItems)
	}
}
