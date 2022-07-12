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
	_, err := mock.GetCommits("", "", "", "", "")
	if err != nil {
		return
	}
	_, err = mock.GetCommits("mock", "mock", "mock", "mock", "mock")
	if err != nil {
		return
	}
	_, err = mock.GetCommits("mock", "mock", "mock", "mock", "mock")
	if err != nil {
		return
	}
	_, err = mock.GetRepositories("", "", "", "")
	if err != nil {
		return
	}
	_, err = mock.GetRepositories("mock", "mock", "mock", "mock")
	if err != nil {
		return
	}
	_, err = mock.GetProjects("", "", "")
	if err != nil {
		return
	}
	_, err = mock.GetProjects("mock", "mock", "mock")
	if err != nil {
		return
	}
	cmd := NewUserCountCommand(nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "User count command must exist")

	mockParentCmd.AddCommand(cmd)

	mockParentCmd.SetArgs([]string{UcCommand, "-h"})
	assert.NilError(t, cmd.Execute())
}
