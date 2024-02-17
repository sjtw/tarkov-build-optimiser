package items_router

import (
	"database/sql"
	"tarkov-build-optimiser/internal/models"

	"github.com/labstack/echo/v4"
)

func Bind(e *echo.Group, db *sql.DB) *echo.Group {

	// e.GET("/items", func(c echo.Context) error {
	// 	ctx := c.Request().Context()
	// 	res, err := tarkovdev.GetItems(ctx, tdc)
	// 	if err != nil {
	// 		return c.String(500, err.Error())
	// 	}

	// 	return c.JSON(200, res)
	// })

	e.GET("/weapons", func(c echo.Context) error {
		// ctx := c.Request().Context()
		res, err := models.GetWeapons(db)
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	e.GET("/weapons/:id", func(c echo.Context) error {
		res, err := models.GetWeapon(db, c.Param("id"))
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	// e.GET("/items/mods/:id", func(c echo.Context) error {
	// 	ctx := c.Request().Context()
	// 	res, err := tarkovdev.GetModByID(ctx, tdc, c.Param("id"))
	// 	if err != nil {
	// 		return c.String(500, err.Error())
	// 	}

	// 	return c.JSON(200, res)
	// })

	return e
}
