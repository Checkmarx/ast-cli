package osscache

import (
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type PackageEntry struct {
	PackageManager  string          `json:"packageManager"`
	PackageName     string          `json:"packageName"`
	PackageVersion  string          `json:"packageVersion"`
	Status          string          `json:"status"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
	CVE         string `json:"CVE"`
	Description string `json:"Description"`
	Severity    string `json:"Severity"`
}

func (p *PackageEntry) ConvertVulnerabilities() []wrappers.RealtimeScannerVulnerability {
	vulnerabilities := make([]wrappers.RealtimeScannerVulnerability, len(p.Vulnerabilities))
	for i, v := range p.Vulnerabilities {
		vulnerabilities[i] = wrappers.RealtimeScannerVulnerability{
			CVE:         v.CVE,
			Description: v.Description,
			Severity:    v.Severity,
		}
	}
	return vulnerabilities
}

type Cache struct {
	TTL      time.Time      `json:"ttl"`
	Packages []PackageEntry `json:"packages"`
}

func (c *Cache) GetTTL() time.Time {
	return c.TTL
}

func (c *Cache) SetTTL(t time.Time) {
	c.TTL = t
}
