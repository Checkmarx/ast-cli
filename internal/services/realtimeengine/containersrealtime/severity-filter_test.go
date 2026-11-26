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
			ImageName: "critical-img", ImageTag: "1.0", Status: "Critical",
			Vulnerabilities: []Vulnerability{
				{CVE: "CVE-2", Severity: "Critical"},
				{CVE: "CVE-3", Severity: "High"},
				{CVE: "CVE-4", Severity: "Low"},
			},
		},
		{
			ImageName: "low-img", ImageTag: "2.0", Status: "Low",
			Vulnerabilities: []Vulnerability{{CVE: "CVE-5", Severity: "Low"}},
		},
		{ImageName: "clean", ImageTag: "3.0", Status: "OK"},
		{ImageName: "pinned", ImageTag: "sha256:abc", Status: "Unknown"},
	}
}

func imageNames(images []ContainerImage) []string {
	names := make([]string, 0, len(images))
	for i := range images {
		names = append(names, images[i].ImageName)
	}
	return names
}

func TestFilterImagesBySeverity_ZeroThreshold_ReturnsInput(t *testing.T) {
	in := containerImages()
	assert.Equal(t, in, FilterImagesBySeverity(in, 0))
}

func TestFilterImagesBySeverity_High_DropsImagesBelowAndTrimsTheRest(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("high")
	out := FilterImagesBySeverity(containerImages(), threshold)

	assert.Equal(t, []string{"evil", "critical-img", "clean", "pinned"}, imageNames(out))
	assert.Equal(t, []Vulnerability{{CVE: "CVE-2", Severity: "Critical"}, {CVE: "CVE-3", Severity: "High"}}, out[1].Vulnerabilities)
}

func TestFilterImagesBySeverity_Low_KeepsEverything(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("low")
	assert.Equal(t, containerImages(), FilterImagesBySeverity(containerImages(), threshold))
}

func TestFilterImagesBySeverity_MaliciousKeptUntouched(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("critical")
	out := FilterImagesBySeverity(containerImages(), threshold)

	assert.Equal(t, containerImages()[0], out[0], "malicious image must be returned unchanged")
}

func TestFilterImagesBySeverity_DoesNotMutateInput(t *testing.T) {
	threshold, _ := realtimeengine.ParseSeverityThreshold("critical")
	in := containerImages()
	_ = FilterImagesBySeverity(in, threshold)
	assert.Equal(t, containerImages(), in, "input slice must not be mutated")
}
