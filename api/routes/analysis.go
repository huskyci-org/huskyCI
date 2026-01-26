package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/huskyci-org/huskyCI/api/analysis"
	"github.com/huskyci-org/huskyCI/api/auth"
	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/token"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/util"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	tokenValidator token.TValidator
)

func init() {
	tokenCaller := token.TCaller{}
	hashGen := auth.Pbkdf2Caller{}
	tokenHandler := token.THandler{
		External: &tokenCaller,
		HashGen:  &hashGen,
	}
	tokenValidator = token.TValidator{
		TokenVerifier: &tokenHandler,
	}
}

const logActionReceiveRequest = "ReceiveRequest"
const logActionGetAnalysis = "GetAnalysis"
const logInfoAnalysis = "ANALYSIS"

// GetAnalysis returns the status of a given analysis given a RID.
func GetAnalysis(c echo.Context) error {

	RID := c.Param("id")
	attemptToken := c.Request().Header.Get("Husky-Token")

	if err := util.CheckMaliciousRID(RID, c); err != nil {
		log.Error(logActionGetAnalysis, logInfoAnalysis, 1017, RID)
		return err
	}

	analysisQuery := map[string]interface{}{"RID": RID}
	log.Info(logActionGetAnalysis, logInfoAnalysis, 114, RID)
	analysisResult, err := apiContext.APIConfiguration.DBInstance.FindOneDBAnalysis(analysisQuery)

	if err != nil {
		if err == mongo.ErrNoDocuments || err.Error() == "No data found" {
			log.Warning(logActionGetAnalysis, logInfoAnalysis, 106, RID)
			reply := map[string]interface{}{
				"success": false,
				"error":   "analysis not found",
				"message": fmt.Sprintf("No analysis found with RID: %s. Please verify the RID and try again.", RID),
				"rid":     RID,
			}
			return c.JSON(http.StatusNotFound, reply)
		}
		log.Error(logActionGetAnalysis, logInfoAnalysis, 1020, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": "An unexpected error occurred while retrieving the analysis. Please try again later or contact support if the issue persists.",
		}
		return c.JSON(http.StatusInternalServerError, reply)
	}

	if !tokenValidator.HasAuthorization(attemptToken, analysisResult.URL) {
		log.Error(logActionGetAnalysis, logInfoAnalysis, 1027, RID)
		reply := map[string]interface{}{
			"success": false,
			"error":   "permission denied",
			"message": "The provided token does not have permission to access this analysis. Please verify your token has access to the repository.",
		}
		return c.JSON(http.StatusUnauthorized, reply)
	}

	// Log the successful retrieval of analysis data
	log.Info(logActionGetAnalysis, logInfoAnalysis, 113, "Analysis data retrieved successfully for RID:", RID)

	// Add logging to capture the analysis result being returned
	log.Info(logActionGetAnalysis, logInfoAnalysis, 113, "Analysis result:", analysisResult)

	return c.JSON(http.StatusOK, analysisResult)
}

// ReceiveRequest receives the request and performs several checks before starting a new analysis.
func ReceiveRequest(c echo.Context) error {

	RID := c.Response().Header().Get(echo.HeaderXRequestID)
	attemptToken := c.Request().Header.Get("Husky-Token")

	// step-00: is this a valid JSON?
	repository := types.Repository{}
	err := c.Bind(&repository)
	if err != nil {
		log.Error(logActionReceiveRequest, logInfoAnalysis, 1015, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid request format",
			"message": "The request body must be valid JSON with 'repositoryURL' and 'repositoryBranch' fields. Example: {\"repositoryURL\": \"https://github.com/user/repo.git\", \"repositoryBranch\": \"main\"}",
		}
		return c.JSON(http.StatusBadRequest, reply)
	}
	if !tokenValidator.HasAuthorization(attemptToken, repository.URL) {
		log.Error("ReceivedRequest", logInfoAnalysis, 1027, RID)
		reply := map[string]interface{}{
			"success": false,
			"error":   "permission denied",
			"message": fmt.Sprintf("The provided token does not have permission to analyze repository: %s. Please verify your token has access to this repository.", repository.URL),
		}
		return c.JSON(http.StatusUnauthorized, reply)
	}
	// step-01: Check malicious inputs
	sanitizedRepoURL, err := util.CheckValidInput(repository, c)
	if err != nil {
		return err
	}
	repository.URL = sanitizedRepoURL

	// step-02: is this repository already in MongoDB?
	repositoryQuery := map[string]interface{}{"repositoryURL": repository.URL}
	_, err = apiContext.APIConfiguration.DBInstance.FindOneDBRepository(repositoryQuery)
	if err != nil {
		if err == mongo.ErrNoDocuments || err.Error() == "No data found" {
			// step-02-o1: repository not found! insert it into MongoDB
			repository.CreatedAt = time.Now()
			err = apiContext.APIConfiguration.DBInstance.InsertDBRepository(repository)
			if err != nil {
				log.Error(logActionReceiveRequest, logInfoAnalysis, 1010, err)
				reply := map[string]interface{}{
					"success": false,
					"error":   "internal server error",
					"message": "Failed to register the repository. Please try again later or contact support if the issue persists.",
				}
				return c.JSON(http.StatusInternalServerError, reply)
			}
		} else {
			// step-02-o2: another error searching for repositoryQuery
			log.Error(logActionReceiveRequest, logInfoAnalysis, 1013, err)
			reply := map[string]interface{}{
				"success": false,
				"error":   "internal server error",
				"message": "An unexpected error occurred while processing your request. Please try again later.",
			}
			return c.JSON(http.StatusInternalServerError, reply)
		}
	} else { // err == nil
		// step-03: repository found! does it have a running status analysis?
		analysisQuery := map[string]interface{}{"repositoryURL": repository.URL, "repositoryBranch": repository.Branch}
		analysisResult, err := apiContext.APIConfiguration.DBInstance.FindOneDBAnalysis(analysisQuery)
		if err != nil {
			if err == mongo.ErrNoDocuments || err.Error() == "No data found" {
				// nice! we can start this analysis!
			} else {
				// step-03-err: another error searching for analysisQuery
				log.Error(logActionReceiveRequest, logInfoAnalysis, 1009, err)
				reply := map[string]interface{}{
					"success": false,
					"error":   "internal server error",
					"message": "An unexpected error occurred while checking for existing analyses. Please try again later.",
				}
				return c.JSON(http.StatusInternalServerError, reply)
			}
		} else { // err == nil
			// step 03-a: Ops, this analysis is already running!
			if analysisResult.Status == "running" {
				log.Warning(logActionReceiveRequest, logInfoAnalysis, 104, analysisResult.URL)
				reply := map[string]interface{}{
					"success": false,
					"error":   "analysis already running",
					"message": fmt.Sprintf("An analysis for repository '%s' on branch '%s' is already in progress. Please wait for it to complete or use the existing analysis RID: %s", repository.URL, repository.Branch, analysisResult.RID),
					"rid":     analysisResult.RID,
				}
				return c.JSON(http.StatusConflict, reply)
			}
		}
	}

	// step 04: lets start this analysis!
	log.Info(logActionReceiveRequest, logInfoAnalysis, 16, repository.Branch, repository.URL)
	go analysis.StartAnalysis(RID, repository)
	reply := map[string]interface{}{
		"success": true,
		"error":   "",
		"message": fmt.Sprintf("Analysis started successfully for repository '%s' on branch '%s'", repository.URL, repository.Branch),
		"rid":     RID,
	}
	return c.JSON(http.StatusCreated, reply)
}
