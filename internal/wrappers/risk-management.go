package wrappers

import "time"

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

type EnrichmentSource struct {
	CNAS               string `json:"CNAS"`
	CorrelationSastSca string `json:"Correlation_SastSca"`
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
	EnrichmentSources  EnrichmentSource   `json:"enrichmentSources"`
	CreatedAt          time.Time          `json:"createdAt"`
	ApplicationsScores []ApplicationScore `json:"applicationsScores"`
}

type ASPMResult struct {
	ProjectID            string                      `json:"projectID"`
	ScanID               string                      `json:"scanID"`
	ApplicationNameIDMap []RiskManagementApplication `json:"applicationNameIDMap"`
	Results              []RiskManagementResult      `json:"results"`
}
