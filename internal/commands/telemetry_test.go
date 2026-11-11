package commands

import (
	"testing"
)

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--problem-severity", "Critical", "--type", "click", "--sub-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}

func TestTelemetryHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "telemetry", "ai")
}
