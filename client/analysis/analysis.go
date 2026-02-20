// Copyright 2019 Globo.com authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package analysis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/huskyci-org/huskyCI/client/config"
	"github.com/huskyci-org/huskyCI/client/types"
	"github.com/huskyci-org/huskyCI/client/util"
)

// StartAnalysis starts a container and returns its RID and error.
func StartAnalysis() (string, error) {

	// preparing POST to HuskyCI
	huskyStartAnalysisURL := config.HuskyAPI + "/analysis"

	requestPayload := types.JSONPayload{
		RepositoryURL:      config.RepositoryURL,
		RepositoryBranch:   config.RepositoryBranch,
		LanguageExclusions: config.LanguageExclusions,
	}

	marshalPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	httpClient, err := util.NewClient(config.HuskyUseTLS)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", huskyStartAnalysisURL, bytes.NewBuffer(marshalPayload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Husky-Token", config.HuskyToken)
	req.Header.Add("User-Agent", "huskyci-client")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 401 {
			errorMsg := fmt.Sprintf("Authentication failed: The provided Husky-Token is invalid or expired.\n\nTip: Generate a new token using the huskyCI API or verify your token has access to repository: %s", config.RepositoryURL)
			return "", errors.New(errorMsg)
		}
		if resp.StatusCode == 400 {
			errorMsg := fmt.Sprintf("Bad request: Invalid request parameters.\n\nStatus: %d\nResponse: %s\n\nTip: Verify that the repository URL and branch are correct", resp.StatusCode, string(body))
			return "", errors.New(errorMsg)
		}
		if resp.StatusCode == 409 {
			errorMsg := fmt.Sprintf("Conflict: An analysis is already running for this repository and branch.\n\nStatus: %d\nResponse: %s\n\nTip: Wait for the existing analysis to complete or use a different branch", resp.StatusCode, string(body))
			return "", errors.New(errorMsg)
		}
		errorMsg := fmt.Sprintf("Failed to start analysis: Unexpected response from API.\n\nStatus Code: %d\nResponse: %s\n\nTip: Check the huskyCI API status and try again", resp.StatusCode, string(body))
		return "", errors.New(errorMsg)
	}

	RID := resp.Header.Get("X-Request-Id")
	if RID == "" {
		errorMsg := "Failed to start analysis: No request ID (RID) received from the API.\n\nTip: This may indicate an issue with the huskyCI API. Please check the API status and try again."
		return "", errors.New(errorMsg)
	}

	// Setting analysis values on the JSON output
	outputJSON.Summary.URL = requestPayload.RepositoryURL
	outputJSON.Summary.Branch = requestPayload.RepositoryBranch
	outputJSON.Summary.RID = RID

	return RID, nil
}

// GetAnalysis gets the results of an analysis.
func GetAnalysis(RID string) (types.Analysis, error) {

	analysis := types.Analysis{}
	getAnalysisURL := config.HuskyAPI + "/analysis/" + RID

	if !types.IsJSONoutput {
		fmt.Printf("[HUSKYCI] Checking analysis status (RID: %s)...\n", RID)
	}

	httpClient, err := util.NewClient(config.HuskyUseTLS)
	if err != nil {
		return analysis, err
	}

	req, err := http.NewRequest("GET", getAnalysisURL, nil)
	if err != nil {
		return analysis, err
	}

	req.Header.Add("Husky-Token", config.HuskyToken)
	req.Header.Add("User-Agent", "huskyci-client")

	resp, err := httpClient.Do(req)
	if err != nil {
		return analysis, fmt.Errorf("network error while fetching analysis: %w\n\nTip: Check your network connection and verify the API endpoint is accessible", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 404 {
			errorMsg := fmt.Sprintf("Analysis not found: No analysis found with RID '%s'.\n\nTip: Verify the RID is correct and the analysis exists", RID)
			return analysis, errors.New(errorMsg)
		}
		if resp.StatusCode == 401 {
			errorMsg := fmt.Sprintf("Authentication failed: Invalid or expired token.\n\nTip: Generate a new token using the huskyCI API", RID)
			return analysis, errors.New(errorMsg)
		}
		errorMsg := fmt.Sprintf("Failed to retrieve analysis: Unexpected response from API.\n\nStatus Code: %d\nResponse: %s\n\nTip: Check the huskyCI API status and try again", resp.StatusCode, string(body))
		return analysis, errors.New(errorMsg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return analysis, err
	}

	err = json.Unmarshal(body, &analysis)
	if err != nil {
		return analysis, err
	}

	return analysis, nil
}

// MonitorAnalysis will keep monitoring an analysis until it has finished or timed out.
func MonitorAnalysis(RID string) (types.Analysis, error) {

	analysis := types.Analysis{}
	timeout := time.After(60 * time.Minute)
	retryTick := time.NewTicker(60 * time.Second)
	checkCount := 0

	if !types.IsJSONoutput {
		fmt.Println("[HUSKYCI] Monitoring analysis progress...")
		fmt.Printf("[HUSKYCI] Analysis RID: %s\n", RID)
		fmt.Println("[HUSKYCI] This may take several minutes depending on your codebase size...")
	}

	for {
		select {
		case <-timeout:
			return analysis, fmt.Errorf("analysis timed out after 60 minutes\n\nTip: Large codebases may take longer to analyze. Try again or contact support if this persists")
		case <-retryTick.C:
			checkCount++
			analysis, err := GetAnalysis(RID)
			if err != nil {
				return analysis, err
			}
			if analysis.Status == "finished" {
				if !types.IsJSONoutput {
					fmt.Printf("[HUSKYCI] ✓ Analysis completed after %d checks\n", checkCount)
				}
				return analysis, nil
			} else if analysis.Status == "error running" {
				errorMsg := fmt.Sprintf("Analysis failed with error: %v\n\nTip: Check the analysis details for more information about what went wrong", analysis.ErrorFound)
				return analysis, fmt.Errorf(errorMsg)
			}
			if !types.IsJSONoutput {
				fmt.Printf("[HUSKYCI] ⏳ Analysis in progress... (check #%d)\n", checkCount)
			}
		}
	}
}

// PrintResults prints huskyCI output either in JSON or the standard output.
func PrintResults(analysis types.Analysis) error {

	prepareAllSummary(analysis)

	if types.IsJSONoutput {
		err := printJSONOutput()
		if err != nil {
			return err
		}
	} else {
		printSTDOUTOutput(analysis)
	}

	return nil
}
