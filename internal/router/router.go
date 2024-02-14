package router

import (
	"tarkov-build-optimiser/internal/db"
	users_router "tarkov-build-optimiser/internal/router/users"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type RouterConfig struct {
	DB *db.Database
}

func NewRouter(config RouterConfig) *echo.Echo {
	e := echo.New()

	// Set up middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api := e.Group("/api")
	api.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})
	usersGroup := api.Group("/users")

	users_router.Bind(usersGroup)

	return e
}
