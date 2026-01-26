package securitytest

import (
	"encoding/json"

	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/types"
)

// TrivyOutput represents the output structure from a Trivy security scan.
type TrivyOutput struct {
	Results []TrivyResult `json:"Results"`
}

// TrivyResult represents a single scan result from Trivy, containing target information and vulnerabilities.
type TrivyResult struct {
	Target          string `json:"Target"`
	Vulnerabilities []struct {
		VulnerabilityID string `json:"VulnerabilityID"`
		PkgName         string `json:"PkgName"`
		Severity        string `json:"Severity"`
		Description     string `json:"Description"`
	} `json:"Vulnerabilities"`
}

func analyzeTrivy(trivyScan *SecTestScanInfo) error {
	trivyOutput := TrivyOutput{}
	if err := json.Unmarshal([]byte(trivyScan.Container.COutput), &trivyOutput); err != nil {
		log.Error("analyzeTrivy", "TRIVY", 1040, trivyScan.Container.COutput, err)
		trivyScan.ErrorFound = err
		return err
	}

	trivyScan.FinalOutput = trivyOutput
	trivyScan.prepareTrivyVulns()
	return nil
}

func (trivyScan *SecTestScanInfo) prepareTrivyVulns() {
	trivyOutput := trivyScan.FinalOutput.(TrivyOutput)
	huskyCITrivyResults := types.HuskyCISecurityTestOutput{}

	for _, result := range trivyOutput.Results {
		for _, vuln := range result.Vulnerabilities {
			trivyVuln := types.HuskyCIVulnerability{
				Language:     "generic",
				SecurityTool: "Trivy",
				Severity:     vuln.Severity,
				Title:        vuln.VulnerabilityID,
				Details:      vuln.Description,
				File:         result.Target,
			}

			switch vuln.Severity {
			case "LOW":
				huskyCITrivyResults.LowVulns = append(huskyCITrivyResults.LowVulns, trivyVuln)
			case "MEDIUM":
				huskyCITrivyResults.MediumVulns = append(huskyCITrivyResults.MediumVulns, trivyVuln)
			case "HIGH":
				huskyCITrivyResults.HighVulns = append(huskyCITrivyResults.HighVulns, trivyVuln)
			}
		}
	}

	trivyScan.Vulnerabilities = huskyCITrivyResults
}
