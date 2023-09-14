package wrappers

type ScanSummariesModel struct {
	ScansSummaries []ScanSumaries `json:"scansSummaries,omitempty,"`
	TotalCount     int            `json:"totalCount,omitempty,"`
}

type ScanSumaries struct {
	SastCounters          SastCounters          `json:"sastCounters,omitempty,"`
	KicsCounters          KicsCounters          `json:"kicsCounters,omitempty,"`
	ScaCounters           ScaCounters           `json:"scaCounters,omitempty,"`
	ScaPackagesCounters   ScaPackagesCounters   `json:"scaPackagesCounters,omitempty,"`
	ScaContainersCounters ScaContainersCounters `json:"scaContainersCounters,omitempty,"`
	ApiSecCounters        ApiSecCounters        `json:"apiSecCounters,omitempty,"`
}

type SastCounters struct {
	QueriesCounters        []interface{}            `json:"queriesCounters,omitempty,"`
	SinkFileCounters       []interface{}            `json:"sinkFileCounters,omitempty,"`
	LanguageCounters       []languageCounters       `json:"languageCounters,omitempty,"`
	ComplianceCounters     []complianceCounters     `json:"complianceCounters,omitempty,"`
	SeverityCounters       []severityCounters       `json:"severityCounters,omitempty,"`
	StatusCounters         []statusCounters         `json:"statusCounters,omitempty,"`
	StateCounters          []stateCounters          `json:"stateCounters,omitempty,"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters,omitempty,"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters,omitempty,"`
	AgeCounters            []ageCounters            `json:"ageCounters,omitempty,"`
	TotalCounter           int                      `json:"totalCounter,omitempty,"`
	FilesScannedCounter    int                      `json:"filesScannedCounter,omitempty,"`
}
type KicsCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters,omitempty,"`
	StatusCounters         []statusCounters         `json:"statusCounters,omitempty,"`
	StateCounters          []stateCounters          `json:"stateCounters,omitempty,"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters,omitempty,"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters,omitempty,"`
	AgeCounters            []ageCounters            `json:"ageCounters,omitempty,"`
	TotalCounter           int                      `json:"totalCounter,omitempty,"`
	FilesScannedCounter    int                      `json:"filesScannedCounter,omitempty,"`
	PlatformSummary        []plataformSummary       `json:"platformSummary,omitempty,"`
	CategorySummary        []categorySummary        `json:"categorySummary,omitempty,"`
}

type ScaCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters,omitempty,"`
	StatusCounters         []interface{}            `json:"statusCounters,omitempty,"`
	StateCounters          []stateCounters          `json:"stateCounters,omitempty,"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters,omitempty,"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters,omitempty,"`
	AgeCounters            []ageCounters            `json:"ageCounters,omitempty,"`
	TotalCounter           int                      `json:"totalCounter,omitempty,"`
	FilesScannedCounter    int                      `json:"filesScannedCounter,omitempty,"`
}

type ScaPackagesCounters struct {
	SeverityCounters       []severityCounters  `json:"severityCounters,omitempty,"`
	StatusCounters         []interface{}       `json:"statusCounters,omitempty,"`
	StateCounters          []stateCounters     `json:"stateCounters,omitempty,"`
	SeverityStatusCounters []interface{}       `json:"severityStatusCounters,omitempty,"`
	SourceFileCounters     []interface{}       `json:"sourceFileCounters,omitempty,"`
	AgeCounters            []interface{}       `json:"ageCounters,omitempty,"`
	TotalCounter           int                 `json:"totalCounter,omitempty,"`
	FilesScannedCounter    int                 `json:"filesScannedCounter,omitempty,"`
	OutdatedCounter        int                 `json:"outdatedCounter,omitempty,"`
	RiskLevelCounters      []riskLevelCounters `json:"riskLevelCounters,omitempty,"`
	LicenseCounters        []lincenseCounters  `json:"licenseCounters,omitempty,"`
	PackageCounters        []packageCounters   `json:"packageCounters,omitempty,"`
}

type ScaContainersCounters struct {
	TotalPackagesCounter        int                `json:"totalPackagesCounter,omitempty,"`
	TotalVulnerabilitiesCounter int                `json:"totalVulnerabilitiesCounter,omitempty,"`
	SeverityCounters            []severityCounters `json:"severityVulnerabilitiesCounters,omitempty,"`
	StateCounters               []stateCounters    `json:"stateVulnerabilitiesCounters,omitempty,"`
	StatusCounters              []statusCounters   `json:"statusVulnerabilitiesCounters,omitempty,"`
	AgeCounters                 []ageCounters      `json:"ageVulnerabilitiesCounters,omitempty,"`
	PackageCounters             []packageCounters  `json:"packageVulnerabilitiesCounters,omitempty,"`
}

type ApiSecCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters,omitempty,"`
	StatusCounters         []statusCounters         `json:"statusCounters,omitempty,"`
	StateCounters          []stateCounters          `json:"stateCounters,omitempty,"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters,omitempty,"`
	SourceFileCounters     interface{}              `json:"sourceFileCounters,omitempty,"`
	AgeCounters            []ageCounters            `json:"ageCounters,omitempty,"`
	TotalCounter           int                      `json:"totalCounter,omitempty,"`
	FilesScannedCounter    int                      `json:"filesScannedCounter,omitempty,"`
	ApiSecTotal            int                      `json:"apiSecTotal,omitempty,"`
}
type categorySummary struct {
	Category string `json:"category,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}
type lincenseCounters struct {
	License string `json:"license,omitempty,"`
	Counter int    `json:"counter,omitempty,"`
}
type plataformSummary struct {
	Platform string `json:"platform,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}
type severityStatusCounters struct {
	Severity string `json:"severity,omitempty,"`
	Status   string `json:"status,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}
type languageCounters struct {
	Language string `json:"language,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}
type riskLevelCounters struct {
	RiskLevel string `json:"riskLevel,omitempty,"`
	Counter   int    `json:"counter,omitempty,"`
}
type severityCounters struct {
	Severity string `json:"severity,omitempty,"`
	Counter  int    `json:"counter,omitempty,"`
}

type ageCounters struct {
	Age              string             `json:"age,omitempty,"`
	SeverityCounters []severityCounters `json:"severityCounters,omitempty,"`
	Counter          int                `json:"counter,omitempty,"`
}

type complianceCounters struct {
	Compliance string `json:"compliance,omitempty,"`
	Counter    int    `json:"counter,omitempty,"`
}
type statusCounters struct {
	Status  string `json:"status,omitempty,"`
	Counter int    `json:"counter,omitempty,"`
}
type packageCounters struct {
	Package string `json:"package,omitempty,"`
	Counter int    `json:"counter,omitempty,"`
}
type stateCounters struct {
	State   string `json:"state,omitempty,"`
	Counter int    `json:"counter,omitempty,"`
}
