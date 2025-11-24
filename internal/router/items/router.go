package items_router

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/models"
	"tarkov-build-optimiser/internal/queue"

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
		traderLevels, err := getTraderLevelParams(c)
		if err != nil {
			return c.String(400, err.Error())
		}

		constraints.TraderLevels = traderLevels

		// Check if build already exists and is completed
		build, err := models.GetOptimumBuildByConstraints(db, itemId, buildType, constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get optimum build. item %s, constraints %v", itemId, constraints)
			return c.String(500, err.Error())
		}

		// If build exists and is completed, return it
		if build != nil && build.Status == models.EvaluationCompleted.ToString() {
			return c.JSON(200, build)
		}

		// Check if weapon exists
		weaponExists, err := models.IsWeapon(db, itemId)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to check if weapon exists. item %s", itemId)
			return c.String(500, "Internal server error")
		}

		if !weaponExists {
			return c.String(404, "Weapon not found")
		}

		// Check if already queued or processing
		queueEntry, err := queue.CheckQueueStatus(db, itemId, buildType, constraints)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to check queue status. item %s, constraints %v", itemId, constraints)
			return c.String(500, "Internal server error")
		}

		// If already queued/processing, return 202
		if queueEntry != nil {
			return c.JSON(202, map[string]interface{}{
				"status":   string(queueEntry.Status),
				"message":  "Build queued for calculation",
				"queue_id": queueEntry.QueueID,
			})
		}

		// Queue the build for calculation
		queueID, err := queue.CreateQueueEntry(db, itemId, buildType, constraints, queue.PriorityAPI)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to queue build. item %s, constraints %v", itemId, constraints)
			return c.String(500, "Internal server error")
		}

		log.Info().Msgf("Queued build calculation for item %s, queue_id %d", itemId, queueID)

		return c.JSON(202, map[string]interface{}{
			"status":   string(queue.StatusQueued),
			"message":  "Build queued for calculation",
			"queue_id": queueID,
		})
	})

	return e
}
