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

// HandleToken generate an access token for a specific repository or a generic token.
// If repositoryURL is provided, the token will be scoped to that repository.
// If repositoryURL is empty or omitted, a generic token will be created that works with any repository.
func HandleToken(c echo.Context) error {
	repoRequest := types.TokenRequest{}
	if err := c.Bind(&repoRequest); err != nil {
		log.Error("HandleToken", "TOKEN", 1025, err)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request format",
			"message": "The request body must be valid JSON. Provide 'repositoryURL' for a repository-specific token, or omit it for a generic token. Example: {\"repositoryURL\": \"https://github.com/user/repo.git\"} or {}",
		})
	}
	
	tokenType := "repository-specific"
	if repoRequest.RepositoryURL == "" {
		tokenType = "generic"
		log.Info("HandleToken", "TOKEN", 24, "Generating generic token (no repository URL)")
	} else {
		log.Info("HandleToken", "TOKEN", 24, repoRequest.RepositoryURL)
	}
	
	accessToken, err := tokenHandler.GenerateAccessToken(repoRequest)
	if err != nil {
		log.Error("HandleToken ", "TOKEN", 1026, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "token generation failure",
			"message": "Failed to generate access token. Please verify the repository URL and try again.",
		})
	}
	
	var message string
	if repoRequest.RepositoryURL != "" {
		message = fmt.Sprintf("Token generated successfully for repository: %s", repoRequest.RepositoryURL)
	} else {
		message = "Generic token generated successfully. This token can be used with any repository."
	}
	
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success":    true,
		"huskytoken": accessToken,
		"tokenType":  tokenType,
		"message":    message,
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
