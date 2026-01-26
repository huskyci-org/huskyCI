package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthCheck is the heath check function.
func HealthCheck(c echo.Context) error {
	return c.String(http.StatusOK, "WORKING\n")
}
