package commands

import (
	"gotest.tools/assert"
	"testing"
)

func TestWebAppHealthCheck(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "health")
	assert.NilError(t, err)
}
