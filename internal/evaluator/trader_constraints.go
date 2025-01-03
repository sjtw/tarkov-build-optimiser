package evaluator

import (
	"fmt"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
)

func validateTraderLevels(offers []models.TraderOffer, levels []models.TraderLevel) bool {
	for i := 0; i < len(offers); i++ {
		o := offers[i]
		if o.Trader == "" && o.MinTraderLevel == 0 {
			// TODO: I think this can be assumed to be a preset item? Worth checking for edge cases nonetheless
			continue
		}

		for j := 0; j < len(levels); j++ {
			if offers[i].Trader == levels[j].Name && levels[j].Level >= offers[i].MinTraderLevel {
				return true
			}
		}
	}

	return false
}

// GenerateTraderLevelVariations generates all possible combinations of trader levels for the given trader names
func GenerateTraderLevelVariations(traderNames []string) [][]models.TraderLevel {
	traderCount := len(traderNames)
	maxLevel := 4
	totalCombinations := helpers.Pow(maxLevel, traderCount)

	var traders [][]models.TraderLevel
	for i := 0; i < totalCombinations; i++ {
		var combination []models.TraderLevel

		for j := 0; j < traderCount; j++ {
			level := (i / helpers.Pow(maxLevel, j) % maxLevel) + 1
			combination = append(combination, models.TraderLevel{Name: traderNames[j], Level: level})
		}

		traders = append(traders, combination)
	}

	return traders
}

// createTraderLevelHash creates a hash string from the given trader level combinations
// TODO: include game data version in the hash for keying stored builds?
func createTraderLevelHash(traders []models.TraderLevel) string {
	result := ""
	for _, trader := range traders {
		result += fmt.Sprintf("%s:%d,", trader.Name, trader.Level)
	}
	return result
}
