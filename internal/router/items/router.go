package items_router

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/models"

	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

// getTraderLevelParams parses the trader level parameters from the query string, returning a slice of TraderLevel structs
// for each trader level parameter found in the query string. If a parameter for a trader is found, level is set to 4 (max).
func getTraderLevelParams(c echo.Context) ([]models.TraderLevel, error) {
	traderLevels := make([]models.TraderLevel, 0)

	for _, traderName := range models.TraderNames {
		paramKey := strings.ToLower(fmt.Sprintf("%s_level", traderName))

		value := c.QueryParam(paramKey)
		if value == "" {
			traderLevels = append(traderLevels, models.TraderLevel{Name: traderName, Level: 4})
			continue
		}

		level, err := strconv.Atoi(value)
		if err != nil {
			msg := fmt.Sprintf("Invalid level for trader %s", traderName)
			return nil, errors.New(msg)
		}

		if level > 4 || level < 1 {
			msg := fmt.Sprintf("Invalid level [%d] for trader %s", level, traderName)
			return nil, errors.New(msg)
		}

		traderLevels = append(traderLevels, models.TraderLevel{Name: traderName, Level: level})
	}

	return traderLevels, nil
}

func Bind(e *echo.Group, db *sql.DB) *echo.Group {
	e.GET("/weapons", func(c echo.Context) error {
		res, err := models.GetWeaponsShort(db)
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	e.GET("/weapons/:item_id/calculate", func(c echo.Context) error {
		constraints := models.EvaluationConstraints{
			TraderLevels:     []models.TraderLevel{},
			IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical", "Mount"},
		}

		itemId := c.Param("item_id")
		buildType := c.QueryParam("build_type")
		if budgetParam := c.QueryParam("rub_budget"); budgetParam != "" {
			budget, err := strconv.Atoi(budgetParam)
			if err != nil {
				return c.String(400, "Invalid rub_budget parameter")
			}
			if budget < 0 {
				return c.String(400, "Budget must be non-negative")
			}
			constraints.RubBudget = &budget
		}
		traderLevels, err := getTraderLevelParams(c)
		if err != nil {
			return c.String(400, err.Error())
		}

		constraints.TraderLevels = traderLevels

		build, err := models.GetOptimumBuildByConstraints(db, itemId, buildType, constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get optimum build. item %s, constraints %v", itemId, constraints)
			return c.String(500, err.Error())
		}

		if build == nil {
			if constraints.RubBudget != nil {
				return c.JSON(404, map[string]interface{}{
					"error":  "no_build_within_budget",
					"budget": *constraints.RubBudget,
					"message": fmt.Sprintf("No valid build available within %d RUB for trader levels %v",
						*constraints.RubBudget, constraints.TraderLevels),
				})
			}
			return c.String(404, "Build not found")
		}

		return c.JSON(200, build)
	})

	return e
}
