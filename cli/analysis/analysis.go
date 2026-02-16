package analysis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/huskyci-org/huskyCI/cli/config"
	"github.com/huskyci-org/huskyCI/cli/types"
	"github.com/huskyci-org/huskyCI/cli/util"
	"github.com/huskyci-org/huskyCI/cli/vulnerability"
	"github.com/src-d/enry/v2"
)

// verboseMode stores whether verbose mode is enabled
var verboseMode bool

// SetVerbose sets the verbose mode flag
func SetVerbose(v bool) {
	verboseMode = v
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verboseMode
}

// Analysis is the struct that stores all data from analysis performed.
type Analysis struct {
	ID              string                        `bson:"ID" json:"ID"`
	RID             string                        `bson:"RID" json:"RID"` // Request ID from API
	CompressedFile  CompressedFile                `bson:"compressedFile" json:"compressedFile"`
	Errors          []string                      `bson:"errorsFound,omitempty" json:"errorsFound"`
	Languages       []string                      `bson:"languages" json:"languages"`
	Path            string                        `json:"-"` // Path being analyzed (for Enry generation)
	StartedAt       time.Time                     `bson:"startedAt" json:"startedAt"`
	FinishedAt      time.Time                     `bson:"finishedAt" json:"finishedAt"`
	Vulnerabilities []vulnerability.Vulnerability `bson:"vulnerabilities" json:"vulnerabilities"`
	Result          Result                        `bson:"result,omitempty" json:"result"`
	APITarget       *types.Target                 `json:"-"` // API target configuration
}

// CompressedFile holds the info from the compressed file
type CompressedFile struct {
	Name string `bson:"name" json:"name"`
	Size string `bson:"size" json:"size"`
}

// Result holds the status and the info of an analysis.
type Result struct {
	Status string `bson:"status" json:"status"`
	Info   string `bson:"info,omitempty" json:"info"`
}

// New returns a new analysis struct
func New() *Analysis {
	return &Analysis{
		ID: uuid.New().String(),
	}
}

// CheckPath checks the given path to check which languages were found and do some others security checks
func (a *Analysis) CheckPath(path string) error {

	fullPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error resolving path '%s': %w", path, err)
	}

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Resolved path: %s\n", fullPath)
	}

	// Check if path exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s\n\nTip: Make sure the path is correct and try again", fullPath)
	}

	fmt.Printf("🔍 Scanning code from: %s\n", fullPath)

	// Store path for later use (e.g., Enry output generation)
	a.Path = fullPath

	if err := a.setLanguages(fullPath); err != nil {
		if err.Error() == "no languages found" {
			return fmt.Errorf("no supported programming languages found in '%s'\n\nTip: Make sure the directory contains code files in supported languages (Python, Ruby, JavaScript, Go, Java, C#, HCL)", fullPath)
		}
		return fmt.Errorf("error detecting languages: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Detected %d languages: %v\n", len(a.Languages), a.Languages)
	}

	fmt.Println("\n📋 Detected languages:")
	securityTests := a.getAvailableSecurityTests(a.Languages)
	for language := range securityTests {
		fmt.Printf("  ✓ %s\n", language)
		if IsVerbose() {
			fmt.Printf("    [VERBOSE] Security tests: %v\n", securityTests[language])
		}
	}

	return nil
}

// CompressFiles will compress all files from a given path into a single file named GUID
func (a *Analysis) CompressFiles(path string) error {

	fmt.Println("\n📦 Compressing code...")

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Compressing files from path: %s\n", path)
	}

	if err := a.HouseCleaning(); err != nil {
		// it's ok. maybe the file is not there yet.
		if IsVerbose() {
			fmt.Printf("[VERBOSE] Could not clean previous zip file (this is OK if it doesn't exist): %v\n", err)
		}
	}

	allFilesAndDirNames, err := util.GetAllAllowedFilesAndDirsFromPath(path)
	if err != nil {
		return fmt.Errorf("error reading files from path: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Found %d files/directories to compress\n", len(allFilesAndDirNames))
	}

	zipFilePath, err := util.CompressFiles(allFilesAndDirNames)
	if err != nil {
		return fmt.Errorf("error compressing files: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Zip file created at: %s\n", zipFilePath)
	}

	if err := a.setZipSize(zipFilePath); err != nil {
		return fmt.Errorf("error calculating archive size: %w", err)
	}

	fmt.Printf("✓ Compressed successfully! Size: %s\n", a.CompressedFile.Size)

	return nil
}

// SendZip will send the zip file to the huskyCI API to start the analysis
func (a *Analysis) SendZip() error {
	fmt.Println("\n🚀 Sending code to huskyCI API...")

	// Get API target configuration
	target, err := config.GetCurrentTarget()
	if err != nil {
		return fmt.Errorf("failed to get API target configuration: %w\n\nTip: Configure a target using 'huskyci target-add <name> <endpoint>'", err)
	}

	if target.Token == "" {
		return fmt.Errorf("authentication token not found\n\nTip: Set HUSKYCI_CLI_TOKEN environment variable or configure token storage")
	}

	a.APITarget = target

	if IsVerbose() {
		zipFilePath, err := config.GetHuskyZipFilePath()
		if err == nil {
			fmt.Printf("[VERBOSE] Zip file path: %s\n", zipFilePath)
		}
		fmt.Printf("[VERBOSE] Analysis ID: %s\n", a.ID)
		fmt.Printf("[VERBOSE] API endpoint: %s\n", target.Endpoint)
	}

	// Create HTTP client
	useTLS := util.IsHTTPS(target.Endpoint)
	httpClient, err := util.NewHTTPClient(useTLS)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// For local file analysis, upload the zip file first
	zipFilePath, err := config.GetHuskyZipFilePath()
	if err != nil {
		return fmt.Errorf("failed to get zip file path: %w", err)
	}

	// Upload zip file for local analysis
	fmt.Println("📤 Uploading zip file...")
	if IsVerbose() {
		fmt.Printf("[VERBOSE] Preparing to upload zip file: %s\n", zipFilePath)
		fmt.Printf("[VERBOSE] Analysis ID (RID): %s\n", a.ID)
	}
	normalizedEndpoint := util.NormalizeURL(target.Endpoint)
	uploadURL := fmt.Sprintf("%s/analysis/upload?rid=%s", normalizedEndpoint, a.ID)
	
	if IsVerbose() {
		fmt.Printf("[VERBOSE] Upload URL: %s\n", uploadURL)
	}
	
	zipFile, err := os.Open(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipFile.Close()
	
	if IsVerbose() {
		fileInfo, _ := zipFile.Stat()
		fmt.Printf("[VERBOSE] Zip file opened successfully, size: %d bytes\n", fileInfo.Size())
	}

	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, err := writer.CreateFormFile("zipfile", filepath.Base(zipFilePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := io.Copy(part, zipFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	writer.Close()

	uploadReq, err := http.NewRequest("POST", uploadURL, &uploadBody)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Add("Husky-Token", target.Token)
	uploadReq.Header.Add("User-Agent", "huskyci-cli")

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Sending upload request to: %s\n", uploadURL)
		fmt.Printf("[VERBOSE] Content-Type: %s\n", writer.FormDataContentType())
		fmt.Printf("[VERBOSE] Upload body size: %d bytes\n", uploadBody.Len())
	}

	uploadResp, err := httpClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload zip file: %w\n\nTip: Check your network connection and verify the API endpoint is accessible", err)
	}
	defer uploadResp.Body.Close()

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Upload response status: %d\n", uploadResp.StatusCode)
	}

	uploadBodyBytes, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode != http.StatusCreated {
		if IsVerbose() {
			fmt.Printf("[VERBOSE] Upload failed, response: %s\n", string(uploadBodyBytes))
		}
		return fmt.Errorf("failed to upload zip file\n\nStatus Code: %d\nResponse: %s\n\nTip: Verify the API supports zip file uploads", uploadResp.StatusCode, string(uploadBodyBytes))
	}

	// Verify the upload response contains the correct RID
	var uploadRespData map[string]interface{}
	if err := json.Unmarshal(uploadBodyBytes, &uploadRespData); err == nil {
		if respRID, ok := uploadRespData["rid"].(string); ok && respRID != "" {
			if respRID != a.ID {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Warning: Upload response RID (%s) differs from expected RID (%s)\n", respRID, a.ID)
				}
				// Use the RID from the response if different
				a.ID = respRID
			}
			if IsVerbose() {
				fmt.Printf("[VERBOSE] Upload confirmed with RID: %s\n", respRID)
			}
		}
	}

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Zip file uploaded successfully with RID: %s\n", a.ID)
		fmt.Printf("[VERBOSE] Upload response: %s\n", string(uploadBodyBytes))
	}
	fmt.Println("✓ Zip file uploaded successfully!")

	// Generate Enry output locally for file:// URLs
	// This avoids docker-in-docker issues where Enry can't see extracted files
	var enryOutput string
	if a.Path != "" {
		if IsVerbose() {
			fmt.Printf("[VERBOSE] Generating Enry output locally from path: %s\n", a.Path)
		}
		enryOutput, err = a.generateEnryOutput(a.Path)
		if err != nil {
			if IsVerbose() {
				fmt.Printf("[VERBOSE] Warning: Failed to generate Enry output locally: %v (API will run Enry instead)\n", err)
			}
			// Continue without Enry output - API will run Enry
			enryOutput = ""
		} else {
			if IsVerbose() {
				fmt.Printf("[VERBOSE] Generated Enry output: %s\n", enryOutput)
			}
		}
	}
	
	// Prepare request payload for analysis
	requestPayload := types.JSONPayload{
		RepositoryURL:      fmt.Sprintf("file://%s", a.ID), // Using analysis ID as identifier
		RepositoryBranch:   "local",
		LanguageExclusions: make(map[string]bool),
		EnryOutput:         enryOutput, // Send Enry output to API
	}

	marshalPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create POST request to start analysis
	apiURL := fmt.Sprintf("%s/analysis", normalizedEndpoint)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(marshalPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Husky-Token", target.Token)
	req.Header.Add("User-Agent", "huskyci-cli")

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Sending POST request to: %s\n", apiURL)
		fmt.Printf("[VERBOSE] Request payload: %s\n", string(marshalPayload))
	}

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to API: %w\n\nTip: Check your network connection and verify the API endpoint is accessible", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("authentication failed: The provided token is invalid or expired\n\nTip: Generate a new token using the huskyCI API")
		}
		if resp.StatusCode == http.StatusBadRequest {
			// Try to extract error message from response
			var errorResp map[string]interface{}
			errorMsg := "Invalid request parameters"
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if msg, ok := errorResp["message"].(string); ok && msg != "" {
					errorMsg = msg
				} else if errStr, ok := errorResp["error"].(string); ok && errStr != "" {
					errorMsg = errStr
				}
			}
			
			// Check if this is a file:// URL issue (zip file not found)
			if strings.Contains(requestPayload.RepositoryURL, "file://") {
				if strings.Contains(string(body), "zip file not found") || strings.Contains(errorMsg, "zip file not found") {
					return fmt.Errorf("zip file not found on server\n\nRID used: %s\nStatus: %d\nResponse: %s\n\nPossible causes:\n  1. The zip file upload may have failed silently\n  2. The API server may not have write permissions to /tmp/huskyci-zips\n  3. There may be a mismatch between the upload RID and analysis RID\n\nTroubleshooting:\n  - Run with --verbose flag to see detailed logs\n  - Check API server logs for upload errors\n  - Verify the API server has write access to /tmp/huskyci-zips directory\n  - Try uploading again: huskyci run %s", a.ID, resp.StatusCode, string(body), a.ID)
				}
				return fmt.Errorf("local file analysis error\n\nRID: %s\nStatus: %d\nResponse: %s\n\nTip: The zip file was uploaded but the analysis request failed. Check the API logs for more details.", a.ID, resp.StatusCode, string(body))
			}
			
			return fmt.Errorf("bad request: %s\n\nStatus: %d\nResponse: %s\n\nTip: Verify that the repository URL is a valid git repository URL (e.g., https://github.com/user/repo.git)", errorMsg, resp.StatusCode, string(body))
		}
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("conflict: An analysis is already running\n\nStatus: %d\nResponse: %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("failed to start analysis: Unexpected response from API\n\nStatus Code: %d\nResponse: %s\n\nTip: Check the huskyCI API status and try again", resp.StatusCode, string(body))
	}

	// Extract RID from response header
	RID := resp.Header.Get("X-Request-Id")
	if RID == "" {
		// Try to parse from response body
		var responseBody map[string]interface{}
		if err := json.Unmarshal(body, &responseBody); err == nil {
			if rid, ok := responseBody["rid"].(string); ok {
				RID = rid
			}
		}
	}

	if RID == "" {
		return fmt.Errorf("failed to start analysis: No request ID (RID) received from the API\n\nTip: This may indicate an issue with the huskyCI API. Please check the API status")
	}

	a.RID = RID
	a.Result.Status = "running"
	a.StartedAt = time.Now()

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Analysis started successfully with RID: %s\n", RID)
	}

	fmt.Println("✓ Code sent successfully!")
	return nil
}

// CheckStatus is a worker to check the huskyCI API for the status of the particular analysis
func (a *Analysis) CheckStatus() error {
	if a.RID == "" {
		return fmt.Errorf("no RID available - analysis was not started successfully")
	}

	if a.APITarget == nil {
		target, err := config.GetCurrentTarget()
		if err != nil {
			return fmt.Errorf("failed to get API target configuration: %w", err)
		}
		a.APITarget = target
	}

	fmt.Println("\n⏳ Checking analysis status...")

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Analysis RID: %s\n", a.RID)
		fmt.Printf("[VERBOSE] API endpoint: %s\n", a.APITarget.Endpoint)
	}

	// Create HTTP client
	useTLS := util.IsHTTPS(a.APITarget.Endpoint)
	httpClient, err := util.NewHTTPClient(useTLS)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Poll API for analysis status
	timeout := time.After(60 * time.Minute)
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	checkCount := 0

	for {
		select {
		case <-timeout:
			return fmt.Errorf("analysis timed out after 60 minutes\n\nTip: Large codebases may take longer to analyze. Try again or contact support if this persists")
		case <-ticker.C:
			checkCount++

			// Create GET request
			normalizedEndpoint := util.NormalizeURL(a.APITarget.Endpoint)
			apiURL := fmt.Sprintf("%s/analysis/%s", normalizedEndpoint, a.RID)
			req, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create HTTP request: %w", err)
			}

			req.Header.Add("Husky-Token", a.APITarget.Token)
			req.Header.Add("User-Agent", "huskyci-cli")

			if IsVerbose() && checkCount%12 == 0 { // Log every minute (12 * 5 seconds)
				fmt.Printf("[VERBOSE] Checking analysis status (attempt #%d)...\n", checkCount)
			}

			// Send request
			resp, err := httpClient.Do(req)
			if err != nil {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Network error (will retry): %v\n", err)
				}
				continue // Retry on network errors
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				if checkCount < 3 {
					// Analysis might not be created yet, wait a bit
					continue
				}
				return fmt.Errorf("analysis not found: No analysis found with RID '%s'\n\nTip: Verify the RID is correct and the analysis exists", a.RID)
			}

			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("authentication failed: Invalid or expired token\n\nTip: Generate a new token using the huskyCI API")
			}

			if resp.StatusCode != http.StatusOK {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Unexpected status code %d, will retry\n", resp.StatusCode)
				}
				continue // Retry on other errors
			}

			// Parse response
			var apiAnalysis types.Analysis
			if err := json.Unmarshal(body, &apiAnalysis); err != nil {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Failed to parse response (will retry): %v\n", err)
				}
				continue
			}

			// Update analysis status
			a.Result.Status = apiAnalysis.Status
			if apiAnalysis.ErrorFound != "" {
				a.Errors = append(a.Errors, apiAnalysis.ErrorFound)
			}

			if !apiAnalysis.StartedAt.IsZero() {
				a.StartedAt = apiAnalysis.StartedAt
			}
			if !apiAnalysis.FinishedAt.IsZero() {
				a.FinishedAt = apiAnalysis.FinishedAt
			}

			// Convert API vulnerabilities to CLI format
			if err := a.convertAPIVulnerabilities(apiAnalysis); err != nil {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Warning: Failed to convert vulnerabilities: %v\n", err)
				}
			}

			if IsVerbose() {
				fmt.Printf("[VERBOSE] Current status: %s (check #%d)\n", a.Result.Status, checkCount)
			}

			// Check if analysis is complete
			if apiAnalysis.Status == "finished" {
				if IsVerbose() {
					fmt.Printf("[VERBOSE] Analysis completed after %d checks\n", checkCount)
				}
				fmt.Println("✓ Analysis check completed!")
				return nil
			}

			if apiAnalysis.Status == "error running" {
				errorMsg := apiAnalysis.ErrorFound
				if errorMsg == "" {
					errorMsg = "Unknown error occurred during analysis"
				}
				return fmt.Errorf("analysis failed: %s\n\nTip: Check the analysis details for more information", errorMsg)
			}

			// Status is "running" or other, continue polling
		}
	}
}

// PrintVulns prints all vulnerabilities found after the analysis has been finished
func (a *Analysis) PrintVulns() {
	fmt.Println("\n📊 Analysis Results:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if IsVerbose() {
		fmt.Printf("[VERBOSE] Analysis ID: %s\n", a.ID)
		fmt.Printf("[VERBOSE] Status: %s\n", a.Result.Status)
		if a.Result.Info != "" {
			fmt.Printf("[VERBOSE] Info: %s\n", a.Result.Info)
		}
		fmt.Printf("[VERBOSE] Vulnerabilities count: %d\n", len(a.Vulnerabilities))
		if len(a.Errors) > 0 {
			fmt.Printf("[VERBOSE] Errors: %v\n", a.Errors)
		}
	}

	// Check if we have any vulnerabilities to display
	if len(a.Vulnerabilities) == 0 {
		if a.Result.Status == "" {
			fmt.Println("\n⚠️  No analysis results available.")
			fmt.Println("   This may indicate that:")
			fmt.Println("   • The analysis has not completed yet")
			fmt.Println("   • The API integration is not fully implemented")
			if IsVerbose() {
				fmt.Println("   • Use --verbose flag for more debugging information")
			}
		} else if a.Result.Status == "finished" {
			fmt.Println("\n✅ No vulnerabilities found!")
			fmt.Println("   Your code appears to be secure.")
		} else {
			fmt.Printf("\n📋 Analysis Status: %s\n", a.Result.Status)
			if a.Result.Info != "" {
				fmt.Printf("   Info: %s\n", a.Result.Info)
			}
		}
		return
	}

	// Group vulnerabilities by severity
	highVulns := []vulnerability.Vulnerability{}
	mediumVulns := []vulnerability.Vulnerability{}
	lowVulns := []vulnerability.Vulnerability{}
	infoVulns := []vulnerability.Vulnerability{}

	for _, vuln := range a.Vulnerabilities {
		switch vuln.Severity {
		case "HIGH", "high", "High":
			highVulns = append(highVulns, vuln)
		case "MEDIUM", "medium", "Medium":
			mediumVulns = append(mediumVulns, vuln)
		case "LOW", "low", "Low":
			lowVulns = append(lowVulns, vuln)
		default:
			infoVulns = append(infoVulns, vuln)
		}
	}

	// Print summary
	fmt.Println("\n📈 Summary:")
	fmt.Printf("   🔴 High:   %d\n", len(highVulns))
	fmt.Printf("   🟠 Medium: %d\n", len(mediumVulns))
	fmt.Printf("   🟡 Low:    %d\n", len(lowVulns))
	if len(infoVulns) > 0 {
		fmt.Printf("   ℹ️  Info:   %d\n", len(infoVulns))
	}

	// Print vulnerabilities by severity
	if len(highVulns) > 0 {
		fmt.Println("\n🔴 HIGH SEVERITY VULNERABILITIES:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for i, vuln := range highVulns {
			printVulnerability(vuln, i+1)
		}
	}

	if len(mediumVulns) > 0 {
		fmt.Println("\n🟠 MEDIUM SEVERITY VULNERABILITIES:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for i, vuln := range mediumVulns {
			printVulnerability(vuln, i+1)
		}
	}

	if len(lowVulns) > 0 {
		fmt.Println("\n🟡 LOW SEVERITY VULNERABILITIES:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for i, vuln := range lowVulns {
			printVulnerability(vuln, i+1)
		}
	}

	if len(infoVulns) > 0 {
		fmt.Println("\nℹ️  INFO VULNERABILITIES:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for i, vuln := range infoVulns {
			printVulnerability(vuln, i+1)
		}
	}
}

// convertAPIVulnerabilities converts API vulnerability format to CLI format
func (a *Analysis) convertAPIVulnerabilities(apiAnalysis types.Analysis) error {
	a.Vulnerabilities = []vulnerability.Vulnerability{}

	// Convert vulnerabilities from HuskyCIResults
	results := apiAnalysis.HuskyCIResults

	// Go vulnerabilities (Gosec)
	for _, vuln := range results.GoResults.HuskyCIGosecOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Go", "gosec"))
	}
	for _, vuln := range results.GoResults.HuskyCIGosecOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Go", "gosec"))
	}
	for _, vuln := range results.GoResults.HuskyCIGosecOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Go", "gosec"))
	}

	// Python vulnerabilities (Bandit)
	for _, vuln := range results.PythonResults.HuskyCIBanditOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "bandit"))
	}
	for _, vuln := range results.PythonResults.HuskyCIBanditOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "bandit"))
	}
	for _, vuln := range results.PythonResults.HuskyCIBanditOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "bandit"))
	}

	// Python vulnerabilities (Safety)
	for _, vuln := range results.PythonResults.HuskyCISafetyOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "safety"))
	}
	for _, vuln := range results.PythonResults.HuskyCISafetyOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "safety"))
	}
	for _, vuln := range results.PythonResults.HuskyCISafetyOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Python", "safety"))
	}

	// Ruby vulnerabilities (Brakeman)
	for _, vuln := range results.RubyResults.HuskyCIBrakemanOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Ruby", "brakeman"))
	}
	for _, vuln := range results.RubyResults.HuskyCIBrakemanOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Ruby", "brakeman"))
	}
	for _, vuln := range results.RubyResults.HuskyCIBrakemanOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Ruby", "brakeman"))
	}

	// JavaScript vulnerabilities (NpmAudit)
	for _, vuln := range results.JavaScriptResults.HuskyCINpmAuditOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "npmaudit"))
	}
	for _, vuln := range results.JavaScriptResults.HuskyCINpmAuditOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "npmaudit"))
	}
	for _, vuln := range results.JavaScriptResults.HuskyCINpmAuditOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "npmaudit"))
	}

	// JavaScript vulnerabilities (YarnAudit)
	for _, vuln := range results.JavaScriptResults.HuskyCIYarnAuditOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "yarnaudit"))
	}
	for _, vuln := range results.JavaScriptResults.HuskyCIYarnAuditOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "yarnaudit"))
	}
	for _, vuln := range results.JavaScriptResults.HuskyCIYarnAuditOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "JavaScript", "yarnaudit"))
	}

	// Java vulnerabilities (SpotBugs)
	for _, vuln := range results.JavaResults.HuskyCISpotBugsOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Java", "spotbugs"))
	}
	for _, vuln := range results.JavaResults.HuskyCISpotBugsOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Java", "spotbugs"))
	}
	for _, vuln := range results.JavaResults.HuskyCISpotBugsOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Java", "spotbugs"))
	}

	// HCL vulnerabilities (TFSec)
	for _, vuln := range results.HclResults.HuskyCITFSecOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "HCL", "tfsec"))
	}
	for _, vuln := range results.HclResults.HuskyCITFSecOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "HCL", "tfsec"))
	}
	for _, vuln := range results.HclResults.HuskyCITFSecOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "HCL", "tfsec"))
	}

	// C# vulnerabilities (SecurityCodeScan)
	for _, vuln := range results.CSharpResults.HuskyCISecurityCodeScanOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "C#", "securitycodescan"))
	}
	for _, vuln := range results.CSharpResults.HuskyCISecurityCodeScanOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "C#", "securitycodescan"))
	}
	for _, vuln := range results.CSharpResults.HuskyCISecurityCodeScanOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "C#", "securitycodescan"))
	}

	// Generic vulnerabilities (Gitleaks)
	for _, vuln := range results.GenericResults.HuskyCIGitleaksOutput.HighVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Generic", "gitleaks"))
	}
	for _, vuln := range results.GenericResults.HuskyCIGitleaksOutput.MediumVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Generic", "gitleaks"))
	}
	for _, vuln := range results.GenericResults.HuskyCIGitleaksOutput.LowVulns {
		a.Vulnerabilities = append(a.Vulnerabilities, convertHuskyCIVulnToCLIVuln(vuln, "Generic", "gitleaks"))
	}

	return nil
}

// convertHuskyCIVulnToCLIVuln converts a HuskyCIVulnerability to CLI Vulnerability format
func convertHuskyCIVulnToCLIVuln(apiVuln types.HuskyCIVulnerability, language, securityTest string) vulnerability.Vulnerability {
	vuln := vulnerability.New()
	vuln.Language = language
	vuln.SecurityTest = securityTest
	vuln.Severity = apiVuln.Severity
	vuln.Confidence = apiVuln.Confidence
	vuln.File = apiVuln.File
	vuln.Line = apiVuln.Line
	vuln.Code = apiVuln.Code
	vuln.Details = apiVuln.Details
	vuln.Type = apiVuln.Type
	if vuln.Type == "" {
		vuln.Type = apiVuln.Title
	}
	vuln.VunerableBelow = apiVuln.VunerableBelow
	vuln.Version = apiVuln.Version
	vuln.Occurrences = apiVuln.Occurrences
	return *vuln
}

// printVulnerability prints a single vulnerability in a formatted way
func printVulnerability(vuln vulnerability.Vulnerability, index int) {
	fmt.Printf("\n[%d] %s\n", index, vuln.Type)
	if vuln.Language != "" {
		fmt.Printf("    Language: %s\n", vuln.Language)
	}
	if vuln.SecurityTest != "" {
		fmt.Printf("    Security Test: %s\n", vuln.SecurityTest)
	}
	if vuln.File != "" {
		fmt.Printf("    File: %s", vuln.File)
		if vuln.Line != "" {
			fmt.Printf(" (Line: %s)", vuln.Line)
		}
		fmt.Println()
	}
	if vuln.Code != "" {
		fmt.Printf("    Code: %s\n", vuln.Code)
	}
	if vuln.Details != "" {
		fmt.Printf("    Details: %s\n", vuln.Details)
	}
	if vuln.Severity != "" {
		fmt.Printf("    Severity: %s", vuln.Severity)
		if vuln.Confidence != "" {
			fmt.Printf(" (Confidence: %s)", vuln.Confidence)
		}
		fmt.Println()
	}
	if vuln.Version != "" {
		fmt.Printf("    Version: %s", vuln.Version)
		if vuln.VunerableBelow != "" {
			fmt.Printf(" (Vulnerable below: %s)", vuln.VunerableBelow)
		}
		fmt.Println()
	}
	if vuln.Occurrences > 1 {
		fmt.Printf("    Occurrences: %d\n", vuln.Occurrences)
	}
}

// HouseCleaning will do stuff to clean the $HOME directory.
func (a *Analysis) HouseCleaning() error {

	zipFilePath, err := config.GetHuskyZipFilePath()
	if err != nil {
		return err
	}

	return util.DeleteHuskyFile(zipFilePath)
}

func (a *Analysis) setZipSize(destination string) error {
	friendlySize, err := util.GetZipFriendlySize(destination)
	if err != nil {
		return err
	}
	a.CompressedFile.Size = friendlySize
	return nil
}

func (a *Analysis) setLanguages(pathReceived string) error {
	err := filepath.Walk(pathReceived,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fileName := info.Name()
			lang, _ := enry.GetLanguageByExtension(fileName)
			a.Languages = util.AppendIfMissing(a.Languages, lang)
			return nil
		})
	if err != nil {
		return err
	}
	if len(a.Languages) == 0 {
		return errors.New("no languages found")
	}
	return nil
}

// generateEnryOutput generates Enry JSON output format (language -> files mapping)
// This matches the format expected by the API: {"Go": ["file1.go", "file2.go"], "Python": ["file1.py"]}
func (a *Analysis) generateEnryOutput(pathReceived string) (string, error) {
	enryMap := make(map[string][]string)
	
	err := filepath.Walk(pathReceived,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Skip directories
			if info.IsDir() {
				return nil
			}
			
			// Get relative path from the root
			relPath, err := filepath.Rel(pathReceived, path)
			if err != nil {
				return err
			}
			
			// Detect language by extension
			lang, _ := enry.GetLanguageByExtension(info.Name())
			if lang != "" && lang != "Text" {
				// Normalize language name to match API expectations (e.g., "Go" not "GoLang")
				normalizedLang := normalizeLanguageName(lang)
				if normalizedLang != "" {
					enryMap[normalizedLang] = append(enryMap[normalizedLang], relPath)
				}
			}
			return nil
		})
	
	if err != nil {
		return "", fmt.Errorf("error generating Enry output: %w", err)
	}
	
	// Convert to JSON
	enryJSON, err := json.Marshal(enryMap)
	if err != nil {
		return "", fmt.Errorf("error marshaling Enry output: %w", err)
	}
	
	return string(enryJSON), nil
}

// normalizeLanguageName normalizes language names to match API expectations
func normalizeLanguageName(lang string) string {
	// Map common language variations to API-expected names
	normalizations := map[string]string{
		"Go":        "Go",
		"Golang":    "Go",
		"Python":    "Python",
		"Ruby":      "Ruby",
		"JavaScript": "JavaScript",
		"TypeScript": "JavaScript", // TypeScript files are often analyzed with JS tools
		"Java":      "Java",
		"C#":        "C#",
		"CSharp":    "C#",
		"HCL":       "HCL",
		"Terraform": "HCL",
	}
	
	if normalized, ok := normalizations[lang]; ok {
		return normalized
	}
	
	// Return empty string for unsupported languages
	// Only return languages that HuskyCI supports
	supportedLanguages := map[string]bool{
		"Go":         true,
		"Python":     true,
		"Ruby":       true,
		"JavaScript": true,
		"Java":       true,
		"C#":         true,
		"HCL":        true,
	}
	
	if supportedLanguages[lang] {
		return lang
	}
	
	return ""
}

// getAvailableSecurityTests returns the huskyCI securityTests available.
// Later on this check can be done using an API endpoint via cache.
func (a *Analysis) getAvailableSecurityTests(languages []string) map[string][]string {

	var list = make(map[string][]string)

	// Language securityTests
	for _, language := range languages {
		switch language {
		case "Go":
			list[language] = []string{"huskyci/gosec"}
		case "Python":
			list[language] = []string{"huskyci/bandit", "huskyci/safety"}
		case "Ruby":
			list[language] = []string{"huskyci/brakeman"}
		case "JavaScript":
			list[language] = []string{"huskyci/npmaudit", "huskyci/yarnaudit"}
		case "Java":
			list[language] = []string{"huskyci/spotbugs"}
		case "HCL":
			list[language] = []string{"huskyci/tfsec"}
		case "C#":
			list[language] = []string{"huskyci/securitycodescan"}
		}
	}

	// Generic securityTests:
	list["Generic"] = []string{"huskyci/gitleaks"}

	return list
}
