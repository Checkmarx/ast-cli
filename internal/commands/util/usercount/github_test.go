package usercount

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestGitHubUserCountOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + OrgsFlag,
			"a,b",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}
func TestGitHubUserCountRepos(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + OrgsFlag,
			"a",
			"--" + ReposFlag,
			"a,b,c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}

func TestGitHubUserCountMissingArgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingArgs)
}

func TestGitHubUserCountMissingOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + ReposFlag,
			"repo",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingOrg)
}

func TestGitHubUserCountManyOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{}, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			GithubCommand,
			"--" + ReposFlag,
			"repo",
			"--" + OrgsFlag,
			"a,b,c",
			"--" + params.FormatFlag,
			printer.FormatJSON,
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, tooManyOrgs)
}
