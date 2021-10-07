package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewEnvCheckCommand(t *testing.T) {
	cmd := NewEnvCheckCommand()
	assert.Assert(t, cmd != nil, "Env check command must exist")

	err := cmd.Execute()
	assert.NilError(t, err, "Env check command should run with no errors")
}
