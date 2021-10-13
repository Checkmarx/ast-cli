package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewVersionCommand(t *testing.T) {
	cmd := NewVersionCommand()
	assert.Assert(t, cmd != nil, "Version command must exist")

	err := cmd.Execute()
	assert.NilError(t, err, "Version command should run with no errors")
}
