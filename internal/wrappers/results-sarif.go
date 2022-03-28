package wrappers

var (
	SarifName           = "Checkmarx AST"
	SarifVersion        = "1.0"
	SarifInformationURI = "https://checkmarx.atlassian.net/wiki/spaces/AST/pages/5844861345/CxAST+Documentation"
)

type SarifResultsCollection struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SarifRun `json:"runs"`
}

type SarifRun struct {
	Tool    SarifTool         `json:"tool"`
	Results []SarifScanResult `json:"results"`
}

type SarifTool struct {
	Driver SarifDriver `json:"driver"`
}

type SarifDriver struct {
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	InformationURI string            `json:"informationUri"`
	Rules          []SarifDriverRule `json:"rules"`
}

type SarifDriverRule struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	HelpURI string `json:"helpUri"`
}

type SarifScanResult struct {
	RuleID              string                  `json:"ruleId"`
	Level               string                  `json:"level"`
	Message             SarifMessage            `json:"message"`
	PartialFingerprints *SarifResultFingerprint `json:"partialFingerprints,omitempty"`
	Locations           []SarifLocation         `json:"locations,omitempty"`
}

type SarifLocation struct {
	PhysicalLocation SarifPhysicalLocation `json:"physicalLocation"`
}

type SarifPhysicalLocation struct {
	ArtifactLocation SarifArtifactLocation `json:"artifactLocation"`
	Region           *SarifRegion          `json:"region,omitempty"`
}

type SarifRegion struct {
	StartLine   uint `json:"startLine,omitempty"`
	StartColumn uint `json:"startColumn,omitempty"`
	EndColumn   uint `json:"endColumn,omitempty"`
}

type SarifArtifactLocation struct {
	URI string `json:"uri"`
}

type SarifMessage struct {
	Text string `json:"text"`
}

type SarifResultFingerprint struct {
	PrimaryLocationLineHash string `json:"primaryLocationLineHash,omitempty"`
}
