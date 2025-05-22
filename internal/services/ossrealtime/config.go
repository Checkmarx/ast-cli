package ossrealtime

import (
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// OssPackage represents a package's details for OSS scanning.
type OssPackage struct {
	PackageManager  string          `json:"PackageManager"`
	PackageName     string          `json:"PackageName"`
	PackageVersion  string          `json:"PackageVersion"`
	FilePath        string          `json:"FilePath"`
	LineStart       int             `json:"LineStart"`
	LineEnd         int             `json:"LineEnd"`
	StartIndex      int             `json:"StartIndex"`
	EndIndex        int             `json:"EndIndex"`
	Status          string          `json:"Status"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
}

// OssPackageResults holds the results of an OSS scan.
type OssPackageResults struct {
	Packages []OssPackage `json:"Packages"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}

func NewOssVulnerabilitiesFromRealtimeScannerVulnerabilities(vulnerabilities []wrappers.RealtimeScannerVulnerability) []Vulnerability {
	vulns := make([]Vulnerability, len(vulnerabilities))
	for i, v := range vulnerabilities {
		vulns[i] = Vulnerability{
			CVE:         v.CVE,
			Description: v.Description,
			Severity:    v.Severity,
		}
	}
	return vulns
}

func NewOssVulnerabilitiesFromOssCacheVulnerabilities(vulnerabilities []osscache.Vulnerability) []Vulnerability {
	vulns := make([]Vulnerability, len(vulnerabilities))
	for i, v := range vulnerabilities {
		vulns[i] = Vulnerability{
			CVE:         v.CVE,
			Description: v.Description,
			Severity:    v.Severity,
		}
	}
	return vulns
}
