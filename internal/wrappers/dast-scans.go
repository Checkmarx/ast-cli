package wrappers

// DastScansWrapper defines the interface for DAST scans operations
type DastScansWrapper interface {
	// Get retrieves scans for an environment with optional query parameters (from, to, search, sort)
	Get(params map[string]string) (*DastScansCollectionResponseModel, *ErrorModel, error)
}

// DastScansCollectionResponseModel represents the response from the DAST scans API
type DastScansCollectionResponseModel struct {
	Scans      []DastScanResponseModel `json:"scans"`
	TotalScans int                     `json:"totalScans"`
}

// DastScanResponseModel represents a single DAST scan
type DastScanResponseModel struct {
	ScanID            string    `json:"scanId"`
	Initiator         string    `json:"initiator"`
	ScanType          string    `json:"scanType"`
	Created           string    `json:"created"`
	RiskLevel         RiskLevel `json:"riskLevel"`
	RiskRating        string    `json:"riskRating"`
	AlertRiskLevel    RiskLevel `json:"alertRiskLevel"`
	StartTime         string    `json:"startTime"`
	UpdateTime        string    `json:"updateTime"`
	ScanDuration      int       `json:"scanDuration"`
	LastStatus        string    `json:"lastStatus"`
	Statistics        string    `json:"statistics"`
	HasResults        bool      `json:"hasResults"`
	ScannedPathsCount int       `json:"scannedPathsCount"`
	HasLog            bool      `json:"hasLog"`
	Source            string    `json:"source"`
}

