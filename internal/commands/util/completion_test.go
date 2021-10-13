package util

import (
	"testing"

	"gotest.tools/assert"

	"github.com/spf13/cobra"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := testCmdExistsAndRequiredShell(t)

	testShell(cmd, "bash", true, t)
	testShell(cmd, "zsh", true, t)
	testShell(cmd, "fish", true, t)
	testShell(cmd, "powershell", true, t)
}

func TestNewCompletionCommandInvalidValue(t *testing.T) {
	cmd := testCmdExistsAndRequiredShell(t)

	testShell(cmd, "invalidShellValue", false, t)
}

func testCmdExistsAndRequiredShell(t *testing.T) *cobra.Command {
	cmd := NewCompletionCommand()
	assert.Assert(t, cmd != nil, "Completion command must exist")

	err := cmd.Execute()
	assert.Assert(t, err != nil, "Shell type must be defined")

	return cmd
}

func testShell(cmd *cobra.Command, shellType string, assertNil bool, t *testing.T) {
	err := executeTestCommand(cmd, "--shell", shellType)

	if assertNil {
		assert.NilError(t, err)
		return
	}

	assert.Assert(t, err != nil)
}
