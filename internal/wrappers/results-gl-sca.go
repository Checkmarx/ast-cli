package wrappers

const (
	AnalyzerScaName = "CxOne"
	AnalyzerScaID   = AnalyzerScaName + "-SCA"
	ScannerID       = "SCA"
	ScannerType     = "dependency_scanning"
	ScaSchema       = "https://gitlab.com/gitlab-org/security-products/security-report-schemas/-/blob/master/dist/dependency-scanning-report-format.json"
	SchemaVersion   = "15.0.0"
)

type GlScaResultsCollection struct {
	Scan               ScanGlScaDepReport        `json:"scan"`
	Schema             string                    `json:"schema"`
	Version            string                    `json:"version"`
	Vulnerabilities    []GlScaDepVulnerabilities `json:"vulnerabilities"`
	ScaDependencyFiles []ScaDependencyFile       `json:"dependency_files"`
}

type GlVendor struct {
	VendorGlname string `json:"name"`
}

type ScanGlScaDepReport struct {
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

type GlScaDepVulnerabilities struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Severity    string                        `json:"severity"`
	Solution    interface{}                   `json:"solution"`
	Identifiers []IdentifierDep               `json:"identifiers,omitempty"`
	Links       []LinkDep                     `json:"links"`
	TrackingDep TrackingDep                   `json:"tracking"`
	Flags       []string                      `json:"flags"`
	LocationDep GlScaDepVulnerabilityLocation `json:"location,omitempty"`
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

type GlScaDepVulnerabilityLocation struct {
	File       string                `json:"file"`
	Dependency ScaDependencyLocation `json:"dependency"`
}

type ScaDependencyLocation struct {
	Package                      PackageName `json:"package"`
	ScaDependencyLocationVersion string      `json:"version"`
	Direct                       bool        `json:"direct"`
	ScaDependencyPath            uint        `json:"iid,omitempty"`
}

type PackageName struct {
	Name string `json:"name"`
}

type ScaDependencyFile struct {
	Path           string                  `json:"path"`
	PackageManager string                  `json:"package_manager"`
	Dependencies   []ScaDependencyLocation `json:"dependencies"`
}
