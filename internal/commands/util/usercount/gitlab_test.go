package usercount

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestGitLabUserCountGroups(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, nil, nil, mock.GitLabMockWrapper{})
	assert.Assert(t, cmd != nil, "GitLab user count command must exist")

	cmd.SetArgs(
		[]string{
			GitLabCommand,
			"--" + GitLabGroupsFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestGitLabUserCountProjects(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, nil, nil, mock.GitLabMockWrapper{})
	assert.Assert(t, cmd != nil, "GitLab user count command must exist")

	cmd.SetArgs(
		[]string{
			GitLabCommand,
			"--" + GitLabProjectsFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}

func TestGitLabUserCountError(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, nil, nil, mock.GitLabMockWrapper{})
	assert.Assert(t, cmd != nil, "GitLab user count command must exist")

	cmd.SetArgs(
		[]string{
			GitLabCommand,
			"--" + GitLabGroupsFlag,
			"a,b",
			"--" + GitLabProjectsFlag,
			"c,d",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, gitLabTooManyGroupsAndProjects)
}

func TestGitLabUserCountOnlyToken(t *testing.T) {
	cmd := NewUserCountCommand(nil, nil, nil, nil, mock.GitLabMockWrapper{})
	assert.Assert(t, cmd != nil, "GitLab user count command must exist")

	cmd.SetArgs(
		[]string{
			GitLabCommand,
			"--" + params.SCMTokenFlag,
			"a",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil)
}
