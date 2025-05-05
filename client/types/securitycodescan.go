package types

import "encoding/json"

// SecurityCodeScanOutput is the struct that holds all data from SecurityCodeScan output.
type SecurityCodeScanOutput struct {
	Warnings json.RawMessage          `json:"warnings"`
	Results  []SecurityCodeScanResult `json:"results"`
}

// SecurityCodeScanResult is the struct that holds detailed information of results from SecurityCodeScan output.
type SecurityCodeScanResult struct {
	RuleID      string `json:"rule_id"`
	Link        string `json:"link"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}
