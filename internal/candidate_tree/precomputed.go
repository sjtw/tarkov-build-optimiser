package candidate_tree

import "tarkov-build-optimiser/internal/models"

// PrecomputedSubtreeInfo represents a compact summary of a precomputed subtree result
// for a given root mod/item.
type PrecomputedSubtreeInfo struct {
	RootItemID     string
	EvaluationType string
	RecoilSum      int
	ErgonomicsSum  int
	IsDefinitive   bool
}

// PrecomputedSubtreeProvider provides precomputed subtree information for items.
// Implementations can back this with an in-memory map or database.
type PrecomputedSubtreeProvider interface {
	GetPrecomputedSubtree(itemID string, focusedStat string, constraints models.EvaluationConstraints) (PrecomputedSubtreeInfo, bool)
}

// ApplyPrecomputedPruning traverses the candidate tree and, for any slot where a precomputed
// subtree is available for one or more allowed items, prunes sibling allowed items to prefer
// the precomputed optimum.
//
// Rules:
// - If exactly one allowed item has a precomputed result marked IsDefinitive, keep only that item
// - If multiple have results, keep the best for the focused stat (recoil lower is better; ergonomics higher is better)
// - Otherwise leave the slot as-is
func ApplyPrecomputedPruning(tree *CandidateTree, focusedStat string, provider PrecomputedSubtreeProvider) {
	if tree == nil || tree.Item == nil || provider == nil {
		return
	}
	pruneItemWithPrecomputed(tree.Item, focusedStat, provider, tree.Constraints)
}

func pruneItemWithPrecomputed(item *Item, focusedStat string, provider PrecomputedSubtreeProvider, constraints models.EvaluationConstraints) {
	if item == nil {
		return
	}
	for _, slot := range item.Slots {
		pruneSlotWithPrecomputed(slot, focusedStat, provider, constraints)
		// Recurse into allowed items after pruning
		for _, ai := range slot.AllowedItems {
			pruneItemWithPrecomputed(ai, focusedStat, provider, constraints)
		}
	}
}

func pruneSlotWithPrecomputed(slot *ItemSlot, focusedStat string, provider PrecomputedSubtreeProvider, constraints models.EvaluationConstraints) {
	if slot == nil || len(slot.AllowedItems) == 0 {
		return
	}

	type candidate struct {
		item *Item
		info PrecomputedSubtreeInfo
		ok   bool
	}
	candidates := make([]candidate, 0, len(slot.AllowedItems))
	for _, ai := range slot.AllowedItems {
		info, ok := provider.GetPrecomputedSubtree(ai.ID, focusedStat, constraints)
		candidates = append(candidates, candidate{item: ai, info: info, ok: ok})
	}

	// Collect those with precomputed data
	precomp := make([]precompCandidate, 0)
	for _, c := range candidates {
		if c.ok {
			precomp = append(precomp, precompCandidate{item: c.item, info: c.info, ok: c.ok})
		}
	}

	if len(precomp) == 0 {
		return
	}

	// Prefer definitive single result
	definitive := make([]precompCandidate, 0)
	for _, c := range precomp {
		if c.info.IsDefinitive {
			definitive = append(definitive, c)
		}
	}

	var chosen *Item
	if len(definitive) == 1 {
		chosen = definitive[0].item
	} else if len(definitive) > 1 {
		chosen = bestByFocusCandidates(definitive, focusedStat)
	} else {
		// No definitive, choose best among available precomputed
		chosen = bestByFocusCandidates(precomp, focusedStat)
	}

	if chosen != nil {
		slot.AllowedItems = []*Item{chosen}
	}
}

type precompCandidate struct {
	item *Item
	info PrecomputedSubtreeInfo
	ok   bool
}

func bestByFocusCandidates(cs []precompCandidate, focusedStat string) *Item {
	if len(cs) == 0 {
		return nil
	}
	best := cs[0]
	for i := 1; i < len(cs); i++ {
		if focusedStat == "recoil" {
			if cs[i].info.RecoilSum < best.info.RecoilSum {
				best = cs[i]
			} else if cs[i].info.RecoilSum == best.info.RecoilSum && cs[i].info.ErgonomicsSum > best.info.ErgonomicsSum {
				best = cs[i]
			}
		} else {
			if cs[i].info.ErgonomicsSum > best.info.ErgonomicsSum {
				best = cs[i]
			} else if cs[i].info.ErgonomicsSum == best.info.ErgonomicsSum && cs[i].info.RecoilSum < best.info.RecoilSum {
				best = cs[i]
			}
		}
	}
	return best.item
}
