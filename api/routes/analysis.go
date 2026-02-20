package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/huskyci-org/huskyCI/api/analysis"
	"github.com/huskyci-org/huskyCI/api/auth"
	apiContext "github.com/huskyci-org/huskyCI/api/context"
	huskydocker "github.com/huskyci-org/huskyCI/api/dockers"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/token"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/util"
	apiUtil "github.com/huskyci-org/huskyCI/api/util/api"
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
	attemptToken := util.GetTokenFromRequest(c)

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

	log.Info(logActionGetAnalysis, logInfoAnalysis, 113, "Analysis data retrieved successfully for RID:", RID)
	return c.JSON(http.StatusOK, analysisResult)
}

// UploadZip handles zip file uploads for local repository analysis
func UploadZip(c echo.Context) error {
	log.Info("UploadZip", logInfoAnalysis, 25, fmt.Sprintf("RID from query: %s", c.QueryParam("rid")))
	RID := c.Response().Header().Get(echo.HeaderXRequestID)
	_ = util.GetTokenFromRequest(c) // Token retrieved for potential future validation

	// Get RID from query parameter or use request ID
	requestedRID := c.QueryParam("rid")
	if requestedRID == "" {
		requestedRID = RID
	}
	log.Info("UploadZip", logInfoAnalysis, 25, fmt.Sprintf("Using RID: %s (query: %s, header: %s)", requestedRID, c.QueryParam("rid"), RID))

	if requestedRID == "" {
		reply := map[string]interface{}{
			"success": false,
			"error":   "missing RID",
			"message": "RID parameter is required. Provide it as query parameter 'rid' or it will be generated from request ID.",
		}
		return c.JSON(http.StatusBadRequest, reply)
	}

	// Validate RID format
	if err := util.CheckMaliciousRID(requestedRID, c); err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1017, requestedRID)
		return err
	}

	// Ensure zip storage directory exists
	if err := util.EnsureZipStorageDir(); err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1019, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": "Failed to initialize zip storage directory.",
		}
		return c.JSON(http.StatusInternalServerError, reply)
	}

	// Get uploaded file
	file, err := c.FormFile("zipfile")
	if err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1020, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid request",
			"message": "No zip file provided. Use multipart/form-data with 'zipfile' field.",
		}
		return c.JSON(http.StatusBadRequest, reply)
	}

	// Validate file extension
	if filepath.Ext(file.Filename) != ".zip" {
		reply := map[string]interface{}{
			"success": false,
			"error":   "invalid file type",
			"message": "File must be a .zip archive.",
		}
		return c.JSON(http.StatusBadRequest, reply)
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1021, err)
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": "Failed to open uploaded file.",
		}
		return c.JSON(http.StatusInternalServerError, reply)
	}
	defer src.Close()

	// Save file
	zipPath := util.GetZipFilePath(requestedRID)
	log.Info("UploadZip", logInfoAnalysis, 25, fmt.Sprintf("Saving zip file to: %s", zipPath))
	dst, err := os.Create(zipPath)
	if err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1022, fmt.Sprintf("Failed to create file at %s: %v", zipPath, err))
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": fmt.Sprintf("Failed to save uploaded file to %s: %v", zipPath, err),
		}
		return c.JSON(http.StatusInternalServerError, reply)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		log.Error("UploadZip", logInfoAnalysis, 1023, fmt.Sprintf("Failed to copy file content: %v", err))
		reply := map[string]interface{}{
			"success": false,
			"error":   "internal server error",
			"message": fmt.Sprintf("Failed to save uploaded file: %v", err),
		}
		return c.JSON(http.StatusInternalServerError, reply)
	}

	log.Info("UploadZip", logInfoAnalysis, 26, fmt.Sprintf("RID: %s, Filename: %s, Path: %s", requestedRID, file.Filename, zipPath))

	reply := map[string]interface{}{
		"success": true,
		"error":   "",
		"message": fmt.Sprintf("Zip file uploaded successfully for RID: %s", requestedRID),
		"rid":     requestedRID,
	}
	return c.JSON(http.StatusCreated, reply)
}

// ReceiveRequest receives the request and performs several checks before starting a new analysis.
func ReceiveRequest(c echo.Context) error {

	RID := c.Response().Header().Get(echo.HeaderXRequestID)
	attemptToken := util.GetTokenFromRequest(c)

	// step-00: is this a valid JSON?
	// Read raw body first to handle EnryOutput binding
	bodyBytes, _ := io.ReadAll(c.Request().Body)
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	
	repository := types.Repository{}
	err := json.Unmarshal(bodyBytes, &repository)
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

	// step-01a: If this is a file:// URL, verify the zip file exists
	if util.IsFileURL(repository.URL) {
		log.Info(logActionReceiveRequest, logInfoAnalysis, 26, fmt.Sprintf("Processing file:// URL: %s", repository.URL))
		extractedRID := util.ExtractRIDFromFileURL(repository.URL)
		if extractedRID == "" {
			reply := map[string]interface{}{
				"success": false,
				"error":   "invalid file URL",
				"message": "Invalid file:// URL format. Expected format: file://<RID>",
			}
			return c.JSON(http.StatusBadRequest, reply)
		}
		zipPath := util.GetZipFilePath(extractedRID)
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			reply := map[string]interface{}{
				"success": false,
				"error":   "zip file not found",
				"message": fmt.Sprintf("Zip file for RID '%s' not found. Please upload the zip file first using POST /analysis/upload", extractedRID),
			}
			return c.JSON(http.StatusBadRequest, reply)
		}
		// Extract the zip file if not already extracted in API container
		extractedDir := util.GetExtractedDir(extractedRID)
		if _, err := os.Stat(extractedDir); os.IsNotExist(err) {
			// Extract in API container first (for API's own use)
			if err := util.ExtractZip(zipPath, extractedDir); err != nil {
				log.Error(logActionReceiveRequest, logInfoAnalysis, 1018, err)
				reply := map[string]interface{}{
					"success": false,
					"error":   "failed to extract zip file",
					"message": fmt.Sprintf("Failed to extract zip file: %v", err),
				}
				return c.JSON(http.StatusInternalServerError, reply)
			}
		}
		
		// Always extract in dockerapi to ensure dockerapi's Docker daemon can see the files
		// This is necessary because docker-in-docker doesn't properly share bind mounts
		// Even if files exist in API container, dockerapi can't see them
		if os.Getenv("HUSKYCI_INFRASTRUCTURE_USE") == "docker" {
			log.Info(logActionReceiveRequest, logInfoAnalysis, 26, fmt.Sprintf("Attempting to extract zip in dockerapi for RID: %s", extractedRID))
			dockerAPIHost, err := apiContext.APIConfiguration.DBInstance.FindAndModifyDockerAPIAddresses()
			if err != nil {
				log.Error(logActionReceiveRequest, logInfoAnalysis, 1018, fmt.Errorf("failed to get dockerapi host (non-fatal): %v", err))
			} else {
				apiHost, err := apiUtil.FormatDockerHostAddress(dockerAPIHost, apiContext.APIConfiguration)
				if err != nil {
					log.Error(logActionReceiveRequest, logInfoAnalysis, 1018, fmt.Errorf("failed to format dockerapi host (non-fatal): %v", err))
				} else {
					log.Info(logActionReceiveRequest, logInfoAnalysis, 26, fmt.Sprintf("Extracting zip in dockerapi: zipPath=%s, destDir=%s", zipPath, extractedDir))
					// Extract files in dockerapi using a temporary container
					// This ensures dockerapi can see the files even if they already exist in API container
					if err := huskydocker.ExtractZipInDockerAPI(apiHost, zipPath, extractedDir); err != nil {
						// Log but don't fail - extraction in API container may have succeeded
						log.Error(logActionReceiveRequest, logInfoAnalysis, 1018, fmt.Errorf("failed to extract zip in dockerapi (non-fatal): %v", err))
					} else {
						log.Info(logActionReceiveRequest, logInfoAnalysis, 26, fmt.Sprintf("Successfully extracted zip in dockerapi for RID: %s", extractedRID))
					}
				}
			}
		}
	}

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
	// Debug: Log EnryOutput if present
	if util.IsFileURL(repository.URL) {
		enryOutputMsg := fmt.Sprintf("Received file:// URL: %s, EnryOutput present: %v, length: %d", repository.URL, repository.EnryOutput != "", len(repository.EnryOutput))
		log.Info(logActionReceiveRequest, logInfoAnalysis, 16, enryOutputMsg)
		if repository.EnryOutput != "" {
			preview := repository.EnryOutput
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			log.Info(logActionReceiveRequest, logInfoAnalysis, 16, fmt.Sprintf("EnryOutput preview: %s", preview))
		}
	}
	go analysis.StartAnalysis(RID, repository)
	reply := map[string]interface{}{
		"success": true,
		"error":   "",
		"message": fmt.Sprintf("Analysis started successfully for repository '%s' on branch '%s'", repository.URL, repository.Branch),
		"rid":     RID,
	}
	return c.JSON(http.StatusCreated, reply)
}
