package usercount

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestNewUserCountCommand(t *testing.T) {
	mockParentCmd := &cobra.Command{
		Use:   "mock",
		Short: "mock",
	}

	mock := mock.AzureMockWrapper{}
	mock.GetCommits("","","","","")
	mock.GetCommits("mock","mock","mock","mock","mock")
	mock.GetCommits("mock","mock","mock","mock","mock")
	mock.GetRepositories("","","","")
	mock.GetRepositories("mock","mock","mock","mock")
	mock.GetProjects("","","")
	mock.GetProjects("mock","mock","mock")
	cmd := NewUserCountCommand(nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "User count command must exist")

	mockParentCmd.AddCommand(cmd)

	mockParentCmd.SetArgs([]string{UcCommand, "-h"})
	assert.NilError(t, cmd.Execute())
}
