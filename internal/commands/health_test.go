package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestWebAppHealthCheck(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "single-node", "health-check")
	assert.NilError(t, err)
}
