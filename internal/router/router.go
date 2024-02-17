package router

import (
	"tarkov-build-optimiser/internal/db"
	itemsrouter "tarkov-build-optimiser/internal/router/items"
	usersrouter "tarkov-build-optimiser/internal/router/users"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Config struct {
	DB *db.Database
}

func NewRouter(config Config) *echo.Echo {
	// tdc := graphql.NewClient("https://api.tarkov.dev/graphql", http.DefaultClient)
	e := echo.New()
	// Set up middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api := e.Group("/api")
	api.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	usersrouter.Bind(api.Group("/users"))
	itemsrouter.Bind(api.Group("/items"), config.DB.Conn)

	return e
}
