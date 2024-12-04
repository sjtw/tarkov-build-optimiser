package evaluator

import (
	"fmt"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
)

func validateConstraints(offers []models.TraderOffer, constraints models.EvaluationConstraints) bool {
	for i := 0; i < len(offers); i++ {
		o := offers[i]
		if o.Trader == "" && o.MinTraderLevel == 0 {
			// TODO: I think this can be assumed to be a preset item? Worth checking for edge cases nonetheless
			return false
		}

		for j := i + 1; j < len(constraints.TraderLevels); j++ {
			tc := constraints.TraderLevels[j]
			if offers[i].Trader == tc.Name && tc.Level >= offers[i].MinTraderLevel {
				return true
			}
		}
	}

	return false
}

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

func createTraderLevelHash(traders []models.TraderLevel) string {
	result := ""
	for _, trader := range traders {
		result += fmt.Sprintf("%s:%d,", trader.Name, trader.Level)
	}
	return result
}
