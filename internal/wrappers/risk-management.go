package wrappers

type RiskManagementWrapper interface {
	GetTopVulnerabilitiesByProjectID(projectID string) (*ASPMResult, *WebError, error)
}

type RiskManagementResults struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Hash               string             `json:"hash"`
	Type               string             `json:"type"`
	State              string             `json:"state"`
	Engine             string             `json:"engine"`
	Severity           string             `json:"severity"`
	RiskScore          float64            `json:"riskScore"`
	EnrichmentSources  map[string]string  `json:"enrichmentSources"`
	CreatedAt          string             `json:"createdAt"`
	ApplicationsScores []ApplicationScore `json:"applicationsScores"`
}

type ApplicationScore struct {
	ApplicationID string  `json:"applicationID"`
	Score         float64 `json:"score"`
}

type ASPMResult struct {
	ProjectID            string                  `json:"projectID"`
	ScanID               string                  `json:"scanID"`
	ApplicationNameIDMap map[string]string       `json:"applicationNameIDMap"`
	Results              []RiskManagementResults `json:"results"`
}
