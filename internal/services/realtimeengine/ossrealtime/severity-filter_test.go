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
			PackageName: "critical-pkg", Status: "Critical",
			Vulnerabilities: []Vulnerability{
				{CVE: "CVE-2", Severity: "Critical"},
				{CVE: "CVE-3", Severity: "High"},
				{CVE: "CVE-4", Severity: "Medium"},
				{CVE: "CVE-5", Severity: "Low"},
			},
		},
		{
			PackageName: "high-pkg", Status: "High",
			Vulnerabilities: []Vulnerability{{CVE: "CVE-6", Severity: "High"}, {CVE: "CVE-7", Severity: "Low"}},
		},
		{
			PackageName: "low-pkg", Status: "Low",
			Vulnerabilities: []Vulnerability{{CVE: "CVE-8", Severity: "Low"}},
		},
		{PackageName: "clean", Status: "OK"},
		{PackageName: "unresolved", Status: "Unknown"},
	}
}

func packageNames(packages []OssPackage) []string {
	names := make([]string, 0, len(packages))
	for i := range packages {
		names = append(names, packages[i].PackageName)
	}
	return names
}

func filterPackages(t *testing.T, threshold string) []OssPackage {
	parsed, err := realtimeengine.ParseSeverityThreshold(threshold)
	assert.NoError(t, err)
	return FilterPackagesBySeverity(ossPackages(), parsed)
}

func TestFilterPackagesBySeverity_ZeroThreshold_ReturnsInput(t *testing.T) {
	in := ossPackages()
	assert.Equal(t, in, FilterPackagesBySeverity(in, 0))
}

func TestFilterPackagesBySeverity_High_DropsPackagesBelowAndTrimsTheRest(t *testing.T) {
	out := filterPackages(t, "high")

	assert.Equal(t, []string{"evil", "critical-pkg", "high-pkg", "clean", "unresolved"}, packageNames(out))
	assert.Equal(t, []Vulnerability{{CVE: "CVE-2", Severity: "Critical"}, {CVE: "CVE-3", Severity: "High"}}, out[1].Vulnerabilities)
	assert.Equal(t, []Vulnerability{{CVE: "CVE-6", Severity: "High"}}, out[2].Vulnerabilities)
}

func TestFilterPackagesBySeverity_Critical_KeepsOnlyCriticalAndUnrankedPackages(t *testing.T) {
	out := filterPackages(t, "critical")

	assert.Equal(t, []string{"evil", "critical-pkg", "clean", "unresolved"}, packageNames(out))
	assert.Equal(t, []Vulnerability{{CVE: "CVE-2", Severity: "Critical"}}, out[1].Vulnerabilities)
}

func TestFilterPackagesBySeverity_LowAndInfo_KeepEverything(t *testing.T) {
	assert.Equal(t, ossPackages(), filterPackages(t, "low"))
	assert.Equal(t, ossPackages(), filterPackages(t, "info"))
}

func TestFilterPackagesBySeverity_MaliciousKeptUntouched(t *testing.T) {
	out := filterPackages(t, "critical")

	assert.Equal(t, ossPackages()[0], out[0], "malicious package must be returned unchanged")
}

func TestFilterPackagesBySeverity_StatusMatchedIgnoringCase(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("high")
	in := []OssPackage{{PackageName: "upper", Status: "HIGH"}, {PackageName: "lower", Status: "low"}, {PackageName: "evil", Status: "MALICIOUS"}}

	assert.Equal(t, []string{"upper", "evil"}, packageNames(FilterPackagesBySeverity(in, threshold)))
}

func TestFilterPackagesBySeverity_AllDropped_ReturnsEmptyList(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("critical")
	out := FilterPackagesBySeverity([]OssPackage{{PackageName: "low-pkg", Status: "Low"}}, threshold)

	assert.NotNil(t, out, "an emptied list must serialise as [] rather than null")
	assert.Empty(t, out)
}

func TestFilterPackagesBySeverity_DoesNotMutateInput(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("critical")
	in := ossPackages()
	_ = FilterPackagesBySeverity(in, threshold)
	assert.Equal(t, ossPackages(), in, "input slice must not be mutated")
}
