package containersrealtime

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/stretchr/testify/assert"
)

func containerImages() []ContainerImage {
	return []ContainerImage{
		{
			ImageName: "evil", ImageTag: "latest", Status: statusMalicious,
			Vulnerabilities: []Vulnerability{{CVE: "CVE-1", Severity: "Low"}},
		},
		{
			ImageName: "mixed", ImageTag: "1.0", Status: "OK",
			Vulnerabilities: []Vulnerability{
				{CVE: "CVE-2", Severity: "Critical"},
				{CVE: "CVE-3", Severity: "High"},
				{CVE: "CVE-4", Severity: "Low"},
			},
		},
		{
			ImageName: "low-only", ImageTag: "2.0", Status: "OK",
			Vulnerabilities: []Vulnerability{{CVE: "CVE-5", Severity: "Low"}},
		},
	}
}

func TestFilterImagesBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := containerImages()
	assert.Equal(t, in, FilterImagesBySeverity(in, nil))
}

func TestFilterImagesBySeverity_MaliciousKeptUntouched(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical")
	out := FilterImagesBySeverity(containerImages(), f)

	assert.Equal(t, "evil", out[0].ImageName)
	assert.Len(t, out[0].Vulnerabilities, 1)
	assert.Equal(t, "Low", out[0].Vulnerabilities[0].Severity)
}

func TestFilterImagesBySeverity_VulnerabilitiesTrimmedToFilter(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical,high")
	out := FilterImagesBySeverity(containerImages(), f)

	assert.Len(t, out[1].Vulnerabilities, 2)
	assert.Equal(t, "CVE-2", out[1].Vulnerabilities[0].CVE)
	assert.Equal(t, "CVE-3", out[1].Vulnerabilities[1].CVE)
}

func TestFilterImagesBySeverity_EntryKeptWhenListBecomesEmpty(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical")
	in := containerImages()
	out := FilterImagesBySeverity(in, f)

	assert.Len(t, out, len(in))
	assert.Equal(t, "low-only", out[2].ImageName)
	assert.NotNil(t, out[2].Vulnerabilities)
	assert.Len(t, out[2].Vulnerabilities, 0)
	assert.Len(t, in[2].Vulnerabilities, 1, "input slice must not be mutated")
}
