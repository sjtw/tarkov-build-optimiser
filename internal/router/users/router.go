package users_router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func Bind(e *echo.Group) *echo.Group {
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.GET("/:id", func(c echo.Context) error {
		id := c.Param("id")

		return c.String(http.StatusOK, fmt.Sprintf("user id: %s", id))
	})

	return e
}
