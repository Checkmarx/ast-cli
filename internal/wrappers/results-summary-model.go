package wrappers

type ResultsSummary struct {
	ScansSummaries *ScanSumaries `json:"scansSummaries"`
	TotalCount     int           `json:"totalCount"`
}

type ScanSumaries struct {
	SastCounters          *SastCounters          `json:"sastCounters"`
	KicsCounters          *KicsCounters          `json:"kicsCounters"`
	ScaCounters           *ScaCounters           `json:"scaCounters"`
	ScaPackagesCounters   *ScaPackagesCounters   `json:"scaPackagesCounters"`
	ScaContainersCounters *ScaContainersCounters `json:"scaContainersCounters"`
	ApiSecCounters        *ApiSecCounters        `json:"apiSecCounters"`
}

// TODO il: refactor this and remove duplicated structs
type SastCounters struct {
	QueriesCounters  []interface{} `json:"queriesCounters"`
	SinkFileCounters []interface{} `json:"sinkFileCounters"`
	LanguageCounters []struct {
		Language string `json:"language"`
		Counter  int    `json:"counter"`
	} `json:"languageCounters"`
	ComplianceCounters []struct {
		Compliance string `json:"compliance"`
		Counter    int    `json:"counter"`
	} `json:"complianceCounters"`
	SeverityCounters []SeverityCounters `json:"severityCounters"`
	StatusCounters   []struct {
		Status  string `json:"status"`
		Counter int    `json:"counter"`
	} `json:"statusCounters"`
	StateCounters []struct {
		State   string `json:"state"`
		Counter int    `json:"counter"`
	} `json:"stateCounters"`
	SeverityStatusCounters []struct {
		Severity string `json:"severity"`
		Status   string `json:"status"`
		Counter  int    `json:"counter"`
	} `json:"severityStatusCounters"`
	SourceFileCounters  []interface{} `json:"sourceFileCounters"`
	AgeCounters         []AgeCounters
	TotalCounter        int `json:"totalCounter"`
	FilesScannedCounter int `json:"filesScannedCounter"`
}
type KicsCounters struct {
	SeverityCounters []SeverityCounters `json:"severityCounters"`
	StatusCounters   []struct {
		Status  string `json:"status"`
		Counter int    `json:"counter"`
	} `json:"statusCounters"`
	StateCounters []struct {
		State   string `json:"state"`
		Counter int    `json:"counter"`
	} `json:"stateCounters"`
	SeverityStatusCounters []struct {
		Severity string `json:"severity"`
		Status   string `json:"status"`
		Counter  int    `json:"counter"`
	} `json:"severityStatusCounters"`
	SourceFileCounters  []interface{} `json:"sourceFileCounters"`
	AgeCounters         []AgeCounters
	TotalCounter        int `json:"totalCounter"`
	FilesScannedCounter int `json:"filesScannedCounter"`
	PlatformSummary     []struct {
		Platform string `json:"platform"`
		Counter  int    `json:"counter"`
	} `json:"platformSummary"`
	CategorySummary []struct {
		Category string `json:"category"`
		Counter  int    `json:"counter"`
	} `json:"categorySummary"`
}

type ScaCounters struct {
	SeverityCounters []SeverityCounters `json:"severityCounters"`
	StatusCounters   []interface{}      `json:"statusCounters"`
	StateCounters    []struct {
		State   string `json:"state"`
		Counter int    `json:"counter"`
	} `json:"stateCounters"`
	SeverityStatusCounters []struct {
		Severity string `json:"severity"`
		Status   string `json:"status"`
		Counter  int    `json:"counter"`
	} `json:"severityStatusCounters"`
	SourceFileCounters  []interface{} `json:"sourceFileCounters"`
	AgeCounters         []AgeCounters
	TotalCounter        int `json:"totalCounter"`
	FilesScannedCounter int `json:"filesScannedCounter"`
}

type ScaPackagesCounters struct {
	SeverityCounters []SeverityCounters `json:"severityCounters"`
	StatusCounters   []interface{}      `json:"statusCounters"`
	StateCounters    []struct {
		State   string `json:"state"`
		Counter int    `json:"counter"`
	} `json:"stateCounters"`
	SeverityStatusCounters []interface{} `json:"severityStatusCounters"`
	SourceFileCounters     []interface{} `json:"sourceFileCounters"`
	AgeCounters            []interface{} `json:"ageCounters"`
	TotalCounter           int           `json:"totalCounter"`
	FilesScannedCounter    int           `json:"filesScannedCounter"`
	OutdatedCounter        int           `json:"outdatedCounter"`
	RiskLevelCounters      []struct {
		RiskLevel string `json:"riskLevel"`
		Counter   int    `json:"counter"`
	} `json:"riskLevelCounters"`
	LicenseCounters []struct {
		License string `json:"license"`
		Counter int    `json:"counter"`
	} `json:"licenseCounters"`
	PackageCounters []struct {
		Package string `json:"package"`
		Counter int    `json:"counter"`
	} `json:"packageCounters"`
}

type ScaContainersCounters struct {
	TotalPackagesCounter            int `json:"totalPackagesCounter"`
	TotalVulnerabilitiesCounter     int `json:"totalVulnerabilitiesCounter"`
	SeverityVulnerabilitiesCounters []struct {
		Severity string `json:"severity"`
		Counter  int    `json:"counter"`
	} `json:"severityVulnerabilitiesCounters"`
	StateVulnerabilitiesCounters  []interface{} `json:"stateVulnerabilitiesCounters"`
	StatusVulnerabilitiesCounters []struct {
		Status  string `json:"status"`
		Counter int    `json:"counter"`
	} `json:"statusVulnerabilitiesCounters"`
	AgeVulnerabilitiesCounters     []AgeCounters `json:"ageVulnerabilitiesCounters"`
	PackageVulnerabilitiesCounters []struct {
		Package string `json:"package"`
		Counter int    `json:"counter"`
	} `json:"packageVulnerabilitiesCounters"`
}

type ApiSecCounters struct {
	SeverityCounters       *[]SeverityCounters `json:"severityCounters"`
	StatusCounters         []interface{}       `json:"statusCounters"`
	StateCounters          []interface{}       `json:"stateCounters"`
	SeverityStatusCounters []interface{}       `json:"severityStatusCounters"`
	SourceFileCounters     []interface{}       `json:"sourceFileCounters"`
	AgeCounters            []interface{}       `json:"ageCounters"`
	TotalCounter           int                 `json:"totalCounter"`
	FilesScannedCounter    int                 `json:"filesScannedCounter"`
	ApiSecTotal            int                 `json:"apiSecTotal"`
}

type SeverityCounters struct {
	Severity string `json:"severity"`
	Counter  int    `json:"counter"`
}

type AgeCounters struct {
	Age              string             `json:"age"`
	SeverityCounters []SeverityCounters `json:"severityCounters"`
	Counter          int                `json:"counter"`
}
