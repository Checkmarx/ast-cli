package ossrealtime

import "github.com/checkmarx/ast-cli/internal/services/realtimeengine"

// OssPackage represents a package's details for OSS scanning.
type OssPackage struct {
	PackageManager  string                    `json:"PackageManager"`
	PackageName     string                    `json:"PackageName"`
	PackageVersion  string                    `json:"PackageVersion"`
	FilePath        string                    `json:"FilePath"`
	Locations       []realtimeengine.Location `json:"Locations"`
	Status          string                    `json:"Status"`
	Vulnerabilities []Vulnerability           `json:"Vulnerabilities"`
}

// OssPackageResults holds the results of an OSS scan.
type OssPackageResults struct {
	Packages []OssPackage `json:"Packages"`
}

type IgnoredPackage struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	PackageVersion string `json:"PackageVersion"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}
