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
	QueriesCounters        []interface{}            `json:"queriesCounters"`
	SinkFileCounters       []interface{}            `json:"sinkFileCounters"`
	LanguageCounters       []languageCounters       `json:"languageCounters"`
	ComplianceCounters     []complianceCounters     `json:"complianceCounters"`
	SeverityCounters       []severityCounters       `json:"severityCounters"`
	StatusCounters         []statusCounters         `json:"statusCounters"`
	StateCounters          []stateCounters          `json:"stateCounters"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters"`
	AgeCounters            []ageCounters            `json:"ageCounters"`
	TotalCounter           int                      `json:"totalCounter"`
	FilesScannedCounter    int                      `json:"filesScannedCounter"`
}
type KicsCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters"`
	StatusCounters         []statusCounters         `json:"statusCounters"`
	StateCounters          []stateCounters          `json:"stateCounters"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters"`
	AgeCounters            []ageCounters            `json:"ageCounters"`
	TotalCounter           int                      `json:"totalCounter"`
	FilesScannedCounter    int                      `json:"filesScannedCounter"`
	PlatformSummary        []plataformSummary       `json:"platformSummary"`
	CategorySummary        []categorySummary        `json:"categorySummary"`
}

type ScaCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters"`
	StatusCounters         []interface{}            `json:"statusCounters"`
	StateCounters          []stateCounters          `json:"stateCounters"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters"`
	SourceFileCounters     []interface{}            `json:"sourceFileCounters"`
	AgeCounters            []ageCounters            `json:"ageCounters"`
	TotalCounter           int                      `json:"totalCounter"`
	FilesScannedCounter    int                      `json:"filesScannedCounter"`
}

type ScaPackagesCounters struct {
	SeverityCounters       []severityCounters  `json:"severityCounters"`
	StatusCounters         []interface{}       `json:"statusCounters"`
	StateCounters          []stateCounters     `json:"stateCounters"`
	SeverityStatusCounters []interface{}       `json:"severityStatusCounters"`
	SourceFileCounters     []interface{}       `json:"sourceFileCounters"`
	AgeCounters            []interface{}       `json:"ageCounters"`
	TotalCounter           int                 `json:"totalCounter"`
	FilesScannedCounter    int                 `json:"filesScannedCounter"`
	OutdatedCounter        int                 `json:"outdatedCounter"`
	RiskLevelCounters      []riskLevelCounters `json:"riskLevelCounters"`
	LicenseCounters        []lincenseCounters  `json:"licenseCounters"`
	PackageCounters        []packageCounters   `json:"packageCounters"`
}

type ScaContainersCounters struct {
	TotalPackagesCounter        int                `json:"totalPackagesCounter"`
	TotalVulnerabilitiesCounter int                `json:"totalVulnerabilitiesCounter"`
	SeverityCounters            []severityCounters `json:"severityVulnerabilitiesCounters"`
	StateCounters               []stateCounters    `json:"stateVulnerabilitiesCounters"`
	StatusCounters              []statusCounters   `json:"statusVulnerabilitiesCounters"`
	AgeCounters                 []ageCounters      `json:"ageVulnerabilitiesCounters"`
	PackageCounters             []packageCounters  `json:"packageVulnerabilitiesCounters"`
}

type ApiSecCounters struct {
	SeverityCounters       []severityCounters       `json:"severityCounters"`
	StatusCounters         []statusCounters         `json:"statusCounters"`
	StateCounters          []stateCounters          `json:"stateCounters"`
	SeverityStatusCounters []severityStatusCounters `json:"severityStatusCounters"`
	SourceFileCounters     interface{}              `json:"sourceFileCounters"`
	AgeCounters            []ageCounters            `json:"ageCounters"`
	TotalCounter           int                      `json:"totalCounter"`
	FilesScannedCounter    int                      `json:"filesScannedCounter"`
	ApiSecTotal            int                      `json:"apiSecTotal"`
}
type categorySummary struct {
	Category string `json:"category"`
	Counter  int    `json:"counter"`
}
type lincenseCounters struct {
	License string `json:"license"`
	Counter int    `json:"counter"`
}
type plataformSummary struct {
	Platform string `json:"platform"`
	Counter  int    `json:"counter"`
}
type severityStatusCounters struct {
	Severity string `json:"severity"`
	Status   string `json:"status"`
	Counter  int    `json:"counter"`
}
type languageCounters struct {
	Language string `json:"language"`
	Counter  int    `json:"counter"`
}
type riskLevelCounters struct {
	RiskLevel string `json:"riskLevel"`
	Counter   int    `json:"counter"`
}
type severityCounters struct {
	Severity string `json:"severity"`
	Counter  int    `json:"counter"`
}

type ageCounters struct {
	Age              string             `json:"age"`
	SeverityCounters []severityCounters `json:"severityCounters"`
	Counter          int                `json:"counter"`
}

type complianceCounters struct {
	Compliance string `json:"compliance"`
	Counter    int    `json:"counter"`
}
type statusCounters struct {
	Status  string `json:"status"`
	Counter int    `json:"counter"`
}
type packageCounters struct {
	Package string `json:"package"`
	Counter int    `json:"counter"`
}
type stateCounters struct {
	State   string `json:"state"`
	Counter int    `json:"counter"`
}
