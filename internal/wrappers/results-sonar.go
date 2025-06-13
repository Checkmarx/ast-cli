package wrappers

type ScanResultsSonar struct {
	Rules  []SonarRules  `json:"rules"`
	Issues []SonarIssues `json:"issues"`
}

type SonarRules struct {
	ID                 string         `json:"id,omitempty"`
	Name               string         `json:"name,omitempty"`
	Description        string         `json:"description,omitempty"`
	EngineID           string         `json:"engineId,omitempty"`
	CleanCodeAttribute string         `json:"cleanCodeAttribute,omitempty"`
	Impacts            []SonarImpacts `json:"impacts"`
}

type SonarImpacts struct {
	SoftwareQuality string `json:"softwareQuality,omitempty"`
	Severity        string `json:"severity,omitempty"`
}

type SonarIssues struct {
	RuleID             string          `json:"ruleId,omitempty"`
	PrimaryLocation    SonarLocation   `json:"primaryLocation"`
	EffortMinutes      uint            `json:"effortMinutes,omitempty"`
	SecondaryLocations []SonarLocation `json:"secondaryLocations"`
}

type SonarLocation struct {
	Message   string         `json:"message,omitempty"`
	FilePath  string         `json:"filePath,omitempty"`
	TextRange SonarTextRange `json:"textRange"`
}

type SonarTextRange struct {
	StartLine   uint `json:"startLine,omitempty"`
	EndLine     uint `json:"endLine,omitempty"`
	StartColumn uint `json:"startColumn,omitempty"`
	EndColumn   uint `json:"endColumn,omitempty"`
}
