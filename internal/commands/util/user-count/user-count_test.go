package user_count

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
	cmd := NewUserCountCommand(nil)
	assert.Assert(t, cmd != nil, "User count command must exist")

	mockParentCmd.AddCommand(cmd)

	mockParentCmd.SetArgs([]string{"user-count", "-h"})
	assert.NilError(t, cmd.Execute())
}
