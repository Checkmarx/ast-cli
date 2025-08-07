//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
	"testing"
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
