package analysis

import (
	"errors"
	"fmt"
	"os"
	"time"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/securitytest"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/util"
	apiUtil "github.com/huskyci-org/huskyCI/api/util/api"
	"go.mongodb.org/mongo-driver/bson"
)

const logActionStart = "StartAnalysis"
const logInfoAnalysis = "ANALYSIS"

// StartAnalysis starts the analysis given a RID and a repository.
func StartAnalysis(RID string, repository types.Repository) {
	// step 1: create a new analysis into MongoDB based on repository received
	if err := registerNewAnalysis(RID, repository); err != nil {
		return
	}
	log.Info(logActionStart, logInfoAnalysis, 101, RID)

	// step 2: run enry as huskyCI initial step
	enryScan := securitytest.SecTestScanInfo{}
	enryScan.SecurityTestName = "enry"
	allScansResults := securitytest.RunAllInfo{}

	defer func() {
		err := registerFinishedAnalysis(RID, &allScansResults)
		if err != nil {
			log.Error(logActionStart, logInfoAnalysis, 2011, err)
		}
	}()

	infrastructureSelected, hasSelected := os.LookupEnv("HUSKYCI_INFRASTRUCTURE_USE")
	if !hasSelected {
		err := errors.New("HUSKYCI_INFRASTRUCTURE_USE environment variable not set")
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
		return
	}

	var apiHost string

	if infrastructureSelected == "docker" {
		dockerAPIHost, err := apiContext.APIConfiguration.DBInstance.FindAndModifyDockerAPIAddresses()
		if err != nil {
			log.Error(logActionStart, logInfoAnalysis, 2011, err)
			return
		}

		configAPI, err := apiContext.DefaultConf.GetAPIConfig()
		if err != nil {
			log.Error(logActionStart, logInfoAnalysis, 2011, err)
			return
		}

		apiHost, err = apiUtil.FormatDockerHostAddress(dockerAPIHost, configAPI)
		if err != nil {
			log.Error(logActionStart, logInfoAnalysis, 2011, err)
			return
		}
	} else if infrastructureSelected == "kubernetes" {
		// Assume that the Kubernetes host is set properly in the configuration or environment variables
		// Implement any specific logic to get the Kubernetes API host if needed
		apiHost = "kubernetes.default.svc" // Example host, replace with actual logic if needed
	} else {
		err := errors.New("invalid HUSKYCI_INFRASTRUCTURE_USE value")
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
		return
	}

	log.Info("StartAnalysisTest", apiHost, 2012, RID)

	// For file:// URLs, check if Enry output was provided by CLI
	// This avoids docker-in-docker issues where Enry can't see extracted files
	if util.IsFileURL(repository.URL) && repository.EnryOutput != "" {
		log.Info(logActionStart, logInfoAnalysis, 16, fmt.Sprintf("Using Enry output provided by CLI for file:// URL: %s", repository.URL))
		// Parse the provided Enry output and populate enryScan.Codes directly
		if err := enryScan.ParseProvidedEnryOutput(repository.EnryOutput, repository.LanguageExclusions); err != nil {
			log.Error(logActionStart, logInfoAnalysis, 2011, fmt.Errorf("failed to parse provided Enry output: %w", err))
			// Fall back to running Enry if parsing fails
			log.Info(logActionStart, logInfoAnalysis, 16, "Falling back to running Enry in container")
		} else {
			log.Info(logActionStart, logInfoAnalysis, 16, fmt.Sprintf("Successfully parsed %d languages from provided Enry output", len(enryScan.Codes)))
			// Skip running Enry since we have the output
			goto skipEnryRun
		}
	}

	if err := enryScan.New(RID, repository.URL, repository.Branch, enryScan.SecurityTestName, repository.LanguageExclusions, apiHost); err != nil {
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
		return
	}
	if err := enryScan.Start(); err != nil {
		allScansResults.SetAnalysisError(err)
		return
	}

skipEnryRun:

	// step 3: run generic and languages security tests based on enryScan result in parallel
	if err := allScansResults.Start(enryScan); err != nil {
		allScansResults.SetAnalysisError(err)
		return
	}

	log.Info("StartAnalysis", logInfoAnalysis, 102, RID)
}

func registerNewAnalysis(RID string, repository types.Repository) error {

	newAnalysis := types.Analysis{
		RID:       RID,
		URL:       repository.URL,
		Branch:    repository.Branch,
		Status:    "running",
		StartedAt: time.Now(),
	}

	if err := apiContext.APIConfiguration.DBInstance.InsertDBAnalysis(newAnalysis); err != nil {
		log.Error("registerNewAnalysis", logInfoAnalysis, 2011, err)
		return err
	}

	// log.Info("registerNewAnalysis", logInfoAnalysis, 2012
	return nil
}

func registerFinishedAnalysis(RID string, allScanResults *securitytest.RunAllInfo) error {
	analysisQuery := map[string]interface{}{"RID": RID}
	var errorString string
	if _, ok := allScanResults.ErrorFound.(error); ok {
		errorString = allScanResults.ErrorFound.Error()
	} else {
		errorString = ""
	}
	updateAnalysisQuery := bson.M{
		"status":         allScanResults.Status,
		"commitAuthors":  allScanResults.CommitAuthors,
		"result":         allScanResults.FinalResult,
		"containers":     allScanResults.Containers,
		"huskyciresults": allScanResults.HuskyCIResults,
		"codes":          allScanResults.Codes,
		"errorFound":     errorString,
		"finishedAt":     time.Now(),
	}

	if err := apiContext.APIConfiguration.DBInstance.UpdateOneDBAnalysisContainer(analysisQuery, updateAnalysisQuery); err != nil {
		log.Error("registerFinishedAnalysis", logInfoAnalysis, 2011, err)
		return err
	}
	return nil
}
