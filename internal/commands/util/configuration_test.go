package util

import (
	"fmt"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
	"testing"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()
	assert.Assert(t, cmd != nil, "Config command must exist")

	// Test show command
	err := executeTestCommand(cmd, "show")
	assert.NilError(t, err)

	// Test configure command
	err = executeTestCommand(cmd, "set", "--prop-name", "cx_client_id", "--prop-value", "dummy-client_id")
	assert.NilError(t, err)

	// Test configure command unknown property
	err = executeTestCommand(cmd, "set", "--prop-name", "nonexistent_prop", "--prop-value", "dummy")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed to set property: unknown property or bad value")

	//TODO: try to cover prompt configuration
}

func executeTestCommand(cmd *cobra.Command, args ...string) error {
	fmt.Println("Executing command with args ", args)
	cmd.SetArgs(args)
	cmd.SilenceUsage = false
	return cmd.Execute()
}
