// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestResultHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "result")
	assert.NilError(t, err)
}
