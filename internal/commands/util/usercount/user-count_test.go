package usercount

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
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

func TestGetEnabledRepositories_2Enable1DisableRepo_Success(t *testing.T) {
	repos := []wrappers.AzureRepo{
		{
			Name:       "MOCK REPO",
			IsDisabled: false,
		},
		{
			Name:       "MOCK REPO 2",
			IsDisabled: true,
		},
		{
			Name:       "MOCK REPO 3",
			IsDisabled: false,
		},
	}
	rootRepo := wrappers.AzureRootRepo{Repos: repos}
	enabledRepos := rootRepo.GetEnabledRepos()
	assert.Equal(t, len(enabledRepos.Repos), 2)
}
