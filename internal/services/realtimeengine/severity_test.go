package realtimeengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSeverityFilter_EmptyOrWhitespace_ReturnsNilFilter(t *testing.T) {
	for _, raw := range []string{"", "   ", ",", " , ,"} {
		f, err := ParseSeverityFilter(raw)
		assert.NoError(t, err, "input %q", raw)
		assert.Nil(t, f, "input %q", raw)
		assert.True(t, f.Allows("High"), "nil filter must allow everything")
	}
}

func TestParseSeverityFilter_MixedCaseSpacesAndTrailingComma_Normalised(t *testing.T) {
	f, err := ParseSeverityFilter(" Critical, high ,HIGH,")
	assert.NoError(t, err)
	assert.Len(t, f, 2)
	assert.True(t, f.Allows("critical"))
	assert.True(t, f.Allows("CRITICAL"))
	assert.True(t, f.Allows(" High "))
	assert.False(t, f.Allows("Medium"))
	assert.False(t, f.Allows("Low"))
	assert.False(t, f.Allows(""))
}

func TestParseSeverityFilter_AllAcceptedValues(t *testing.T) {
	f, err := ParseSeverityFilter("critical,high,medium,low,info")
	assert.NoError(t, err)
	assert.Len(t, f, len(acceptedSeverities))
	for _, s := range acceptedSeverities {
		assert.True(t, f.Allows(s))
	}
	assert.False(t, f.Allows("unknown"))
}

func TestParseSeverityFilter_SingleInvalidToken_ReturnsError(t *testing.T) {
	f, err := ParseSeverityFilter("Critical,Hgh")
	assert.Error(t, err)
	assert.Nil(t, f)
	assert.Contains(t, err.Error(), "realtime engine error")
	assert.Contains(t, err.Error(), "hgh")
	assert.Contains(t, err.Error(), "critical, high, medium, low, info")
}

func TestParseSeverityFilter_MultipleInvalidTokens_SingleConsolidatedError(t *testing.T) {
	_, err := ParseSeverityFilter("foo,high,bar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "foo, bar")
}

func TestSeverityFilter_Allows_CaseInsensitive(t *testing.T) {
	f := SeverityFilter{"high": {}}
	assert.True(t, f.Allows("high"))
	assert.True(t, f.Allows("High"))
	assert.True(t, f.Allows("HIGH"))
	assert.False(t, f.Allows("medium"))
}

type finding struct{ id, severity string }

func severityOf(f finding) string { return f.severity }

func TestFilterBySeverity_NilFilter_ReturnsInput(t *testing.T) {
	in := []finding{{"1", "High"}, {"2", "Low"}}
	assert.Equal(t, in, FilterBySeverity(in, nil, severityOf))
}

func TestFilterBySeverity_KeepsOnlyAllowedSeveritiesIgnoringCase(t *testing.T) {
	f, _ := ParseSeverityFilter("critical,high")
	in := []finding{{"1", "Critical"}, {"2", "HIGH"}, {"3", "Medium"}, {"4", "Low"}, {"5", "Unknown"}, {"6", ""}}

	out := FilterBySeverity(in, f, severityOf)

	assert.Equal(t, []finding{{"1", "Critical"}, {"2", "HIGH"}}, out)
	assert.Len(t, in, 6, "input must not be modified")
}

func TestFilterBySeverity_NoMatch_ReturnsEmptyNonNil(t *testing.T) {
	f, _ := ParseSeverityFilter("low")
	out := FilterBySeverity([]finding{{"1", "High"}}, f, severityOf)
	assert.NotNil(t, out)
	assert.Empty(t, out)
}

func TestFilterBySeverity_EmptyInput_ReturnedUnchanged(t *testing.T) {
	f, _ := ParseSeverityFilter("high")
	assert.Nil(t, FilterBySeverity(nil, f, severityOf))
	assert.NotNil(t, FilterBySeverity([]finding{}, f, severityOf))
}
