package wrappers

type ScanResultsSonar struct {
	Results []SonarIssues `json:"issues"`
}

type SonarIssues struct {
	EngineID           string          `json:"engineId,omitempty"`
	RuleID             string          `json:"ruleId,omitempty"`
	Severity           string          `json:"severity,omitempty"`
	Type               string          `json:"type,omitempty"`
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
