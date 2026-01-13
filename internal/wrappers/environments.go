package wrappers

import "encoding/json"

// EnvironmentsWrapper defines the interface for environments operations
type EnvironmentsWrapper interface {
	// Get retrieves environments with optional query parameters (from, to, search, sort)
	Get(params map[string]string) (*EnvironmentsCollectionResponseModel, *ErrorModel, error)
}

// EnvironmentsCollectionResponseModel represents the response from the environments API
type EnvironmentsCollectionResponseModel struct {
	Environments       []EnvironmentResponseModel `json:"environments"`
	MisconfiguredCount int                        `json:"misconfiguredCount"`
	TotalItems         int                        `json:"totalItems"`
	ZrokHost           string                     `json:"zrokHost"`
}

// EnvironmentResponseModel represents a single environment
type EnvironmentResponseModel struct {
	EnvironmentID   string           `json:"environmentId"`
	TunnelID        string           `json:"tunnelId"`
	Created         string           `json:"created"`
	Domain          string           `json:"domain"`
	URL             string           `json:"url"`
	ScanType        string           `json:"scanType"`
	ProjectIds      []string         `json:"projectIds"`
	Tags            []string         `json:"tags"`
	Groups          []string         `json:"groups"`
	Applications    []EnvironmentApp `json:"applications"`
	RiskLevel       RiskLevel        `json:"riskLevel"`
	RiskRating      string           `json:"riskRating"`
	AlertRiskLevel  RiskLevel        `json:"alertRiskLevel"`
	LastScanID      string           `json:"lastScanID"`
	LastScanTime    string           `json:"lastScanTime"`
	LastStatus      string           `json:"lastStatus"`
	AuthSuccess     bool             `json:"authSuccess"`
	IsPublic        bool             `json:"isPublic"`
	AuthMethod      string           `json:"authMethod"`
	LastAuthUUID    string           `json:"lastAuthUUID"`
	LastAuthSuccess bool             `json:"lastAuthSuccess"`
	Settings        json.RawMessage  `json:"settings"` // Keep as raw JSON
	HasReport       bool             `json:"hasReport"`
	HasAuth         bool             `json:"hasAuth"`
	TunnelState     string           `json:"tunnelState"`
	ScanConfig      json.RawMessage  `json:"scanConfig"` // Keep as raw JSON
}

// EnvironmentApp represents an application associated with an environment
type EnvironmentApp struct {
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
