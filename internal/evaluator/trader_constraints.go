package evaluator

import (
	"fmt"
	"tarkov-build-optimiser/internal/helpers"
	"tarkov-build-optimiser/internal/models"
)

func validateConstraints(offers []models.TraderOffer, constraints models.EvaluationConstraints) bool {
	for i := 0; i < len(offers); i++ {
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
	totalCombinations := helpers.Pow(4, traderCount)

	var traders [][]models.TraderLevel
	for i := 0; i < totalCombinations; i++ {
		var combination []models.TraderLevel

		for j := 0; j < traderCount; j++ {
			level := (i / helpers.Pow(5, j) % 5) + 1
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
