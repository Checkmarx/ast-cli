package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestRunAIOInstallCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "aio", "install", toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
	err = executeTestCommand(cmd, "-v", "aio", "install", toFlag(configFileFlag), "./payloads/config.yml")
	assert.NilError(t, err)
}
