package ossrealtime

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/stretchr/testify/assert"
)

func ossPackages() []OssPackage {
	return []OssPackage{
		{
			PackageName: "evil", Status: statusMalicious,
			Vulnerabilities: []Vulnerability{{CVE: "CVE-1", Severity: "Low"}},
		},
		{
			PackageName: "mixed", Status: "OK",
			Vulnerabilities: []Vulnerability{
				{CVE: "CVE-2", Severity: "Critical"},
				{CVE: "CVE-3", Severity: "High"},
				{CVE: "CVE-4", Severity: "Medium"},
				{CVE: "CVE-5", Severity: "Low"},
			},
		},
		{
			PackageName: "low-only", Status: "OK",
			Vulnerabilities: []Vulnerability{{CVE: "CVE-6", Severity: "Low"}},
		},
		{PackageName: "clean", Status: "OK"},
	}
}

func TestFilterPackagesBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := ossPackages()
	assert.Equal(t, in, FilterPackagesBySeverity(in, nil))
}

func TestFilterPackagesBySeverity_MaliciousKeptUntouched(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical")
	out := FilterPackagesBySeverity(ossPackages(), f)

	assert.Equal(t, "evil", out[0].PackageName)
	assert.Equal(t, statusMalicious, out[0].Status)
	assert.Len(t, out[0].Vulnerabilities, 1, "malicious package vulnerabilities must not be trimmed")
	assert.Equal(t, "Low", out[0].Vulnerabilities[0].Severity)
}

func TestFilterPackagesBySeverity_VulnerabilitiesTrimmedToFilter(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical,high")
	out := FilterPackagesBySeverity(ossPackages(), f)

	mixed := out[1]
	assert.Equal(t, "mixed", mixed.PackageName)
	assert.Len(t, mixed.Vulnerabilities, 2)
	assert.Equal(t, "CVE-2", mixed.Vulnerabilities[0].CVE)
	assert.Equal(t, "CVE-3", mixed.Vulnerabilities[1].CVE)
}

func TestFilterPackagesBySeverity_EntryKeptWhenListBecomesEmpty(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical")
	in := ossPackages()
	out := FilterPackagesBySeverity(in, f)

	assert.Len(t, out, len(in), "every package entry must be kept")
	lowOnly := out[2]
	assert.Equal(t, "low-only", lowOnly.PackageName)
	assert.Equal(t, "OK", lowOnly.Status)
	assert.NotNil(t, lowOnly.Vulnerabilities)
	assert.Len(t, lowOnly.Vulnerabilities, 0)
}

func TestFilterPackagesBySeverity_DoesNotMutateInput(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical")
	in := ossPackages()
	_ = FilterPackagesBySeverity(in, f)
	assert.Len(t, in[1].Vulnerabilities, 4, "input slice must not be mutated")
}
