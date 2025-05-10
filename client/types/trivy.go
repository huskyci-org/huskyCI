package types

// TrivyOutput is the struct that holds all data from Trivy output.
type TrivyOutput struct {
	Results []TrivyResult `json:"Results"`
}

// TrivyResult is the struct that holds detailed information of results from Trivy output.
type TrivyResult struct {
	Target          string               `json:"Target"`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities"`
}

// TrivyVulnerability is the struct that holds detailed information of each vulnerability found by Trivy.
type TrivyVulnerability struct {
	VulnerabilityID string `json:"VulnerabilityID"`
	PkgName         string `json:"PkgName"`
	Severity        string `json:"Severity"`
	Description     string `json:"Description"`
}
