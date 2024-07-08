package wrappers

// ExportWrapper is an interface for the export wrapper
type ExportWrapper interface {
	GetExportPackage(scanID string) (*ScaPackageCollectionExport, error)
}

type ScaPackageCollectionExport struct {
	Packages []ScaPackage `json:"Packages,omitempty"`
	ScaTypes []ScaType    `json:"Vulnerabilities,omitempty"`
}

type ScaPackage struct {
	ID                 string          `json:"Id,omitempty"`
	Locations          []*string       `json:"Locations,omitempty"`
	PackagePathArray   [][]PackagePath `json:"PackagePaths,omitempty"`
	Outdated           bool            `json:"Outdated,omitempty"`
	IsDirectDependency bool            `json:"IsDirectDependency"`
	SupportsQuickFix   bool
	FixLink            string
	TypeOfDependency   string
}

type PackagePath struct {
	Name             string `json:"Name,omitempty"`
	Version          string `json:"Version,omitempty"`
	Locations        []*string
	SupportsQuickFix bool
}

type ScaType struct {
	ID        string `json:"Id,omitempty"`
	Type      string `json:"Type,omitempty"`
	IsIgnored bool   `json:"IsIgnored,omitempty"`
}

type RequestPayload struct {
	ScanID     string `json:"ScanId"`
	FileFormat string `json:"FileFormat"`
}

type ExportResponse struct {
	ExportID string `json:"exportId"`
}

type ExportStatusResponse struct {
	ExportID     string `json:"exportId"`
	ExportStatus string `json:"exportStatus"`
	FileURL      string `json:"fileUrl"`
	ErrorMessage string `json:"errorMessage"`
}
