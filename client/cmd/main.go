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

func main() {

	types.FoundVuln = false
	setJSONOutputFlag()

	// step 0: check and set huskyci-client configuration
	if err := initializeConfig(); err != nil {
		printErrorIfNotJSON("[HUSKYCI][ERROR] Check environment variables:", err)
		os.Exit(1)
	}

	// step 1: start analysis and get its RID.
	RID, err := startAnalysis()
	if err != nil {
		fmt.Println("[HUSKYCI][ERROR] Sending request to huskyCI:", err)
		os.Exit(1)
	}

	// step 2.1: keep querying huskyCI API to check if a given analysis has already finished.
	huskyAnalysis, err := analysis.MonitorAnalysis(RID)
	if err != nil {
		s := fmt.Sprintf("[HUSKYCI][ERROR] Monitoring analysis %s: %s", RID, err)
		fmt.Println(s)
		os.Exit(1)
	}

	// step 2.2: prepare the list of securityTests that ran in the analysis.
	passedList, failedList, errorList := categorizeSecurityTests(huskyAnalysis)

	// step 3: print output based on os.Args(1) parameter received
	setJSONOutputFlag()

	err = analysis.PrintResults(huskyAnalysis)
	if err != nil {
		fmt.Println("[HUSKYCI][ERROR] Printing output:", err)
		os.Exit(1)
	}

	// step 3.5: integration with SonarQube
	if err := generateSonarQubeOutput(huskyAnalysis); err != nil {
		fmt.Println("[ERROR] Failed to generate SonarQube JSON file:", err)
		os.Exit(1)
	}

	// step 4: block developer CI if vulnerabilities were found
	if !types.FoundVuln && !types.FoundInfo {
		if !types.IsJSONoutput {
			if len(errorList) > 0 {
				fmt.Println("[HUSKYCI][*] The following securityTests failed to run:")
				fmt.Println("[HUSKYCI][*]", errorList)
			}
			fmt.Println("[HUSKYCI][*] The following securityTests were executed and no blocking vulnerabilities were found:")
			fmt.Println("[HUSKYCI][*]", passedList)
			fmt.Println("[HUSKYCI][*] No issues were found.")
		}
		os.Exit(0)
	}

	if !types.FoundVuln && types.FoundInfo {
		if !types.IsJSONoutput {
			if len(errorList) > 0 {
				fmt.Println("[HUSKYCI][*] The following securityTests failed to run:")
				fmt.Println("[HUSKYCI][*]", errorList)
			}
			fmt.Println("[HUSKYCI][*] The following securityTests were executed and no blocking vulnerabilities were found:")
			fmt.Println("[HUSKYCI][*]", passedList)
			fmt.Println("[HUSKYCI][*] However, some LOW/INFO issues were found...")
		}
		os.Exit(0)
	}

	if types.FoundVuln && !types.IsJSONoutput {
		if len(errorList) > 0 {
			fmt.Println("[HUSKYCI][*] The following securityTests failed to run:")
			fmt.Println("[HUSKYCI][*]", errorList)
		}
		if len(passedList) > 0 {
			fmt.Println("[HUSKYCI][*] The following securityTests were executed and no blocking vulnerabilities were found:")
			fmt.Println("[HUSKYCI][*]", passedList)
		}
		fmt.Println("[HUSKYCI][*] Some HIGH/MEDIUM issues were found in these securityTests:")
		fmt.Println("[HUSKYCI][*]", failedList)
	}

	os.Exit(190)
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
		s := fmt.Sprintf("[HUSKYCI][*] %s -> %s", config.RepositoryBranch, config.RepositoryURL)
		fmt.Println(s)
	}

	RID, err := analysis.StartAnalysis()
	if err != nil {
		return "", err
	}

	if !types.IsJSONoutput {
		fmt.Println("[HUSKYCI][*] huskyCI analysis started! RID:", RID)
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
