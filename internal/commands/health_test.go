package commands

import (
	"gotest.tools/assert"
	"testing"
)

func TestWebAppHealthCheck(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "single-node", "health-check")
	assert.NilError(t, err)
}
