//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/logger"

	"github.com/bouk/monkey"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

const (
	prGithubToken                 = "PR_GITHUB_TOKEN"
	prGithubNamespace             = "PR_GITHUB_NAMESPACE"
	prGithubNumber                = "PR_GITHUB_NUMBER"
	prGithubRepoName              = "PR_GITHUB_REPO_NAME"
	prGitlabRepoName              = "PR_GITLAB_REPO_NAME"
	prGitlabToken                 = "PR_GITLAB_TOKEN"
	prGitlabNamespace             = "PR_GITLAB_NAMESPACE"
	prGitlabProjectId             = "PR_GITLAB_PROJECT_ID"
	prGitlabIid                   = "PR_GITLAB_IID"
	prdDecorationForbiddenMessage = "A PR couldn't be created for this scan because it is still in progress."
	failedGettingScanError        = "Failed showing a scan"
	githubPRCommentCreated        = "github PR comment created successfully."
	gitlabPRCommentCreated        = "gitlab PR comment created successfully."
	outputFileName                = "test_output.log"
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

func TestPRGithubOnPremDecorationSuccessCase(t *testing.T) {
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
		flag(params.CodeRepositoryFlag),
		"https://github.example.com",
	}

	monkey.Patch((*wrappers.PRHTTPWrapper).PostPRDecoration, func(*wrappers.PRHTTPWrapper, *wrappers.PRModel) (string, *wrappers.WebError, error) {
		return githubPRCommentCreated, nil, nil
	})
	defer monkey.Unpatch(wrappers.PRWrapper.PostPRDecoration)

	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")

	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	assert.Equal(t, strings.Contains(stdoutString, githubPRCommentCreated), true, "Expected output: %s", githubPRCommentCreated)
}

func TestPRGithubDecorationFailure(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		"fakeScanID",
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
	assert.ErrorContains(t, err, "scan not found")
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

func TestPRGitlabOnPremDecorationSuccessCase(t *testing.T) {
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
		flag(params.CodeRepositoryFlag),
		"https://gitlab.example.com",
	}

	monkey.Patch((*wrappers.PRHTTPWrapper).PostPRDecoration, func(*wrappers.PRHTTPWrapper, *wrappers.GitlabPRModel) (string, *wrappers.WebError, error) {
		return gitlabPRCommentCreated, nil, nil
	})
	defer monkey.Unpatch(wrappers.PRWrapper.PostPRDecoration)

	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")

	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	assert.Equal(t, strings.Contains(stdoutString, gitlabPRCommentCreated), true, "Expected output: %s", gitlabPRCommentCreated)
}

func TestPRGitlabDecorationFailure(t *testing.T) {

	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		"fakeScanID",
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
	assert.ErrorContains(t, err, "scan not found")
}

func TestPRGithubDecoration_WhenScanIsRunning_ShouldAvoidPRDecorationCommand(t *testing.T) {
	scanID, _ := createScanNoWait(t, Zip, Tags, getProjectNameForScanTests())
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

	file := createOutputFile(t, outputFileName)
	_, _ = executeCommand(t, args...)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, prdDecorationForbiddenMessage), true, "Expected output: %s", prdDecorationForbiddenMessage)

	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)
}

func TestPRGitlabDecoration_WhenScanIsRunning_ShouldAvoidPRDecorationCommand(t *testing.T) {
	scanID, _ := createScanNoWait(t, Zip, Tags, getProjectNameForScanTests())
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

	file := createOutputFile(t, outputFileName)
	_, _ = executeCommand(t, args...)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, prdDecorationForbiddenMessage), true, "Expected output: %s", prdDecorationForbiddenMessage)

	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)
}

func createOutputFile(t *testing.T, fileName string) *os.File {
	file, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	logger.SetOutput(file)
	return file
}

func deleteOutputFile(t *testing.T, file *os.File) {
	file.Close()
	err := os.Remove(file.Name())
	if err != nil {
		logger.Printf("Failed to remove log file: %v", err)
	}
}
