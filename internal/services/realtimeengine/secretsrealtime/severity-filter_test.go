package secretsrealtime

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/stretchr/testify/assert"
)

func secretsResults() []SecretsRealtimeResult {
	return []SecretsRealtimeResult{
		{Title: "valid-key", Severity: criticalSeverity},
		{Title: "unknown-key", Severity: highSeverity},
		{Title: "revoked-key", Severity: mediumSeverity},
	}
}

func TestFilterResultsBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := secretsResults()
	assert.Equal(t, in, FilterResultsBySeverity(in, nil))
}

func TestFilterResultsBySeverity_Critical_KeepsOnlyCritical(t *testing.T) {
	f, err := realtimeengine.ParseSeverityFilter("critical")
	assert.NoError(t, err)

	out := FilterResultsBySeverity(secretsResults(), f)
	assert.Len(t, out, 1)
	assert.Equal(t, "valid-key", out[0].Title)
}

func TestFilterResultsBySeverity_CriticalHigh_DropsMedium(t *testing.T) {
	f, _ := realtimeengine.ParseSeverityFilter("critical,high")
	out := FilterResultsBySeverity(secretsResults(), f)
	assert.Len(t, out, 2)
	for _, r := range out {
		assert.NotEqual(t, mediumSeverity, r.Severity)
	}
}
