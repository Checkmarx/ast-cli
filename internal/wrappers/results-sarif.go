package wrappers

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
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Rules   []SarifDriverRule `json:"rules"`
}

type SarifDriverRule struct {
	ID   interface{} `json:"id"`
	Name string      `json:"name"`
}

type SarifScanResult struct {
	RuleID              interface{}            `json:"ruleId"`
	Message             SarifMessage           `json:"message"`
	PartialFingerprints SarifResultFingerprint `json:"partialFingerprints"`
	Locations           []SarifLocation        `json:"locations"`
}

type SarifLocation struct {
	PhysicalLocation SarifPhysicalLocation `json:"physicalLocation"`
}

type SarifPhysicalLocation struct {
	ArtifactLocation SarifArtifactLocation `json:"artifactLocation"`
	Region           SarifRegion           `json:"region"`
}

type SarifRegion struct {
	StartLine   uint `json:"startLine"`
	StartColumn uint `json:"startColumn"`
	EndColumn   uint `json:"endColumn"`
}

type SarifArtifactLocation struct {
	URI string `json:"uri"`
}

type SarifMessage struct {
	Text string `json:"text"`
}

type SarifResultFingerprint struct {
	PrimaryLocationLineHash string `json:"primaryLocationLineHash"`
}
