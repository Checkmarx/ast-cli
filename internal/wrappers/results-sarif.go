package wrappers

var (
	SarifName           = "Checkmarx One"
	SarifVersion        = "1.0"
	SarifInformationURI = "https://checkmarx.com/resource/documents/en/34965-67042-checkmarx-one.html"
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
	ID              string           `json:"id"`
	Name            string           `json:"name,omitempty"`
	HelpURI         string           `json:"helpUri"`
	Help            SarifHelp        `json:"help"`
	FullDescription SarifDescription `json:"fullDescription"`
	Properties      SarifProperties  `json:"properties,omitempty"`
}

type SarifProperties struct {
	SecuritySeverity string   `json:"security-severity"`
	Name             string   `json:"name"`
	ID               string   `json:"id"`
	Description      string   `json:"description"`
	Tags             []string `json:"tags"`
}

type SarifHelp struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}
type SarifDescription struct {
	Text string `json:"text"`
}

type SarifScanResult struct {
	RuleID              string                  `json:"ruleId"`
	Level               string                  `json:"level"`
	Message             SarifMessage            `json:"message"`
	PartialFingerprints *SarifResultFingerprint `json:"partialFingerprints,omitempty"`
	Locations           []SarifLocation         `json:"locations,omitempty"`
	Properties          *SarifResultProperties  `json:"properties,omitempty"`
}

type SarifLocation struct {
	PhysicalLocation SarifPhysicalLocation `json:"physicalLocation"`
}

type SarifPhysicalLocation struct {
	ArtifactLocation SarifArtifactLocation `json:"artifactLocation"`
	Region           *SarifRegion          `json:"region,omitempty"`
}

type SarifRegion struct {
	StartLine   uint          `json:"startLine,omitempty"`
	StartColumn uint          `json:"startColumn,omitempty"`
	EndColumn   uint          `json:"endColumn,omitempty"`
	Snippet     *SarifSnippet `json:"snippet,omitempty"`
}

type SarifSnippet struct {
	Text string `json:"text,omitempty"`
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

type SarifResultProperties struct {
	Severity string `json:"severity,omitempty"`
	Validity string `json:"validity,omitempty"`
}
