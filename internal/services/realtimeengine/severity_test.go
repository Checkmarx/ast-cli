package realtimeengine

import (
	"bytes"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func thresholdCmd(t *testing.T, args ...string) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SeverityThresholdFlag, "", "")
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	assert.NoError(t, cmd.Flags().Parse(args))
	return cmd, stderr
}

func TestSeverityThresholdFromCmd_EmptyValue_PrintsNoteAndReturnsZero(t *testing.T) {
	for _, value := range []string{"", "   "} {
		cmd, stderr := thresholdCmd(t, "--severity-threshold", value)
		threshold, err := SeverityThresholdFromCmd(cmd)
		assert.NoError(t, err, "input %q", value)
		assert.Equal(t, SeverityThreshold(0), threshold, "input %q", value)
		assert.Equal(t, emptyThresholdNote+"\n", stderr.String(), "input %q", value)
	}
}

func TestSeverityThresholdFromCmd_FlagOmittedOrSet_PrintsNoNote(t *testing.T) {
	for _, args := range [][]string{nil, {"--severity-threshold", "high"}, {"--severity-threshold", "hgh"}} {
		cmd, stderr := thresholdCmd(t, args...)
		_, _ = SeverityThresholdFromCmd(cmd)
		assert.Empty(t, stderr.String(), "args %v", args)
	}
}

func TestParseSeverityThreshold_EmptyOrWhitespace_ReturnsZero(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		threshold, err := ParseSeverityThreshold(raw)
		assert.NoError(t, err, "input %q", raw)
		assert.Equal(t, SeverityThreshold(0), threshold, "input %q", raw)
		assert.True(t, threshold.Allows("Low"), "zero threshold must allow everything")
	}
}

func TestParseSeverityThreshold_ValidLevels_CaseAndSpacesIgnored(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		rank SeverityThreshold
	}{{"critical", 5}, {"High", 4}, {" MEDIUM ", 3}, {"low", 2}, {"Info", 1}} {
		threshold, err := ParseSeverityThreshold(tc.raw)
		assert.NoError(t, err, "input %q", tc.raw)
		assert.Equal(t, tc.rank, threshold, "input %q", tc.raw)
	}
}

func TestParseSeverityThreshold_InvalidValue_ReturnsError(t *testing.T) {
	for _, raw := range []string{"hgh", "urgent", "critical,high", "high,", ",", " , "} {
		threshold, err := ParseSeverityThreshold(raw)
		assert.Error(t, err, "input %q", raw)
		assert.Equal(t, SeverityThreshold(0), threshold, "input %q", raw)
		assert.Contains(t, err.Error(), "realtime engine error")
		assert.Contains(t, err.Error(), "invalid severity-threshold value")
		assert.Contains(t, err.Error(), "single level: critical, high, medium, low or info")
	}
}

func TestParseSeverityThreshold_InvalidValue_ErrorNamesTheValue(t *testing.T) {
	_, err := ParseSeverityThreshold(" Hgh ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `"Hgh"`)
}

func TestSeverityThreshold_Allows_AtOrAboveThreshold(t *testing.T) {
	high, _ := ParseSeverityThreshold("high")
	for _, tc := range []struct {
		severity string
		allowed  bool
	}{
		{"Critical", true}, {"High", true}, {"HIGH", true}, {" high ", true},
		{"Medium", false}, {"Low", false}, {"Info", false},
	} {
		assert.Equal(t, tc.allowed, high.Allows(tc.severity), "severity %q at threshold high", tc.severity)
	}
}

func TestSeverityThreshold_Allows_UnrecognisedSeverityIsKept(t *testing.T) {
	critical, _ := ParseSeverityThreshold("critical")
	for _, severity := range []string{"Unknown", "none", "", "Information"} {
		assert.True(t, critical.Allows(severity), "severity %q has no rank and must be kept", severity)
	}
}

func TestSeverityThreshold_Allows_InfoThresholdAllowsEveryLevel(t *testing.T) {
	info, _ := ParseSeverityThreshold("info")
	for _, severity := range severityOrder {
		assert.True(t, info.Allows(severity), "severity %q at threshold info", severity)
	}
}

type finding struct{ id, severity string }

func severityOf(f finding) string { return f.severity }

func TestFilterBySeverity_ZeroThreshold_ReturnsInput(t *testing.T) {
	in := []finding{{"1", "High"}, {"2", "Low"}}
	assert.Equal(t, in, FilterBySeverity(in, 0, severityOf))
}

func TestFilterBySeverity_KeepsFindingsAtOrAboveThresholdInOrder(t *testing.T) {
	medium, _ := ParseSeverityThreshold("medium")
	in := []finding{{"1", "Low"}, {"2", "Critical"}, {"3", "Info"}, {"4", "MEDIUM"}, {"5", "high"}, {"6", "Unknown"}}

	out := FilterBySeverity(in, medium, severityOf)

	assert.Equal(t, []finding{{"2", "Critical"}, {"4", "MEDIUM"}, {"5", "high"}, {"6", "Unknown"}}, out)
	assert.Len(t, in, 6, "input must not be modified")
}

func TestFilterBySeverity_NothingAtOrAbove_ReturnsEmptyNonNil(t *testing.T) {
	critical, _ := ParseSeverityThreshold("critical")
	out := FilterBySeverity([]finding{{"1", "High"}, {"2", "Low"}}, critical, severityOf)
	assert.NotNil(t, out)
	assert.Empty(t, out)
}

func TestFilterBySeverity_EmptyInput_ReturnedUnchanged(t *testing.T) {
	high, _ := ParseSeverityThreshold("high")
	assert.Nil(t, FilterBySeverity(nil, high, severityOf))
	assert.NotNil(t, FilterBySeverity([]finding{}, high, severityOf))
}
