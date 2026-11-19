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
	assert.Len(t, f, len(AcceptedSeverities))
	for _, s := range AcceptedSeverities {
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
