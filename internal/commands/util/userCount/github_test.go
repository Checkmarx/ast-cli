package userCount

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestGitHubUserCountOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{})
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			"github",
			"--orgs",
			"a,b",
			"--format",
			"json",
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}
func TestGitHubUserCountRepos(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{})
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			"github",
			"--orgs",
			"a",
			"--repos",
			"a,b,c",
			"--format",
			"json",
		},
	)

	err := cmd.Execute()
	assert.Assert(t, err == nil || strings.Contains(err.Error(), "rate"))
}

func TestGitHubUserCountMissingArgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{})
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			"github",
			"--format",
			"json",
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingArgs)
}

func TestGitHubUserCountMissingOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{})
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			"github",
			"--repos",
			"repo",
			"--format",
			"json",
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, missingOrg)
}

func TestGitHubUserCountManyOrgs(t *testing.T) {
	cmd := NewUserCountCommand(mock.GitHubMockWrapper{})
	assert.Assert(t, cmd != nil, "GitHub user count command must exist")

	cmd.SetArgs(
		[]string{
			"github",
			"--repos",
			"repo",
			"--orgs",
			"a,b,c",
			"--format",
			"json",
		},
	)

	err := cmd.Execute()
	assert.Error(t, err, tooManyOrgs)
}
