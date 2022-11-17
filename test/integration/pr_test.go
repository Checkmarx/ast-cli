//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

const (
	prGithubToken     = "PR_GITHUB_TOKEN"
	prGithubNamespace = "PR_GITHUB_NAMESPACE"
	prGithubNumber    = "PR_GITHUB_NUMBER"
	prGithubRepoName  = "PR_GITHUB_REPO_NAME"
)

func TestPRDecorationSuccessCase(t *testing.T) {
	scanID, _ := getRootScan(t)

	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		scanID,
		flag(params.SCMTokenFlag),
		os.Getenv(prGithubToken),
		flag(params.NamespaceFlag),
		os.Getenv(prGithubNamespace),
		flag(params.PRNumberFlag),
		os.Getenv(prGithubNumber),
		flag(params.RepoNameFlag),
		os.Getenv(prGithubRepoName),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")
}

func TestPRDecorationFailure(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		"",
		flag(params.SCMTokenFlag),
		os.Getenv(prGithubToken),
		flag(params.NamespaceFlag),
		os.Getenv(prGithubNamespace),
		flag(params.PRNumberFlag),
		os.Getenv(prGithubNumber),
		flag(params.RepoNameFlag),
		os.Getenv(prGithubRepoName),
	}
	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, "Failed creating PR Decoration")
}
