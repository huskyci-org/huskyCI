package securitytest

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/util"
)

// EnryOutput is the struct that holds all data from Gosec output.
type EnryOutput struct {
	Codes []types.Code
}

func analyzeEnry(enryScan *SecTestScanInfo) error {
	// Unmarshall rawOutput into finalOutput, that is a EnryOutput struct.
	if err := json.Unmarshal([]byte(enryScan.Container.COutput), &enryScan.FinalOutput); err != nil {
		log.Error("analyzeEnry", "ENRY", 1003, enryScan.Container.COutput, err)
		enryScan.ErrorFound = util.HandleScanError(enryScan.Container.COutput, err)
		return enryScan.ErrorFound
	}
	// get all languages and files found based on Enry output
	if err := enryScan.prepareEnryOutput(); err != nil {
		enryScan.ErrorFound = util.HandleScanError(enryScan.Container.COutput, err)
		return enryScan.ErrorFound
	}
	return nil
}

func (enryScan *SecTestScanInfo) prepareEnryOutput() error {
	repositoryLanguages := []types.Code{}
	mapLanguages := make(map[string][]interface{})
	
	// Log the raw enry output for debugging
	outputPreview := enryScan.Container.COutput
	if len(outputPreview) > 500 {
		outputPreview = outputPreview[:500] + "..."
	}
	log.Info("prepareEnryOutput", "ENRY", 16, fmt.Sprintf("Enry raw output (first 500 chars): %s", outputPreview))
	
	err := json.Unmarshal([]byte(enryScan.Container.COutput), &mapLanguages)
	if err != nil {
		log.Error("prepareEnryOutput", "ENRY", 1003, enryScan.Container.COutput, err)
		return err
	}
	
	// Log parsed languages for debugging
	log.Info("prepareEnryOutput", "ENRY", 16, fmt.Sprintf("Parsed %d languages from enry output", len(mapLanguages)))
	
	for name, files := range mapLanguages {
		fs := []string{}
		for _, f := range files {
			if reflect.TypeOf(f).String() == "string" {
				fs = append(fs, f.(string))
			} else {
				errMsg := errors.New("error mapping languages")
				log.Error("prepareEnryOutput", "ENRY", 1032, errMsg)
				return errMsg
			}
		}

		if !enryScan.LanguageExclusions[name] {
			newLanguage := types.Code{
				Language: name,
				Files:    fs,
			}
			repositoryLanguages = append(repositoryLanguages, newLanguage)
			log.Info("prepareEnryOutput", "ENRY", 16, fmt.Sprintf("Added language: %s with %d files", name, len(fs)))
		} else {
			log.Info("prepareEnryOutput", "ENRY", 16, fmt.Sprintf("Skipped excluded language: %s", name))
		}
	}

	enryScan.Codes = repositoryLanguages
	log.Info("prepareEnryOutput", "ENRY", 16, fmt.Sprintf("Final Codes count: %d", len(enryScan.Codes)))
	return nil
}

// ParseProvidedEnryOutput parses Enry JSON output provided by CLI and populates enryScan.Codes
// This is used for file:// URLs to avoid docker-in-docker issues
func (enryScan *SecTestScanInfo) ParseProvidedEnryOutput(enryOutputJSON string, languageExclusions map[string]bool) error {
	repositoryLanguages := []types.Code{}
	mapLanguages := make(map[string][]interface{})
	
	log.Info("parseProvidedEnryOutput", "ENRY", 16, fmt.Sprintf("Parsing provided Enry output (first 500 chars): %s", enryOutputJSON[:min(500, len(enryOutputJSON))]))
	
	err := json.Unmarshal([]byte(enryOutputJSON), &mapLanguages)
	if err != nil {
		log.Error("parseProvidedEnryOutput", "ENRY", 1003, enryOutputJSON, err)
		return err
	}
	
	log.Info("parseProvidedEnryOutput", "ENRY", 16, fmt.Sprintf("Parsed %d languages from provided Enry output", len(mapLanguages)))
	
	for name, files := range mapLanguages {
		fs := []string{}
		for _, f := range files {
			if reflect.TypeOf(f).String() == "string" {
				fs = append(fs, f.(string))
			} else {
				errMsg := errors.New("error mapping languages")
				log.Error("parseProvidedEnryOutput", "ENRY", 1032, errMsg)
				return errMsg
			}
		}

		if !languageExclusions[name] {
			newLanguage := types.Code{
				Language: name,
				Files:    fs,
			}
			repositoryLanguages = append(repositoryLanguages, newLanguage)
			log.Info("parseProvidedEnryOutput", "ENRY", 16, fmt.Sprintf("Added language: %s with %d files", name, len(fs)))
		} else {
			log.Info("parseProvidedEnryOutput", "ENRY", 16, fmt.Sprintf("Skipped excluded language: %s", name))
		}
	}

	enryScan.Codes = repositoryLanguages
	log.Info("parseProvidedEnryOutput", "ENRY", 16, fmt.Sprintf("Final Codes count: %d", len(enryScan.Codes)))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
