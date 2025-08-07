package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--problem-severity", "Critical", "--type", "click", "--sub-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}

func TestTelemetryHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "telemetry", "ai")
}
