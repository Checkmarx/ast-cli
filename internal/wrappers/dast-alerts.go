package wrappers

// DastAlertsWrapper defines the interface for DAST alerts operations
type DastAlertsWrapper interface {
	// GetAlerts retrieves alerts for a specific environment and scan
	GetAlerts(environmentID, scanID string, params map[string]string) (*DastAlertsCollectionResponseModel, *ErrorModel, error)
}

// DastAlertsCollectionResponseModel represents the response from the DAST alerts API
type DastAlertsCollectionResponseModel struct {
	PagesNumber int                      `json:"pages_number"`
	Results     []DastAlertResponseModel `json:"results"`
	Total       int                      `json:"total"`
}

// DastAlertResponseModel represents a single DAST alert
type DastAlertResponseModel struct {
	AlertSimilarityID string   `json:"alert_similarity_id"`
	State             string   `json:"state"`
	Severity          string   `json:"severity"`
	Name              string   `json:"name"`
	NumInstances      int      `json:"num_instances"`
	Status            string   `json:"status"`
	OWASP             []string `json:"owasp"`
	NumNotes          int      `json:"num_notes"`
	Systemic          bool     `json:"systemic"`
}

