package ossrealtime

import (
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// VulnerabilityMapper provides methods to convert vulnerabilities from different sources to the internal model.
type VulnerabilityMapper struct{}

func NewOssVulnerabilityMapper() *VulnerabilityMapper {
	return &VulnerabilityMapper{}
}

// FromRealtimeScanner maps a slice of RealtimeScannerVulnerability to the internal Vulnerability model.
func (VulnerabilityMapper) FromRealtimeScanner(in []wrappers.RealtimeScannerVulnerability) []Vulnerability {
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

// FromCache maps a slice of cache Vulnerability to the internal Vulnerability model.
func (VulnerabilityMapper) FromCache(in []osscache.Vulnerability) []Vulnerability {
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
