package wrappers

import "time"

const (
	SuspMalwareKey = "suspMalware"
	ExplPathKey    = "explPath"
	PubExposedKey  = "pubExposed"
	RuntimeKey     = "runtime"

	SuspMalwareValue = "Suspected Malware"
	ExplPathValue    = "Exploitable Path"
	PubExposedValue  = "Public Exposed"
	RuntimeValue     = "Runtime"
)

type RiskManagementWrapper interface {
	GetTopVulnerabilitiesByProjectID(projectID string) (*ASPMResult, *WebError, error)
}

type ApplicationScore struct {
	ApplicationID string  `json:"applicationID"`
	Score         float64 `json:"score"`
}

type RiskManagementApplication struct {
	ApplicationID   string  `json:"applicationID"`
	ApplicationName string  `json:"applicationName"`
	Score           float64 `json:"score"`
}

type RiskManagementResult struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Hash               string             `json:"hash"`
	Type               string             `json:"type"`
	State              string             `json:"state"`
	Engine             string             `json:"engine"`
	Severity           string             `json:"severity"`
	RiskScore          float64            `json:"riskScore"`
	EnrichmentSources  map[string]string  `json:"enrichmentSources"`
	Traits             map[string]string  `json:"traits"`
	CreatedAt          time.Time          `json:"createdAt"`
	ApplicationsScores []ApplicationScore `json:"applicationsScores"`
}

type ASPMResult struct {
	ProjectID            string                      `json:"projectID"`
	ScanID               string                      `json:"scanID"`
	ApplicationNameIDMap []RiskManagementApplication `json:"applicationNameIDMap"`
	Results              []RiskManagementResult      `json:"results"`
}
