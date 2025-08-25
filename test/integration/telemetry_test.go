//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
)

func TestTelemetryAI(t *testing.T) {
	t.Skip()
	bindKeysToEnvAndDefault(t)
	args := []string{
		"telemetry", "ai",
		flag(params.AiProviderFlag), "Cursor",
		flag(params.ProblemSeverityFlag), "Critical",
		flag(params.TypeFlag), "click",
		flag(params.SubTypeFlag), "ast-results.viewPackageDetails",
		flag(params.AgentFlag), "Cursor",
		flag(params.EngineFlag), "secrets",
	}

	err, _ := executeCommand(t, args...)
	assert.Nil(t, err)
}
