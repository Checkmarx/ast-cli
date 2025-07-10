package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"testing"
)

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--timestamp", "2025-07-10T12:34:56+03:00", "--problem-severity", "Critical", "--click-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}
