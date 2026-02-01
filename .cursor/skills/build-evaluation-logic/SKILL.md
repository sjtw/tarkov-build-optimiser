---
name: build-evaluation-logic
description: Understanding and extending the weapon build optimization algorithm
---

# Build Evaluation Logic

The evaluator is responsible for finding the "best" weapon configuration based on specific criteria (Recoil or Ergonomics) and constraints (Trader Levels).

## Core Concepts

- **Candidate Tree**: A recursive structure representing a weapon and all allowed items for its slots.
- **Trader Level Variations**: Different combinations of trader levels (e.g., all level 1, all level 2, etc.) are evaluated to provide optimal builds for different player progressions.
- **Pruning**: The algorithm uses lower/upper bounds to skip branches that cannot possibly beat the current best build.
- **Conflict Handling**: Items can conflict with each other. The algorithm tracks `excludedItems` to ensure a valid configuration.
- **Caching**: Conflict-free items (those with no conflicts) can have their subtree results cached to speed up evaluation.

## Key Components

### 1. Worker Pool (`evaluate` function)
The evaluator uses a worker pool to process multiple weapons and trader level variations in parallel.

```go
workerCount := runtime.NumCPU() * environment.EvaluatorPoolSizeFactor
inputChan := make(chan Candidateinput, workerCount*2)
// ...
for i := 0; i < workerCount; i++ {
    go func() { /* evaluation logic */ }()
}
```

### 2. Search Algorithm (`processSlots` function)
A depth-first search that explores the possible items for each slot.

- **`focusedStat`**: Either "recoil" (aiming for minimum) or "ergonomics" (aiming for maximum).
- **Pruning Logic**: Uses `computeRecoilLowerBound` or `computeErgoUpperBound` to decide if a branch is worth exploring.

### 3. Caching (`Cache` interface)
Supports both memory and database caching for "conflict-free" subtrees.

## When to Update

- **New Stats**: If you want to optimize for a new stat (e.g., weight, price).
- **Algorithm Improvements**: If you find a more efficient way to prune the search tree.
- **Constraint Changes**: If you need to add new types of constraints (e.g., specific item blacklists).

## Troubleshooting

- **High Memory Usage**: Check if the `resultsChan` is being flushed correctly.
- **Slow Evaluation**: 
  - Ensure `test:mode` is used for development.
  - Verify that pruning is actually skipping branches.
  - Check cache hit rates in the logs.
- **Non-Optimal Results**: Verify the `doesImproveStats` logic and ensuring `UpdateAllowedItems()` is called on the weapon tree.
