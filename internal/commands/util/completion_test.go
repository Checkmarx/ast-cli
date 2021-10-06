package util

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := NewCompletionCommand()
	assert.Assert(t, cmd != nil, "Completion command must exist")

	err := cmd.Execute()
	assert.Assert(t, err != nil, "Shell type must be defined")

	testShell(cmd, "bash", t)
	testShell(cmd, "zsh", t)
	testShell(cmd, "fish", t)
	testShell(cmd, "powershell", t)

	//TODO: catch console to check completion was printed out
	//TODO: wrong type does not return error
	testShell(cmd, "cenas", t)
}

func testShell(cmd *cobra.Command, shellType string, t *testing.T) {
	args := []string{"-s", shellType}
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.NilError(t, err, "Completion command should run with no errors")
}
