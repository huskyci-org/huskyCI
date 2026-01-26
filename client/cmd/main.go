// Copyright 2019 Globo.com authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/huskyci-org/huskyCI/client/integration/sonarqube"

	"github.com/huskyci-org/huskyCI/client/analysis"
	"github.com/huskyci-org/huskyCI/client/config"
	"github.com/huskyci-org/huskyCI/client/types"
)

const (
	huskyCIPrefix = "[HUSKYCI][*]"
	msgNoBlockingVulns = "[HUSKYCI][*] The following securityTests were executed and no blocking vulnerabilities were found:"
	msgSecurityTestsFailed = "[HUSKYCI][*] The following securityTests failed to run:"
	msgNoIssuesFound = "[HUSKYCI][*] No issues were found."
	msgLowInfoIssuesFound = "[HUSKYCI][*] However, some LOW/INFO issues were found..."
	msgHighMediumIssuesFound = "[HUSKYCI][*] Some HIGH/MEDIUM issues were found in these securityTests:"
)

func main() {

	types.FoundVuln = false
	setJSONOutputFlag()

	// step 0: check and set huskyci-client configuration
	if err := initializeConfig(); err != nil {
		if !types.IsJSONoutput {
			fmt.Fprintf(os.Stderr, "\n❌ Configuration Error:\n%s\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[HUSKYCI][ERROR] Configuration error: %s\n", err)
		}
		os.Exit(1)
	}

	// step 1: start analysis and get its RID.
	RID, err := startAnalysis()
	if err != nil {
		if !types.IsJSONoutput {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to start analysis:\n%s\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[HUSKYCI][ERROR] Failed to start analysis: %s\n", err)
		}
		os.Exit(1)
	}

	// step 2.1: keep querying huskyCI API to check if a given analysis has already finished.
	huskyAnalysis, err := analysis.MonitorAnalysis(RID)
	if err != nil {
		if !types.IsJSONoutput {
			fmt.Fprintf(os.Stderr, "\n❌ Analysis monitoring failed:\n%s\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[HUSKYCI][ERROR] Analysis monitoring failed (RID: %s): %s\n", RID, err)
		}
		os.Exit(1)
	}

	// step 2.2: prepare the list of securityTests that ran in the analysis.
	passedList, failedList, errorList := categorizeSecurityTests(huskyAnalysis)

	// step 3: print output based on os.Args(1) parameter received
	setJSONOutputFlag()

	err = analysis.PrintResults(huskyAnalysis)
	if err != nil {
		if !types.IsJSONoutput {
			fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Failed to print results: %s\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[HUSKYCI][ERROR] Failed to print results: %s\n", err)
		}
		// Don't exit here, continue to SonarQube output generation
	}

	// step 3.5: integration with SonarQube
	if err := generateSonarQubeOutput(huskyAnalysis); err != nil {
		if !types.IsJSONoutput {
			fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Failed to generate SonarQube output file: %s\n", err)
			fmt.Fprintf(os.Stderr, "Tip: The analysis completed successfully, but SonarQube integration output could not be generated.\n")
		} else {
			fmt.Fprintf(os.Stderr, "[HUSKYCI][ERROR] Failed to generate SonarQube JSON file: %s\n", err)
		}
		// Don't exit here, continue to vulnerability handling
	}

	// step 4: block developer CI if vulnerabilities were found
	exitCode := handleVulnerabilityResults(passedList, failedList, errorList)
	os.Exit(exitCode)
}

func setJSONOutputFlag() {
	types.IsJSONoutput = len(os.Args) > 1 && os.Args[1] == "JSON"
}

func printErrorIfNotJSON(message string, err error) {
	if !types.IsJSONoutput {
		fmt.Println(message, err)
	}
}

func initializeConfig() error {
	if err := config.CheckEnvVars(); err != nil {
		return err
	}
	config.SetConfigs()
	return nil
}

func startAnalysis() (string, error) {
	if !types.IsJSONoutput {
		fmt.Println("🚀 Starting huskyCI analysis...")
		fmt.Printf("📦 Repository: %s\n", config.RepositoryURL)
		fmt.Printf("🌿 Branch: %s\n", config.RepositoryBranch)
		fmt.Println()
	}

	RID, err := analysis.StartAnalysis()
	if err != nil {
		return "", err
	}

	if !types.IsJSONoutput {
		fmt.Printf("✓ Analysis started successfully!\n")
		fmt.Printf("📋 Request ID (RID): %s\n", RID)
		fmt.Println()
	}

	return RID, nil
}

func categorizeSecurityTests(huskyAnalysis types.Analysis) ([]string, []string, []string) {
	var passedList []string
	var failedList []string
	var errorList []string

	for _, container := range huskyAnalysis.Containers {
		securityTestFullName := fmt.Sprintf("%s:%s", container.SecurityTest.Image, container.SecurityTest.ImageTag)
		switch {
		case container.CResult == "passed" && container.SecurityTest.Name != "gitauthors":
			passedList = append(passedList, securityTestFullName)
		case container.CResult == "failed":
			failedList = append(failedList, securityTestFullName)
		case container.CResult == "error":
			errorList = append(errorList, securityTestFullName)
		}
	}

	return passedList, failedList, errorList
}

func generateSonarQubeOutput(huskyAnalysis types.Analysis) error {
	outputPath := "./huskyCI/"
	outputFileName := "sonarqube.json"

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return sonarqube.GenerateOutputFile(huskyAnalysis, outputPath, outputFileName)
}

func handleVulnerabilityResults(passedList, failedList, errorList []string) int {
	switch {
	case !types.FoundVuln && !types.FoundInfo:
		printNoVulnerabilitiesFound(passedList, errorList)
		return 0
	case !types.FoundVuln && types.FoundInfo:
		printInfoVulnerabilitiesFound(passedList, errorList)
		return 0
	default:
		printVulnerabilitiesFound(passedList, failedList, errorList)
		return 190
	}
}

func printNoVulnerabilitiesFound(passedList, errorList []string) {
	if !types.IsJSONoutput {
		printErrorList(errorList)
		fmt.Println(msgNoBlockingVulns)
		fmt.Println(huskyCIPrefix, passedList)
		fmt.Println(msgNoIssuesFound)
	}
}

func printInfoVulnerabilitiesFound(passedList, errorList []string) {
	if !types.IsJSONoutput {
		printErrorList(errorList)
		fmt.Println(msgNoBlockingVulns)
		fmt.Println(huskyCIPrefix, passedList)
		fmt.Println(msgLowInfoIssuesFound)
	}
}

func printVulnerabilitiesFound(passedList, failedList, errorList []string) {
	if !types.IsJSONoutput {
		printErrorList(errorList)
		if len(passedList) > 0 {
			fmt.Println(msgNoBlockingVulns)
			fmt.Println(huskyCIPrefix, passedList)
		}
		fmt.Println(msgHighMediumIssuesFound)
		fmt.Println(huskyCIPrefix, failedList)
	}
}

func printErrorList(errorList []string) {
	if len(errorList) > 0 {
		fmt.Println(msgSecurityTestsFailed)
		fmt.Println(huskyCIPrefix, errorList)
	}
}
