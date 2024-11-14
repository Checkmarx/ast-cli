//go:build integration

package integration

import (
	"fmt"
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
	prAzureToken                  = "AZURE_NEW_TOKEN"
	prAzureOrganization           = "AZURE_NEW_ORG"
	prAzureProject                = "AZURE_PROJECT_NAME"
	prAzureNumber                 = "AZURE_PR_NUMBER"
	prdDecorationForbiddenMessage = "A PR couldn't be created for this scan because it is still in progress."
	failedGettingScanError        = "Failed showing a scan"
	githubPRCommentCreated        = "github PR comment created successfully."
	gitlabPRCommentCreated        = "gitlab PR comment created successfully."
	azurePRCommentCreated         = "azure PR comment created successfully."
	outputFileName                = "test_output.log"
	scans                         = "api/scans"
)

var completedScanId = ""
var runningScanId = ""

func getCompletedScanID(t *testing.T) string {
	if completedScanId != "" {
		return completedScanId
	}
	scanWrapper := wrappers.NewHTTPScansWrapper(scans)
	scanID, _ := getRootScan(t, params.IacType)

	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)

	for isRunning, err := util.IsScanRunningOrQueued(scanWrapper, scanID); isRunning; isRunning, err = util.IsScanRunningOrQueued(scanWrapper, scanID) {
		if err != nil {
			t.Fatalf("Failed to get scan status: %v", err)
		}
		logger.PrintIfVerbose("Waiting for scan to finish. scan running: " + fmt.Sprintf("%t", isRunning))
	}
	completedScanId = scanID
	return scanID
}

func getRunningScanId(t *testing.T) string {
	scanWrapper := wrappers.NewHTTPScansWrapper(scans)
	if runningScanId != "" {
		isRunning, _ := util.IsScanRunningOrQueued(scanWrapper, runningScanId)
		if isRunning {
			return runningScanId
		}
	}
	scanID, _ := createScanNoWait(t, Zip, Tags, getProjectNameForScanTests())
	runningScanId = scanID
	return scanID
}

func TestPRGithubDecorationSuccessCase(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
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

func TestPRGithubDecoration_WhenUseCodeRepositoryFlag_ShouldSuccess(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
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
	defer monkey.Unpatch((*wrappers.PRHTTPWrapper).PostPRDecoration)

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
	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
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

func TestPRGitlabDecoration_WhenUseCodeRepositoryFlag_ShouldSuccess(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
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

	monkey.Patch((*wrappers.PRHTTPWrapper).PostGitlabPRDecoration, func(*wrappers.PRHTTPWrapper, *wrappers.GitlabPRModel) (string, *wrappers.WebError, error) {
		return gitlabPRCommentCreated, nil, nil
	})
	defer monkey.Unpatch((*wrappers.PRHTTPWrapper).PostGitlabPRDecoration)

	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")

	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	assert.Equal(t, strings.Contains(stdoutString, gitlabPRCommentCreated), true, "Expected output: %s", gitlabPRCommentCreated, " | actual: ", stdoutString)

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

func TestPRAzureDecorationSuccessCase(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"azure",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
		flag(params.SCMTokenFlag),
		os.Getenv(prAzureToken),
		flag(params.NamespaceFlag),
		os.Getenv(prAzureOrganization),
		flag(params.AzureProjectFlag),
		os.Getenv(prAzureProject),
		flag(params.PRNumberFlag),
		os.Getenv(prAzureNumber),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")
}

func TestPRAzureDecoration_WhenUseCodeRepositoryFlag_ShouldSuccess(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"azure",
		flag(params.ScanIDFlag),
		getCompletedScanID(t),
		flag(params.SCMTokenFlag),
		os.Getenv(prAzureToken),
		flag(params.NamespaceFlag),
		os.Getenv(prAzureOrganization),
		flag(params.AzureProjectFlag),
		os.Getenv(prAzureProject),
		flag(params.PRNumberFlag),
		os.Getenv(prAzureNumber),
		flag(params.CodeRepositoryFlag),
		"https://azure.example.com",
	}

	monkey.Patch((*wrappers.PRHTTPWrapper).PostAzurePRDecoration, func(*wrappers.PRHTTPWrapper, *wrappers.AzurePRModel) (string, *wrappers.WebError, error) {
		return azurePRCommentCreated, nil, nil
	})
	defer monkey.Unpatch((*wrappers.PRHTTPWrapper).PostAzurePRDecoration)

	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(t, file)
	defer logger.SetOutput(os.Stdout)

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Error should be nil")

	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	assert.Equal(t, strings.Contains(stdoutString, azurePRCommentCreated), true, "Expected output: %s", azurePRCommentCreated)
}

func TestPRAzureDecorationFailure(t *testing.T) {

	args := []string{
		"utils",
		"pr",
		"azure",
		flag(params.ScanIDFlag),
		"fakeScanID",
		flag(params.SCMTokenFlag),
		os.Getenv(prAzureToken),
		flag(params.NamespaceFlag),
		os.Getenv(prAzureOrganization),
		flag(params.AzureProjectFlag),
		os.Getenv(prAzureProject),
		flag(params.PRNumberFlag),
		os.Getenv(prAzureNumber),
	}
	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, "scan not found")
}

func TestPRGithubDecoration_WhenScanIsRunning_ShouldAvoidPRDecorationCommand(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"github",
		flag(params.ScanIDFlag),
		getRunningScanId(t),
		flag(params.SCMTokenFlag),
		os.Getenv(prGithubToken),
		flag(params.NamespaceFlag),
		os.Getenv(prGithubNamespace),
		flag(params.PRNumberFlag),
		os.Getenv(prGithubNumber),
		flag(params.RepoNameFlag),
		os.Getenv(prGithubRepoName),
	}

	runPRTestForRunningScan(t, args)
}

func TestPRGitlabDecoration_WhenScanIsRunning_ShouldAvoidPRDecorationCommand(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"gitlab",
		flag(params.ScanIDFlag),
		getRunningScanId(t),
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

	runPRTestForRunningScan(t, args)
}

func TestPRAzureDecoration_WhenScanIsRunning_ShouldAvoidPRDecorationCommand(t *testing.T) {
	args := []string{
		"utils",
		"pr",
		"azure",
		flag(params.ScanIDFlag),
		getRunningScanId(t),
		flag(params.SCMTokenFlag),
		os.Getenv(prAzureToken),
		flag(params.NamespaceFlag),
		os.Getenv(prAzureOrganization),
		flag(params.AzureProjectFlag),
		os.Getenv(prAzureProject),
		flag(params.PRNumberFlag),
		os.Getenv(prAzureNumber),
	}

	runPRTestForRunningScan(t, args)
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

func runPRTestForRunningScan(t *testing.T, args []string) {
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
