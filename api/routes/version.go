package routes

import (
	"net/http"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/labstack/echo"
)

// GetAPIVersion returns the API version
func GetAPIVersion(c echo.Context) error {
	configAPI := apiContext.APIConfiguration
	return c.JSON(http.StatusOK, GetRequestResult(configAPI))
}

// GetRequestResult returns a map containing API's version and release date
func GetRequestResult(configAPI *apiContext.APIConfig) map[string]string {
	requestResult := map[string]string{
		"version": configAPI.Version,
		"date":    configAPI.ReleaseDate,
	}
	return requestResult
}
