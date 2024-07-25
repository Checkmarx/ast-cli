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
	prGitlabRepoName  = "PR_GITLAB_REPO_NAME"
	prGitlabToken     = "PR_GITLAB_TOKEN"
	prGitlabNamespace = "PR_GITLAB_NAMESPACE"
	prGitlabProjectId = "PR_GITLAB_PROJECT_ID"
	prGitlabIid       = "PR_GITLAB_IID"
)

func TestPRGithubDecorationSuccessCase(t *testing.T) {
	scanID, _ := getRootScan(t, params.SastType)
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
		"--debug",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")
}

func TestPRGithubDecorationFailure(t *testing.T) {
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
	assert.ErrorContains(t, err, "Failed creating github PR Decoration")
}

func TestPRGitlabDecorationSuccessCase(t *testing.T) {
	scanID, _ := getRootScan(t, params.SastType)

	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		scanID,
		flag(params.SCMTokenFlag),
		os.Getenv(prGitlabToken),
		flag(params.NamespaceFlag),
		os.Getenv(prGitlabNamespace),
		flag(params.RepoNameFlag),
		os.Getenv(prGitlabRepoName),
		flag(params.PRGitlabProjectFlag),
		os.Getenv(prGitlabProjectId),
		flag(params.PRIidFlag),
		os.Getenv(prGitlabIid),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")
}

func TestPRGitlabDecorationFailure(t *testing.T) {

	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		"",
		flag(params.SCMTokenFlag),
		os.Getenv(prGitlabToken),
		flag(params.NamespaceFlag),
		os.Getenv(prGitlabNamespace),
		flag(params.RepoNameFlag),
		os.Getenv(prGitlabRepoName),
		flag(params.PRGitlabProjectFlag),
		os.Getenv(prGitlabProjectId),
		flag(params.PRIidFlag),
		os.Getenv(prGitlabIid),
	}
	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, "Failed creating gitlab MR Decoration")
}
