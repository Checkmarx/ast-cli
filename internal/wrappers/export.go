package wrappers

type ExportWrapper interface {
	InitiateExportRequest(payload *ExportRequestPayload) (*ExportResponse, error)
	GetExportReportStatus(reportID string) (*ExportPollingResponse, error)
	DownloadExportReport(reportID, targetFile string) error
	GetScaPackageCollectionExport(fileURL string) (*ScaPackageCollectionExport, error)
}

type ScaPackageCollectionExport struct {
	Packages []ScaPackage `json:"Packages,omitempty"`
	ScaTypes []ScaType    `json:"Vulnerabilities,omitempty"`
}

type ScaPackage struct {
	ID                      string          `json:"Id,omitempty"`
	Name                    string          `json:"Name,omitempty"`
	Locations               []*string       `json:"Locations,omitempty"`
	PackagePathArray        [][]PackagePath `json:"PackagePaths,omitempty"`
	Outdated                bool            `json:"Outdated,omitempty"`
	IsDirectDependency      bool            `json:"IsDirectDependency"`
	IsDevelopmentDependency bool            `json:"IsDevelopmentDependency"`
	SupportsQuickFix        bool
	FixLink                 string
	TypeOfDependency        string
}

type PackagePath struct {
	ID               string `json:"Id,omitempty"`
	Name             string `json:"Name,omitempty"`
	Version          string `json:"Version,omitempty"`
	IsResolved       bool   `json:"IsResolved,omitempty"`
	Locations        []*string
	SupportsQuickFix bool
}

type ScaType struct {
	ID        string `json:"Id,omitempty"`
	Type      string `json:"Type,omitempty"`
	IsIgnored bool   `json:"IsIgnored,omitempty"`
}
