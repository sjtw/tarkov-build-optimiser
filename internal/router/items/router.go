package items_router

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/weapon_tree"

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
			IgnoredSlotNames: []string{"Scope", "Ubgl"},
		}

		itemId := c.Param("item_id")
		buildType := c.QueryParam("build_type")
		traderLevels, err := getTraderLevelParams(c)
		if err != nil {
			return c.String(400, err.Error())
		}

		constraints.TraderLevels = traderLevels
		//build, err := models.GetOptimumBuild(db, itemId, buildType, constraints)
		//if err != nil {
		//	return err
		//}

		//if build != nil {
		//	log.Info().Msg("Returning pre-generated build")
		//	return c.JSON(200, build)
		//}

		log.Info().Msg("No pre-generated build - calculating")

		dataService := weapon_tree.CreateDataService(db)
		weaponTree, err := weapon_tree.ConstructWeaponTree(itemId, dataService)
		if err != nil {
			return c.String(500, err.Error())
		}
		e := evaluator.CreateEvaluator(dataService)
		result, err := e.EvaluateWeaponEvaluationTask(evaluator.WeaponEvaluationTask{
			Constraints:    constraints,
			WeaponTree:     *weaponTree,
			EvaluationType: buildType,
		})
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, result)
	})

	return e
}
