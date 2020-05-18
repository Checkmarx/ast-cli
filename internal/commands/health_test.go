package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestRunHealthCommandNoFlag(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "health",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunHealthCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "health")
	assert.NilError(t, err)
}
