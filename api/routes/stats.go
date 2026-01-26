package routes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
)

const logActionGetMetric = "GetMetric"
const logInfoStats = "STATS"

// GetMetric returns data about the metric received
func GetMetric(c echo.Context) error {
	url := c.Request().URL.String()
	metricType := strings.ToLower(c.Param("metric_type"))
	queryParams := c.QueryParams()

	if result, ok := apiContext.APIConfiguration.Cache.Get(url); ok {
		return c.JSON(http.StatusOK, result)
	}

	result, err := apiContext.APIConfiguration.DBInstance.GetMetricByType(metricType, queryParams)
	if err != nil {
		httpStatus, reply := checkError(err, metricType)
		return c.JSON(httpStatus, reply)
	}

	apiContext.APIConfiguration.Cache.Set(url, result, cache.DefaultExpiration)

	return c.JSON(http.StatusOK, result)
}

func checkError(err error, metricType string) (int, map[string]interface{}) {
	switch err.Error() {
	case "invalid time_range query string param":
		log.Warning(logActionGetMetric, logInfoStats, 111, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid time_range parameter",
			"message": "The 'time_range' query parameter is invalid. Please provide a valid time range format.",
		}
		return http.StatusBadRequest, reply
	case "invalid metric type":
		log.Warning(logActionGetMetric, logInfoStats, 112, metricType, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid metric type",
			"message": fmt.Sprintf("The metric type '%s' is not valid. Please check the available metric types and try again.", metricType),
		}
		return http.StatusBadRequest, reply
	default:
		log.Error(logActionGetMetric, logInfoStats, 2017, metricType, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": "An unexpected error occurred while retrieving metrics. Please try again later.",
		}
		return http.StatusInternalServerError, reply
	}
}
