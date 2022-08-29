//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
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
	assertError(t, err, "Response status code 201")
}

func TestPRDecorationFailure(t *testing.T) {
	_ = viper.BindEnv(params.SCMTokenKey)
	_ = viper.BindEnv(params.OrgNamespaceKey)
	_ = viper.BindEnv(params.OrgRepoNameKey)
	_ = viper.BindEnv(params.PRNumberKey)

	args := []string{
		"utils",
		"pr",
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
	assertError(t, err, "Value of scan-id is invalid")
}
