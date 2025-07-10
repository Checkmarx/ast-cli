//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTelemetryAI(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"telemetry", "ai",
		flag(params.AiProviderFlag), "Cursor",
		flag(params.TimestampFlag), "2025-07-10T12:34:56+03:00",
		flag(params.ProblemSeverityFlag), "Critical",
		flag(params.ClickTypeFlag), "ast-results.viewPackageDetails",
		flag(params.AgentFlag), "Cursor",
		flag(params.EngineFlag), "secrets",
	}

	err, _ := executeCommand(t, args...)
	assert.Nil(t, err)
}
