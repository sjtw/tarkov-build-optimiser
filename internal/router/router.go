package router

import (
	"net/http"
	"tarkov-build-optimiser/internal/db"
	users_router "tarkov-build-optimiser/internal/router/users"
	"tarkov-build-optimiser/internal/tarkovdev"

	"github.com/Khan/genqlient/graphql"
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

	api.GET("/test", func(c echo.Context) error {
		client := graphql.NewClient("https://api.tarkov.dev/graphql", http.DefaultClient)

		res, err := tarkovdev.GetItems(c.Request().Context(), client)
		if err != nil {
			return c.String(500, err.Error())
		}

		return c.JSON(200, res)
	})

	usersGroup := api.Group("/users")

	users_router.Bind(usersGroup)

	return e
}
