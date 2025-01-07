package items_router

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/evaluator"
	"tarkov-build-optimiser/internal/models"

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

	//e.GET("/weapons/:item_id/calculate", func(c echo.Context) error {
	//	constraints := models.EvaluationConstraints{
	//		TraderLevels:     []models.TraderLevel{},
	//		IgnoredSlotNames: []string{"Scope", "Ubgl"},
	//	}
	//
	//	itemId := c.Param("item_id")
	//	buildType := c.QueryParam("build_type")
	//	traderLevels, err := getTraderLevelParams(c)
	//	if err != nil {
	//		return c.String(400, err.Error())
	//	}
	//
	//	constraints.TraderLevels = traderLevels
	//	//build, err := models.GetOptimumBuild(db, itemId, buildType, constraints)
	//	//if err != nil {
	//	//	return err
	//	//}
	//
	//	//if build != nil {
	//	//	log.Info().Msg("Returning pre-generated build")
	//	//	return c.JSON(200, build)
	//	//}
	//
	//	log.Info().Msg("No pre-generated build - calculating")
	//
	//	dataService := candidate_tree.CreateDataService(db)
	//	weaponTree, err := candidate_tree.CreateWeaponCandidateTree(itemId, constraints, dataService)
	//	if err != nil {
	//		return c.String(500, err.Error())
	//	}
	//	e := evaluator.CreateEvaluator(dataService)
	//	result, err := e.EvaluateWeaponEvaluationTask(evaluator.WeaponEvaluationTask{
	//		Constraints:    constraints,
	//		CandidateTree:  *weaponTree,
	//		EvaluationType: buildType,
	//	})
	//	if err != nil {
	//		return c.String(500, err.Error())
	//	}
	//
	//	return c.JSON(200, result)
	//})

	e.GET("/items/:item_id/calculate/", func(c echo.Context) error {
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

		prebuild, err := models.GetOptimumBuild(db, itemId, buildType, constraints)
		if err != nil {
			log.Error().Msgf("Failed to get optimum build. item %s, constraints %v", itemId, constraints)
			return c.String(500, err.Error())
		}

		if prebuild != nil {
			return c.JSON(200, prebuild)
		}

		log.Info().Msgf("No pre-generated build - calculating. Item ID: %s, constraints %v", itemId, constraints)

		dataService := candidate_tree.CreateDataService(db)
		candidateTree, err := candidate_tree.CreateItemCandidateTree(itemId, constraints, dataService)
		if err != nil {
			return c.String(500, err.Error())
		}
		build := evaluator.FindBestBuild(candidateTree, buildType, map[string]bool{})
		log.Info().Msg("Build evaluation complete")
		result, err := build.ToEvaluatedWeapon()
		log.Info().Msg("Build converted to evaluated weapon")

		return c.JSON(200, result)
	})

	e.GET("/weapons/:item_id/calculate", func(c echo.Context) error {
		constraints := models.EvaluationConstraints{
			TraderLevels: []models.TraderLevel{},
			//IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical", "Mount", "Pistol Grip", "Ch. Handle", "Foregrip", "Magazine", "Gas Block", "Muzzle", "Handguard"},
			IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical", "Mount"},
			//IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical", "Mount", "Ch. Handle", "Magazine", "Muzzle", "Handguard"},
			IgnoredItemIDs: []string{"5b7be4895acfc400170e2dd5"},
		}

		log.Info().Msgf("Trader Level Constraints: %v", constraints.TraderLevels)

		itemId := c.Param("item_id")
		buildType := c.QueryParam("build_type")
		traderLevels, err := getTraderLevelParams(c)
		if err != nil {
			return c.String(400, err.Error())
		}

		constraints.TraderLevels = traderLevels

		prebuild, err := models.GetOptimumBuild(db, itemId, buildType, constraints)
		if err != nil {
			log.Error().Msgf("Failed to get optimum build. item %s, constraints %v", itemId, constraints)
			return c.String(500, err.Error())
		}

		if prebuild != nil {
			return c.JSON(200, prebuild)
		}

		log.Info().Msg("No pre-generated build - calculating")

		dataService := candidate_tree.CreateDataService(db)
		log.Info().Msg("Constructing weapon tree")
		weaponTree, err := candidate_tree.CreateWeaponCandidateTree(itemId, constraints, dataService)
		if err != nil {
			log.Info().Msg("Weapon tree construction failed")
			return c.String(500, err.Error())
		}
		log.Info().Msg("Weapon tree constructed")

		if buildType == "recoil" {
			log.Info().Msg("Sorting candidate tree for recoil")
			weaponTree.SortAllowedItems("recoil-min")
		} else {
			log.Info().Msg("Sorting candidate tree for ergonomics")
			weaponTree.SortAllowedItems("ergonomics-max")
		}

		excluded := map[string]bool{
			//"5b7be4895acfc400170e2dd5": true,
			//"5a33e75ac4a2826c6e06d759": true,
			//"55d354084bdc2d8c2f8b4568": true,
			//
			////	 high hit rate items
			//"5c791e872e2216001219c40a": true,
			//"558032614bdc2de7118b4585": true,
			//"5c1bc7432e221602b412949d": true,
			//"5c7fc87d2e221644f31c0298": true,
			//"58c157be86f77403c74b2bb6": true,
			//"655dccfdbdcc6b5df71382b6": true,
			//"5f6340d3ca442212f4047eb2": true,
			//"58c157c886f774032749fb06": true,
			//"59f8a37386f7747af3328f06": true,
			//"661e52b5b099f32c28003586": true,
			//"5c1bc5af2e221602b412949b": true,
			//"5c1bc4812e22164bef5cfde7": true,
			//"59fc48e086f77463b1118392": true,
			//"55f84c3c4bdc2d5f408b4576": true,
			//
			//// high frequency rails etc
			//"57d17e212459775a1179a0f5": true,
			//"5c0102b20db834001d23eebc": true,
			//"5b7be4895acfc400170e2dd5": true,
			//"6267c6396b642f77f56f5c1c": true,
			//"5d133067d7ad1a33013f95b4": true,
			//"59e0bed186f774156f04ce84": true,
			//"5a9d6d13a2750c00164f6b03": true,
			//"5b7be46e5acfc400170e2dcf": true,
			//// Magpul M-LOK 2.5 inch rail
			//"5b7be47f5acfc400170e2dd2": true,
			//// AR-15 Daniel Defense RIS II 12.25 lower handguard (Coyote Brown)
			//"638f2003bbd47aeb9e0ff637": true,
			//// AR-15 Daniel Defense RIS II 9.5 lower handguard (Coyote Brown)
			//"638f1ff84822287cad04be9d": true,
			//"5b30bc285acfc47a8608615d": true,
		}
		build := evaluator.FindBestBuild(weaponTree, buildType, excluded)
		log.Info().Msg("Build evaluation complete")
		result, err := build.ToEvaluatedWeapon()
		log.Info().Msg("Build converted to evaluated weapon")

		itemEvaluationResult := result.ToItemEvaluationResult()

		go func() {
			err := models.UpsertOptimumBuild(db, &itemEvaluationResult, constraints)
			if err != nil {
				log.Error().Err(err).Msg("Failed to upsert optimum build")
			}
		}()

		return c.JSON(200, itemEvaluationResult)
	})

	return e
}
