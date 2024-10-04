package evaluator

import (
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/helpers"
	"testing"
)

func TestGenerateVariations(t *testing.T) {
	traderLevels := GenerateTraderLevelVariations(TraderNames)

	expectedCombinations := helpers.Pow(5, len(TraderNames))
	assert.Len(t, traderLevels, expectedCombinations)
	combinationMap := make(map[string]bool)

	for _, combination := range traderLevels {
		key := createTraderLevelHash(combination)

		for _, trader := range combination {
			assert.GreaterOrEqual(t, trader.Level, 1, "Level is below 1 for trader: %s", trader.Name)
			assert.LessOrEqual(t, trader.Level, 5, "Level is above 5 for trader: %s", trader.Name)
		}

		assert.False(t, combinationMap[key], "Duplicate combination found: %v", combination)

		combinationMap[key] = true
	}

	assert.Len(t, combinationMap, expectedCombinations, "Some combinations are missing or duplicated.")
}
