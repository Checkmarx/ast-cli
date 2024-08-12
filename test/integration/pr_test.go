//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/logger"
	"io/ioutil"
	"os"
	"strings"
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

	testFileName := "test_output.log"
	file := createOutputFile(t, testFileName)

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")

	prdDecorationForbiddenMessage := "no pr decoration have been created because scan did not end"
	stdoutString := readOutputFile(t, testFileName)
	assert.Equal(t, strings.Contains(stdoutString, prdDecorationForbiddenMessage), true, "Expected output: %s", prdDecorationForbiddenMessage, " Actual: %s", stdoutString)

	defer deleteOutputFile(t, testFileName, file)
	defer logger.SetOutput(os.Stdout)
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

func createOutputFile(t *testing.T, fileName string) *os.File {
	file, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	logger.SetOutput(file)
	return file
}

func readOutputFile(t *testing.T, fileName string) string {
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	return string(fileContent)
}

func deleteOutputFile(t *testing.T, fileName string, file *os.File) {
	file.Close()
	err := os.Remove(fileName)
	if err != nil {
		logger.Printf("Failed to remove log file: %v", err)
	}
}
