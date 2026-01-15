package wrappers

import "encoding/json"

// DastEnvironmentsWrapper defines the interface for DAST environments operations
type DastEnvironmentsWrapper interface {
	// Get retrieves environments with optional query parameters (from, to, search, sort)
	Get(params map[string]string) (*DastEnvironmentsCollectionResponseModel, *ErrorModel, error)
}

// DastEnvironmentsCollectionResponseModel represents the response from the DAST environments API
type DastEnvironmentsCollectionResponseModel struct {
	Environments       []DastEnvironmentResponseModel `json:"environments"`
	MisconfiguredCount int                            `json:"misconfiguredCount"`
	TotalItems         int                            `json:"totalItems"`
	ZrokHost           string                         `json:"zrokHost"`
}

// DastEnvironmentResponseModel represents a single DAST environment
type DastEnvironmentResponseModel struct {
	EnvironmentID   string               `json:"environmentId"`
	TunnelID        string               `json:"tunnelId"`
	Created         string               `json:"created"`
	Domain          string               `json:"domain"`
	URL             string               `json:"url"`
	ScanType        string               `json:"scanType"`
	ProjectIds      []string             `json:"projectIds"`
	Tags            []string             `json:"tags"`
	Groups          []string             `json:"groups"`
	Applications    []DastEnvironmentApp `json:"applications"`
	RiskLevel       RiskLevel            `json:"riskLevel"`
	RiskRating      string               `json:"riskRating"`
	AlertRiskLevel  RiskLevel            `json:"alertRiskLevel"`
	LastScanID      string               `json:"lastScanID"`
	LastScanTime    string               `json:"lastScanTime"`
	LastStatus      string               `json:"lastStatus"`
	AuthSuccess     bool                 `json:"authSuccess"`
	IsPublic        bool                 `json:"isPublic"`
	AuthMethod      string               `json:"authMethod"`
	LastAuthUUID    string               `json:"lastAuthUUID"`
	LastAuthSuccess bool                 `json:"lastAuthSuccess"`
	Settings        json.RawMessage      `json:"settings"` // Keep as raw JSON
	HasReport       bool                 `json:"hasReport"`
	HasAuth         bool                 `json:"hasAuth"`
	TunnelState     string               `json:"tunnelState"`
	ScanConfig      json.RawMessage      `json:"scanConfig"` // Keep as raw JSON
}

// DastEnvironmentApp represents an application associated with a DAST environment
type DastEnvironmentApp struct {
	ApplicationID string `json:"applicationId"`
	IsPrimary     bool   `json:"isPrimary"`
}

// RiskLevel represents risk counts by severity
type RiskLevel struct {
	CriticalCount int `json:"criticalCount"`
	HighCount     int `json:"highCount"`
	MediumCount   int `json:"mediumCount"`
	LowCount      int `json:"lowCount"`
	InfoCount     int `json:"infoCount"`
}
