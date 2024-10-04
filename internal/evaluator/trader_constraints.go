package evaluator

import (
	"fmt"
	"tarkov-build-optimiser/internal/helpers"
)

type TraderLevel struct {
	Name  string
	Level int
}

var TraderNames = []string{"Jaeger", "Prapor", "Peacekeeper", "Mechanic", "Skier"}

func GenerateTraderLevelVariations(traderNames []string) [][]TraderLevel {
	traderCount := len(traderNames)
	totalCombinations := helpers.Pow(5, traderCount)

	var traders [][]TraderLevel
	for i := 0; i < totalCombinations; i++ {
		var combination []TraderLevel

		for j := 0; j < traderCount; j++ {
			level := (i / helpers.Pow(5, j) % 5) + 1
			combination = append(combination, TraderLevel{Name: traderNames[j], Level: level})
		}

		traders = append(traders, combination)
	}

	return traders
}

func createTraderLevelHash(traders []TraderLevel) string {
	result := ""
	for _, trader := range traders {
		result += fmt.Sprintf("%s:%d,", trader.Name, trader.Level)
	}
	return result
}
