package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
	"testing"
)

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--timestamp", "2025-07-10T12:34:56+03:00", "--problem-severity", "Critical", "--click-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}

func TestRunTelemetryAI_ValidTimestamp(t *testing.T) {
	execCmdNilAssertion(t,
		"telemetry", "ai", "--ai-provider", "TestAI", "--timestamp", "2025-07-10T12:34:56+03:00", "--problem-severity", "High", "--click-type", "test.click", "--agent", "TestAgent", "--engine", "TestEngine")
}

func TestRunTelemetryAI_InvalidTimestamp(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"telemetry", "ai", "--ai-provider", "TestAI", "--timestamp", "invalid-timestamp", "--problem-severity", "High", "--click-type", "test.click", "--agent", "TestAgent", "--engine", "TestEngine")
	assert.Assert(t, err.Error() == "Invalid timestamp format: parsing time \"invalid-timestamp\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"invalid-timestamp\" as \"2006\"")
}

func TestRunTelemetryAI_EmptyTimestamp(t *testing.T) {
	execCmdNilAssertion(t,
		"telemetry", "ai", "--ai-provider", "TestAI", "--problem-severity", "High", "--click-type", "test.click", "--agent", "TestAgent", "--engine", "TestEngine")
}

func TestTelemetryHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "telemetry", "ai")
}
