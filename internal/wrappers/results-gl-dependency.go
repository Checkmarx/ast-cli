package wrappers

const (
	AnalyzerScaName  = "CxOne"
	AnalyzerScaID    = AnalyzerScaName + "-SCA"
	ScannerID        = "SCA"
	ScannerType      = "dependency_scanning"
	DependencySchema = "https://gitlab.com/gitlab-org/gitlab/-/raw/master/lib/gitlab/ci/parsers/security/validators/schemas/15.0.0/sast-report-format.json"
	SchemaVersion    = "15.0.0"
)

type GlDependencyResultsCollection struct {
	Scan            ScanGlDepReport        `json:"scan"`
	Schema          string                 `json:"schema"`
	Version         string                 `json:"version"`
	Vulnerabilities []GlDepVulnerabilities `json:"vulnerabilities"`
	DependencyFiles []DependencyFile       `json:"dependency_files"`
}

type GlVendor struct {
	VendorGlname string `json:"name"`
}

type ScanGlDepReport struct {
	EndTime   string        `json:"end_time,omitempty"`
	Analyzer  GlDepAnalyzer `json:"analyzer,omitempty"`
	Scanner   GlDepScanner  `json:"scanner,omitempty"`
	StartTime string        `json:"start_time,omitempty"`
	Status    string        `json:"status,omitempty"`
	Type      string        `json:"type"`
}

type GlDepAnalyzer struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	VendorGlSCA  GlVendor `json:"vendor"`
	VersionGlSca string   `json:"version,omitempty"`
}

type GlDepScanner struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	VersionGlSca string   `json:"version,omitempty"`
	VendorGlSCA  GlVendor `json:"vendor"`
}

type GlDepVulnerabilities struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Severity    string                     `json:"severity"`
	Solution    interface{}                `json:"solution"`
	Identifiers []IdentifierDep            `json:"identifiers"`
	Links       []LinkDep                  `json:"links"`
	TrackingDep TrackingDep                `json:"tracking"`
	Flags       []string                   `json:"flags"`
	LocationDep GlDepVulnerabilityLocation `json:"location,omitempty"`
}

type IdentifierDep struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type LinkDep struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type TrackingDep struct {
	Items []ItemDep `json:"items"`
}

type ItemDep struct {
	Signature []SignatureDep `json:"signatures"`
	File      string         `json:"file"`
	EndLine   uint           `json:"end_line"`
	StartLine uint           `json:"start_line"`
}

type SignatureDep struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type GlDepVulnerabilityLocation struct {
	File       string             `json:"file"`
	Dependency DependencyLocation `json:"dependency"`
}

type DependencyLocation struct {
	Package                   PackageName `json:"package"`
	DependencyLocationVersion string      `json:"version"`
	Iid                       string      `json:"iid"`
	Direct                    bool        `json:"direct"`
	DependencyPath            string      `json:"iid"`
}

type PackageName struct {
	Name string `json:"name"`
}

type DependencyFile struct {
	Path           string               `json:"path"`
	PackageManager string               `json:"package_manager"`
	Dependencies   []DependencyLocation `json:"dependencies"`
}
