package sonarqube

// HuskyCISonarOutput is the struct that holds the Sonar output
type HuskyCISonarOutput struct {
	Rules  []SonarRule  `json:"rules"`
	Issues []SonarIssue `json:"issues"`
}

// SonarRule represents a single rule in the SonarQube Generic Issue Import Format
type SonarRule struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	EngineID           string        `json:"engineId"`
	CleanCodeAttribute string        `json:"cleanCodeAttribute"`
	Type               string        `json:"type"`
	Severity           string        `json:"severity"`
	Impacts            []SonarImpact `json:"impacts"`
}

// SonarImpact represents the impact of a rule on software quality
type SonarImpact struct {
	SoftwareQuality string `json:"softwareQuality"`
	Severity        string `json:"severity"`
}

// SonarIssue represents a single issue in the SonarQube Generic Issue Import Format
type SonarIssue struct {
	RuleID             string          `json:"ruleId"`
	EffortMinutes      int             `json:"effortMinutes,omitempty"`
	PrimaryLocation    SonarLocation   `json:"primaryLocation"`
	SecondaryLocations []SonarLocation `json:"secondaryLocations,omitempty"`
}

// SonarLocation is the struct that holds a vulnerability location within code
type SonarLocation struct {
	Message   string         `json:"message,omitempty"`
	FilePath  string         `json:"filePath"`
	TextRange SonarTextRange `json:"textRange,omitempty"`
}

// SonarTextRange is the struct that holds additional location fields
type SonarTextRange struct {
	StartLine   int `json:"startLine,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}
