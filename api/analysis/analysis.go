package analysis

import (
	"time"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/securitytest"
	"github.com/huskyci-org/huskyCI/api/types"
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

	dockerAPIHost, err := apiContext.APIConfiguration.DBInstance.FindAndModifyDockerAPIAddresses()
	if err != nil {
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
	}

	configAPI, err := apiContext.DefaultConf.GetAPIConfig()
	if err != nil {
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
	}

	dockerHost := apiUtil.FormatDockerHostAddress(dockerAPIHost, configAPI)

	log.Info("StartAnalysisTest", dockerHost, 2012, RID)

	if err := enryScan.New(RID, repository.URL, repository.Branch, enryScan.SecurityTestName, repository.LanguageExclusions, dockerHost); err != nil {
		log.Error(logActionStart, logInfoAnalysis, 2011, err)
		return
	}
	if err := enryScan.Start(); err != nil {
		allScansResults.SetAnalysisError(err)
		return
	}

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
