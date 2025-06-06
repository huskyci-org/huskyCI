package securitytest

import (
	"encoding/json"
	"errors"
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
	err := json.Unmarshal([]byte(enryScan.Container.COutput), &mapLanguages)
	if err != nil {
		log.Error("prepareEnryOutput", "ENRY", 1003, enryScan.Container.COutput, err)
		return err
	}
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
		}
	}

	enryScan.Codes = repositoryLanguages
	return nil
}
