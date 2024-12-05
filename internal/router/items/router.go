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

	"github.com/labstack/echo/v4"
)

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
			TraderLevels: []models.TraderLevel{},
		}

		itemId := c.Param("item_id")
		buildType := c.QueryParam("build_type")
		traderLevels, err := getTraderLevelParams(c)
		if err != nil {
			return c.String(400, err.Error())
		}

		constraints.TraderLevels = traderLevels

		build, err := models.GetOptimumBuild(db, itemId, buildType, constraints)
		if err != nil {
			return err
		}

		if build != nil {
			log.Info().Msg("Returning pre-generated build")
			return c.JSON(200, build)
		}

		log.Info().Msg("No pre-generated build - calculating")

		dataService := evaluator.CreateDataService(db)
		weapon, err := evaluator.ConstructWeaponTree(itemId, dataService)
		e := evaluator.CreateEvaluator(dataService)
		result, err := e.EvaluateTask(evaluator.Task{
			Constraints:    constraints,
			Weapon:         *weapon,
			EvaluationType: buildType,
		})
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, result)
	})

	return e
}
