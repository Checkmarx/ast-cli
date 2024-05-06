package wrappers

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *WebError, error)
	GetAllResultsPackageByScanID(params map[string]string) (*[]ScaPackageCollection, *WebError, error)
	GetAllResultsTypeByScanID(params map[string]string) (*[]ScaTypeCollection, *WebError, error)
	GetResultsURL(projectID string) (string, error)
}

// ScanSummariesModel model used to parse the response from the scan-summary API
type ScanSummariesModel struct {
	ScansSummaries []ScanSumaries `json:"scansSummaries,omitempty,"`
	TotalCount     int            `json:"totalCount,omitempty,"`
}

type ScanSumaries struct {
	SastCounters          SastCounters          `json:"sastCounters,omitempty,"`
	KicsCounters          KicsCounters          `json:"kicsCounters,omitempty,"`
	ScaCounters           ScaCounters           `json:"scaCounters,omitempty,"`
	ScaContainersCounters ScaContainersCounters `json:"scaContainersCounters,omitempty,"`
}

type SastCounters struct {
	SeverityCounters    []SeverityCounters `json:"SeverityCounters,omitempty,"`
	TotalCounter        int                `json:"totalCounter,omitempty,"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty,"`
}
type KicsCounters struct {
	SeverityCounters    []SeverityCounters `json:"SeverityCounters,omitempty,"`
	TotalCounter        int                `json:"totalCounter,omitempty,"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty,"`
}

type ScaCounters struct {
	SeverityCounters    []SeverityCounters `json:"SeverityCounters,omitempty,"`
	TotalCounter        int                `json:"totalCounter,omitempty,"`
	FilesScannedCounter int                `json:"filesScannedCounter,omitempty,"`
}

type ScaContainersCounters struct {
	SeverityCounters            []SeverityCounters `json:"severityVulnerabilitiesCounters,omitempty,"`
	TotalPackagesCounter        int                `json:"totalPackagesCounter,omitempty,"`
	TotalVulnerabilitiesCounter int                `json:"totalVulnerabilitiesCounter,omitempty,"`
}

type SeverityCounters struct {
	Severity string `json:"severity,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}
