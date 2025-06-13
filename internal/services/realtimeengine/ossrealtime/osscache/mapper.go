package osscache

import "github.com/checkmarx/ast-cli/internal/wrappers"

type OssCacheVulnerabilityMapper struct{}

func NewOssCacheVulnerabilityMapper() *OssCacheVulnerabilityMapper {
	return &OssCacheVulnerabilityMapper{}
}

func (o *OssCacheVulnerabilityMapper) FromRealtimeScannerVulnerability(in []wrappers.RealtimeScannerVulnerability) []Vulnerability {
	out := make([]Vulnerability, len(in))
	for i, v := range in {
		out[i] = Vulnerability{
			CVE:         v.CVE,
			Description: v.Description,
			Severity:    v.Severity,
		}
	}
	return out
}
