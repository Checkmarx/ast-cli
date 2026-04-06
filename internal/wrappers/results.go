package wrappers

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *WebError, error)
	GetResultsURL(projectID string) (string, error)
}

// ScanSummariesModel model used to parse the response from the scan-summary API
type ScanSummariesModel struct {
	ScansSummaries []ScanSumaries `json:"scansSummaries,omitempty"`
	TotalCount     int            `json:"totalCount,omitempty"`
}

type ScanSumaries struct {
	TenantID              string                `json:"tenantId,omitempty"`
	ScanID                string                `json:"scanId,omitempty"`
	SastCounters          SastCounters          `json:"sastCounters,omitempty"`
	KicsCounters          KicsCounters          `json:"kicsCounters,omitempty"`
	ScaCounters           ScaCounters           `json:"scaCounters,omitempty"`
	ScaPackagesCounters   ScaPackagesCounters   `json:"scaPackagesCounters,omitempty"`
	ScaContainersCounters ScaContainersCounters `json:"scaContainersCounters,omitempty"`
	ApiSecCounters        ApiSecCounters        `json:"apiSecCounters,omitempty"`
	MicroEnginesCounters  MicroEnginesCounters  `json:"microEnginesCounters,omitempty"`
	ContainersCounters    ContainersCounters    `json:"containersCounters,omitempty"`
	AiscCounters          AiscCounters          `json:"aiscCounters,omitempty"`
}

type SastCounters struct {
	SeverityCounters    []SeverityCounters `json:"severityCounters,omitempty"`
	TotalCounter        int                `json:"totalCounter,omitempty"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty"`
}

type KicsCounters struct {
	SeverityCounters    []SeverityCounters `json:"severityCounters,omitempty"`
	TotalCounter        int                `json:"totalCounter,omitempty"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty"`
}

type ScaCounters struct {
	SeverityCounters    []SeverityCounters `json:"severityCounters,omitempty"`
	TotalCounter        int                `json:"totalCounter,omitempty"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty"`
}

// ScaPackagesCounters contains counters for SCA packages.
type ScaPackagesCounters struct {
	SeverityCounters  []SeverityCounters  `json:"severityCounters,omitempty"`
	StatusCounters    []StatusCounters    `json:"statusCounters,omitempty"`
	StateCounters     []StateCounters     `json:"stateCounters,omitempty"`
	TotalCounter      int                 `json:"totalCounter,omitempty"`
	OutdatedCounter   int                 `json:"outdatedCounter,omitempty"`
	RiskLevelCounters []RiskLevelCounters `json:"riskLevelCounters,omitempty"`
	LicenseCounters   []LicenseCounters   `json:"licenseCounters,omitempty"`
}

type ScaContainersCounters struct {
	SeverityCounters            []SeverityCounters `json:"severityVulnerabilitiesCounters,omitempty"`
	TotalPackagesCounter        int                `json:"totalPackagesCounter,omitempty"`
	TotalVulnerabilitiesCounter int                `json:"totalVulnerabilitiesCounter,omitempty"`
}

// ApiSecCounters contains counters for API Security findings.
type ApiSecCounters struct {
	SeverityCounters    []SeverityCounters `json:"severityCounters,omitempty"`
	StateCounters       []StateCounters    `json:"stateCounters,omitempty"`
	TotalCounter        int                `json:"totalCounter,omitempty"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty"`
	RiskLevel           string             `json:"riskLevel,omitempty"`
	ApiSecTotal         int                `json:"apiSecTotal,omitempty"`
}

// MicroEnginesCounters contains counters for micro engines.
type MicroEnginesCounters struct {
	SeverityCounters    []SeverityCounters `json:"severityCounters,omitempty"`
	StatusCounters      []StatusCounters   `json:"statusCounters,omitempty"`
	StateCounters       []StateCounters    `json:"stateCounters,omitempty"`
	TotalCounter        int                `json:"totalCounter,omitempty"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty"`
}

// ContainersCounters contains counters for container scanning results.
type ContainersCounters struct {
	TotalPackagesCounter   int                      `json:"totalPackagesCounter,omitempty"`
	TotalCounter           int                      `json:"totalCounter,omitempty"`
	SeverityCounters       []SeverityCounters       `json:"severityCounters,omitempty"`
	StatusCounters         []StatusCounters         `json:"statusCounters,omitempty"`
	StateCounters          []StateCounters          `json:"stateCounters,omitempty"`
	AgeCounters            []AgeCounters            `json:"ageCounters,omitempty"`
	PackageCounters        []PackageCounters        `json:"packageCounters,omitempty"`
	SeverityStatusCounters []SeverityStatusCounters `json:"severityStatusCounters,omitempty"`
}

// AiscCounters contains counters for AISC engine scanning.
type AiscCounters struct {
	AssetsCounter     int `json:"assetsCounter,omitempty"`
	AssetTypesCounter int `json:"assetTypesCounter,omitempty"`
}

// SeverityCounters contains severity level counter information.
type SeverityCounters struct {
	Severity string `json:"severity,omitempty"`
	Counter  int    `json:"counter,omitempty"`
}

// StatusCounters contains status counter information.
type StatusCounters struct {
	Status  string `json:"status,omitempty"`
	Counter int    `json:"counter,omitempty"`
}

// StateCounters contains state counter information.
type StateCounters struct {
	State   string `json:"state,omitempty"`
	Counter int    `json:"counter,omitempty"`
}

// RiskLevelCounters contains risk level counter information.
type RiskLevelCounters struct {
	RiskLevel string `json:"riskLevel,omitempty"`
	Counter   int    `json:"counter,omitempty"`
}

// LicenseCounters contains license counter information.
type LicenseCounters struct {
	License string `json:"license,omitempty"`
	Counter int    `json:"counter,omitempty"`
}

// AgeCounters contains age counter information.
type AgeCounters struct {
	Age              string             `json:"age,omitempty"`
	Counter          int                `json:"counter,omitempty"`
	SeverityCounters []SeverityCounters `json:"severityCounters,omitempty"`
}

// PackageCounters contains package counter information.
type PackageCounters struct {
	PackageID   string `json:"packageId,omitempty"`
	Counter     int    `json:"counter,omitempty"`
	IsMalicious bool   `json:"isMalicious,omitempty"`
}

// SeverityStatusCounters contains combined severity and status counter information.
type SeverityStatusCounters struct {
	Severity string `json:"severity,omitempty"`
	Status   string `json:"status,omitempty"`
	Counter  int    `json:"counter,omitempty"`
}
