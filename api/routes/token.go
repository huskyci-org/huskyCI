package routes

import (
	"fmt"
	"net/http"

	"github.com/huskyci-org/huskyCI/api/auth"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/token"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/labstack/echo/v4"
)

var (
	tokenHandler token.THandler
)

func init() {
	tokenCaller := token.TCaller{}
	hashGen := auth.Pbkdf2Caller{}
	tokenHandler = token.THandler{
		External: &tokenCaller,
		HashGen:  &hashGen,
	}
}

// HandleToken generate an access token for a specific repository
func HandleToken(c echo.Context) error {
	repoRequest := types.TokenRequest{}
	if err := c.Bind(&repoRequest); err != nil {
		log.Error("HandleToken", "TOKEN", 1025, err)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request format",
			"message": "The request body must be valid JSON with a 'repositoryURL' field. Example: {\"repositoryURL\": \"https://github.com/user/repo.git\"}",
		})
	}
	log.Info("HandleToken", "TOKEN", 24, repoRequest.RepositoryURL)
	accessToken, err := tokenHandler.GenerateAccessToken(repoRequest)
	if err != nil {
		log.Error("HandleToken ", "TOKEN", 1026, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "token generation failure",
			"message": "Failed to generate access token. Please verify the repository URL and try again.",
		})
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success":    true,
		"huskytoken": accessToken,
		"message":    fmt.Sprintf("Token generated successfully for repository: %s", repoRequest.RepositoryURL),
	})
}

// HandleDeactivation will deactivate an access token passed in the body
// of the request
func HandleDeactivation(c echo.Context) error {
	tokenRequest := types.AccessToken{}
	if err := c.Bind(&tokenRequest); err != nil {
		log.Error("HandleInvalidate", "TOKEN", 1025, err)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request format",
			"message": "The request body must be valid JSON with a 'huskytoken' field. Example: {\"huskytoken\": \"your-token-here\"}",
		})
	}
	if err := tokenHandler.InvalidateToken(tokenRequest.HuskyToken); err != nil {
		log.Error("HandleInvalidate ", "TOKEN", 1028, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "token deactivation failure",
			"message": "Failed to deactivate the token. Please verify the token and try again.",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"error":   "",
		"message": "Token deactivated successfully",
	})
}
