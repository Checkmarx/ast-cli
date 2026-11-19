package iacrealtime

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/stretchr/testify/assert"
)

func iacResults() []IacRealtimeResult {
	return []IacRealtimeResult{
		{Title: "crit", Severity: "Critical"},
		{Title: "high", Severity: "High"},
		{Title: "med", Severity: "Medium"},
		{Title: "low", Severity: "Low"},
		{Title: "info", Severity: "Info"},
		{Title: "unknown", Severity: "Unknown"},
	}
}

func TestFilterResultsBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := iacResults()
	assert.Equal(t, in, FilterResultsBySeverity(in, nil))
}

func TestFilterResultsBySeverity_CriticalHigh_DropsOthersIncludingUnknown(t *testing.T) {
	f, err := realtimeengine.ParseSeverityFilter("Critical,High")
	assert.NoError(t, err)

	out := FilterResultsBySeverity(iacResults(), f)
	assert.Len(t, out, 2)
	assert.Equal(t, "crit", out[0].Title)
	assert.Equal(t, "high", out[1].Title)
}

func TestFilterResultsBySeverity_Info_KeepsOnlyInfo(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("info")
	out := FilterResultsBySeverity(iacResults(), f)
	assert.Len(t, out, 1)
	assert.Equal(t, "Info", out[0].Severity)
}
