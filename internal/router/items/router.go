package items_router

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/models"

	"github.com/labstack/echo/v4"
)

func Bind(e *echo.Group, db *sql.DB) *echo.Group {
	e.GET("/weapons", func(c echo.Context) error {
		// ctx := c.Request().Context()
		res, err := models.GetWeaponsShort(db)
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	e.GET("/weapons/:item_id/builds", func(c echo.Context) error {
		constraints := models.EvaluationConstraints{
			TraderLevels: []models.TraderLevel{},
		}

		ctx := c.Request().Context()

		itemId := c.Param("item_id")
		buildType := c.QueryParam("build_type")

		for _, traderName := range models.TraderNames {
			paramKey := strings.ToLower(fmt.Sprintf("%s_level", traderName))

			value := c.QueryParam(paramKey)
			if value == "" {
				constraints.TraderLevels = append(constraints.TraderLevels, models.TraderLevel{Name: traderName, Level: 4})
				continue
			}

			level, err := strconv.Atoi(value)
			if err != nil {
				msg := fmt.Sprintf("Invalid level for trader %s", traderName)
				return c.String(400, msg)
			}
			constraints.TraderLevels = append(constraints.TraderLevels, models.TraderLevel{Name: traderName, Level: level})
		}

		res, err := models.GetEvaluatedSubtree(ctx, db, itemId, buildType, constraints)
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	return e
}
