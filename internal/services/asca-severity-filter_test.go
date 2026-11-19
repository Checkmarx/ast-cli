package services

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/stretchr/testify/assert"
)

func ascaDetails() []grpcs.ScanDetail {
	return []grpcs.ScanDetail{
		{RuleID: 1, RuleName: "Critical rule", Severity: "Critical"},
		{RuleID: 2, RuleName: "High rule", Severity: "High"},
		{RuleID: 3, RuleName: "Medium rule", Severity: "Medium"},
		{RuleID: 4, RuleName: "Low rule", Severity: "Low"},
	}
}

func TestFilterScanDetailsBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := ascaDetails()
	out := FilterScanDetailsBySeverity(in, nil)
	assert.Equal(t, in, out)
}

func TestFilterScanDetailsBySeverity_CriticalHigh_DropsMediumAndLow(t *testing.T) {
	f, err := realtimeengine.ParseSeverityFilter("critical,high")
	assert.NoError(t, err)

	out := FilterScanDetailsBySeverity(ascaDetails(), f)
	assert.Len(t, out, 2)
	assert.Equal(t, "Critical", out[0].Severity)
	assert.Equal(t, "High", out[1].Severity)
}

func TestFilterScanDetailsBySeverity_NoMatch_ReturnsEmptyNonNil(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("info")
	out := FilterScanDetailsBySeverity(ascaDetails(), f)
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}
