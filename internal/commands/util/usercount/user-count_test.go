package usercount

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestNewUserCountCommand(t *testing.T) {
	mockParentCmd := &cobra.Command{
		Use:   "mock",
		Short: "mock",
	}
	cmd := NewUserCountCommand(nil, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "User count command must exist")

	mockParentCmd.AddCommand(cmd)

	mockParentCmd.SetArgs([]string{UcCommand, "-h"})
	assert.NilError(t, cmd.Execute())
}
