//go:build !integration

package commands

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	exitCodes "github.com/checkmarx/ast-cli/internal/constants/exit-codes"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	unknownFlag                            = "unknown flag: --chibutero"
	blankSpace                             = " "
	errorMissingBranch                     = "Failed creating a scan: Please provide a branch"
	dummyGitlabRepo                        = "https://gitlab.com/dummy-org/gitlab-dummy"
	dummyRepo                              = "https://github.com/dummyuser/dummy_project.git"
	dummyRepoWithToken                     = "https://token@github.com/dummyuser/dummy_project"
	dummyRepoWithTokenAndUsername          = "https://username:token@github.com/dummyuser/dummy_project"
	dummyShortenedRepoWithToken            = "token@github.com/dummyuser/dummy_project"
	dummyShortenedRepoWithTokenAndUsername = "username:token@github.com/dummyuser/dummy_project"
	dummyShortenedGithubRepo               = "github.com/dummyuser/dummy_project.git"
	dummyToken                             = "dummyToken"
	dummySSHRepo                           = "git@github.com:dummyRepo/dummyProject.git"
	errorSourceBadFormat                   = "Failed creating a scan: Input in bad format: Sources input has bad format: "
	scaPathError                           = "ScaResolver error: exec: \"resolver\": executable file not found in "
	fileSourceFlag                         = "--file"
	fileSourceValueEmpty                   = "data/empty.Dockerfile"
	fileSourceValue                        = "data/Dockerfile"
	fileSourceIncorrectValue               = "data/source.zip"
	fileSourceIncorrectValueError          = "data/source.zip. Provided file is not supported by kics"
	fileSourceError                        = "flag needs an argument: --file"
	engineFlag                             = "--engine"
	engineValue                            = "docker"
	invalidEngineValue                     = "invalidengine"
	engineError                            = "flag needs an argument: --engine"
	additionalParamsFlag                   = "--additional-params"
	additionalParamsValue                  = "-v"
	additionalParamsError                  = "flag needs an argument: --additional-params"
	scanCommand                            = "scan"
	kicsRealtimeCommand                    = "kics-realtime"
	kicsPresetIDIncorrectValueError        = "Invalid value for --iac-security-preset-id flag. Must be a valid UUID."
	InvalidEngineMessage                   = "Please verify if engine is installed"
	SCSScoreCardError                      = "SCS scan failed to start: Scorecard scan is missing required flags, please include in the ast-cli arguments: " +
		"--scs-repo-url your_repo_url --scs-repo-token your_repo_token"
	outputFileName                = "test_output.log"
	noUpdatesForExistingProject   = "No tags or branch to update. Skipping project update."
	ScaResolverZipNotSupportedErr = "Scanning Zip files is not supported by ScaResolver.Please use non-zip source"
)

func TestScanHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "scan")
}

func TestScanCreateHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "scan", "create")
}

func TestScanNoSub(t *testing.T) {
	execCmdNilAssertion(t, "scan")
}

func TestRunGetScanByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "show", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetScanByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "show", "--scan-id", "MOCK")
}

func TestRunDeleteScanByIdCommandNoScanID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "delete")
	assert.Assert(t, err.Error() == "Failed deleting a scan: Please provide at least one scan ID")
}

func TestRunDeleteByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "delete", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteScanByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "delete", "--scan-id", "MOCK")
}

func TestRunCancelScanByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "cancel", "--scan-id", "MOCK")
}

func TestRunGetAllCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "list")
}

func TestRunGetAllCommandList(t *testing.T) {
	execCmdNilAssertion(t, "scan", "list", "--format", "list")
}

func TestRunGetAllCommandLimitList(t *testing.T) {
	execCmdNilAssertion(t, "scan", "list", "--format", "list", "--filter", "limit=40")
}

func TestRunGetAllCommandOffsetList(t *testing.T) {
	execCmdNilAssertion(t, "scan", "list", "--format", "list", "--filter", "offset=0")
}

func TestRunGetAllCommandStatusesList(t *testing.T) {
	execCmdNilAssertion(
		t,
		"scan",
		"list",
		"--format",
		"list",
		"--filter",
		"statuses=Failed;Completed;Running,limit=500",
	)
}

func TestRunGetAllCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "list", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunTagsCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "tags")
}

func TestCreateScan(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch")
}

func TestCreateScanFromFolder_ContainersImagesAndDefaultScanTypes_ScanCreatedSuccessfully(t *testing.T) {
	clearFlags()
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch", "--container-images", "image1:latest,image2:tag"}
	execCmdNilAssertion(t, append(baseArgs, "-s", blankSpace+"."+blankSpace)...)
}

func TestCreateScanFromZip_ContainersImagesAndDefaultScanTypes_ScanCreatedSuccessfully(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "data/sources.zip", "-b", "dummy_branch", "--container-images", "image1:latest,image2:tag")
}

func TestCreateScanFromZip_ContainerTypeAndFilterFlags_ScanCreatedSuccessfully(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--scan-types", "container-security", "-s", "data/sources.zip", "-b", "dummy_branch", "--file-filter", "!.java")
}

func TestCreateScanFromFolder_InvalidContainersImagesAndNoContainerScanType_ScanCreatedSuccessfully(t *testing.T) {
	// When no container scan type is provided, we will ignore the container images flag and its value
	clearFlags()
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch", "--scan-types", "sast", "--container-images", "image1,image2:tag"}
	execCmdNilAssertion(t, append(baseArgs, "-s", blankSpace+"."+blankSpace)...)
}

func TestCreateScanFromFolder_ContainerImagesFlagWithoutValue_FailCreatingScan(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--container-images")
	assert.Assert(t, err.Error() == "flag needs an argument: --container-images")
}

func TestCreateScanFromFolder_InvalidContainerImageFormat_FailCreatingScan(t *testing.T) {
	clearFlags()
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch", "--container-images", "image1,image2:tag", "--scan-types", "containers", "--containers-local-resolution"}
	err := execCmdNotNilAssertion(t, append(baseArgs, "-s", blankSpace+"."+blankSpace)...)
	assert.Assert(t, err.Error() == "Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag> or <image-name>.tar")
}

func TestCreateScanWithThreshold_ShouldSuccess(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--scan-types", "sast", "--threshold", "sca-low=1 ; sast-medium=2")
}

func TestScanCreate_ApplicationNameIsNotExactMatch_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "non-existing-project", "--application-name", "MOC", "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_ExistingProjectAndApplicationWithNoPermission_ShouldCreateScan(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.ApplicationDoesntExist, "-s", dummyRepo, "-b", "dummy_branch")
}

func TestScanCreate_ExistingApplicationWithNoPermission_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "NewProject", "--application-name", mock.NoPermissionApp, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_OnReceivingHttpBadRequestStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "non-existing-project", "--application-name", mock.FakeBadRequest400, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == errorConstants.FailedToGetApplication)
}

func TestScanCreate_OnReceivingHttpInternalServerErrorStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "non-existing-project",
		"--application-name", mock.FakeInternalServerError500, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == errorConstants.FailedToGetApplication)
}

func TestCreateScanSourceDirectory(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "-s", "data", "--file-filter", "!.java")...)
}

func TestCreateScanSourceFile(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "data/sources.zip", "-b", "dummy_branch")
}

func TestCreateScanWithTrimmedSources(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "-s", blankSpace+"."+blankSpace)...)
	execCmdNilAssertion(t, append(baseArgs, "-s", blankSpace+"data/"+blankSpace)...)
	execCmdNilAssertion(t, append(baseArgs, "-s", blankSpace+dummyRepo+blankSpace)...)
}

func TestCreateScanWrongFormatSource(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "-s", "invalidSource")...)
	assert.Assert(t, err.Error() == errorSourceBadFormat+"invalidSource")

	err = execCmdNotNilAssertion(t, append(baseArgs, "-s", "http:")...)
	assert.Assert(t, err.Error() == errorSourceBadFormat+"http:")
}

func TestCreateScanWithScaResolver(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", "data", "-b", "dummy_branch"}
	execCmdNilAssertion(
		t,
		append(
			baseArgs,
			"--sca-resolver",
			viper.GetString(resolverEnvVar),
			"-f",
			"!ScaResolver",
			"--sca-resolver-params",
			"-q",
		)...,
	)
}

func TestCreateScanWithScaResolverFailed(t *testing.T) {
	baseArgs := []string{
		"scan",
		"create",
		"--project-name",
		"MOCK",
		"-s",
		"data",
		"-b",
		"dummy_branch",
		"--sca-resolver",
		"resolver",
	}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Assert(t, strings.Contains(err.Error(), scaPathError), err.Error())
}

func TestCreateScanWithScaResolverParamsWrong(t *testing.T) {
	tests := []struct {
		name              string
		sourceDir         string
		scaResolver       string
		scaResolverParams string
		projectName       string
		expectedError     string
	}{
		{
			name:              "ScaResolver wrong scaResolver path",
			sourceDir:         "/sourceDir",
			scaResolver:       "./ScaResolver",
			scaResolverParams: "params",
			projectName:       "ProjectName",
			expectedError:     "/ScaResolver: no such file or directory",
		},
		{
			name:              "Invalid scaResolverParams format",
			sourceDir:         "/sourceDir",
			scaResolver:       "./ScaResolver",
			scaResolverParams: "\"unclosed quote",
			projectName:       "ProjectName",
			expectedError:     "/ScaResolver: no such file or directory", // Actual error from command execution
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := runScaResolver(tt.sourceDir, tt.scaResolver, tt.scaResolverParams, tt.projectName)
			assert.Assert(t, strings.Contains(err.Error(), tt.expectedError), err.Error())
		})
	}
}

func TestCreateScanWithScaResolverNoScaResolver(t *testing.T) {
	var sourceDir = "/sourceDir"
	var scaResolver = ""
	var scaResolverParams = "params"
	var projectName = "ProjectName"
	err := runScaResolver(sourceDir, scaResolver, scaResolverParams, projectName)
	assert.Assert(t, err == nil)
}

func TestCreateScanWithScanTypes(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sast")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "iac-security")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sca")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sast,api-security")...)

	baseArgs = append(baseArgs, flag(commonParams.ScanTypes), "scs",
		flag(commonParams.SCSRepoURLFlag), "dummyURL",
		flag(commonParams.SCSRepoTokenFlag), "dummyToken")
	execCmdNilAssertion(t, baseArgs...)
}

func TestScanCreate_KicsScannerFail_ReturnCorrectKicsExitCodeAndErrorMessage(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "fake-kics-scanner-fail", "-s", dummyRepo, "-b", "dummy_branch"}
	err := execCmdNotNilAssertion(t, append(baseArgs, "--scan-types", Kics)...)
	assertAstError(t, err, "scan did not complete successfully", exitCodes.KicsEngineFailedExitCode)
}

func TestScanCreate_MultipleScannersFail_ReturnGeneralExitCodeAndErrorMessage(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "fake-multiple-scanner-fails", "-s", dummyRepo, "-b", "dummy_branch"}
	baseArgs = append(baseArgs, "--scan-types", fmt.Sprintf("%s,%s", Kics, Sca))
	err := execCmdNotNilAssertion(t, baseArgs...)
	assertAstError(t, err, "scan did not complete successfully", exitCodes.MultipleEnginesFailedExitCode)
}

func TestScanCreate_ScaScannersFailPartialScan_ReturnScaExitCodeAndErrorMessage(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "fake-sca-fail-partial", "-s", dummyRepo, "-b", "dummy_branch"}
	baseArgs = append(baseArgs, "--scan-types", Sca)
	err := execCmdNotNilAssertion(t, baseArgs...)
	assertAstError(t, err, "scan completed partially", exitCodes.ScaEngineFailedExitCode)
}

func TestScanCreate_MultipleScannersDifferentStatusesOnlyKicsFail_ReturnKicsExitCodeAndErrorMessage(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "fake-kics-fail-sast-canceled", "-s", dummyRepo, "-b", "dummy_branch"}
	baseArgs = append(baseArgs, "--scan-types", fmt.Sprintf("%s,%s,%s", Sca, Sast, Kics))
	err := execCmdNotNilAssertion(t, baseArgs...)
	assertAstError(t, err, "scan did not complete successfully", exitCodes.KicsEngineFailedExitCode)
}

func assertAstError(t *testing.T, err error, expectedErrorMessage string, expectedExitCode int) {
	var e *wrappers.AstError
	if errors.As(err, &e) {
		assert.Equal(t, e.Error(), expectedErrorMessage)
		assert.Equal(t, e.Code, expectedExitCode)
	} else {
		assert.Assert(t, false, "Error is not of type AstError")
	}
}

func TestCreateScanWithNoFilteredProjects(t *testing.T) {
	baseArgs := []string{"scan", "create", "-s", dummyRepo, "-b", "dummy_branch"}
	// Cover "createProject" when no project is filtered when finding the provided project
	execCmdNilAssertion(t, append(baseArgs, "--project-name", "MOCK-NO-FILTERED-PROJECTS")...)
}

func TestCreateScanWithTags(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "--tags", "dummy_tag:sub_dummy_tag")...)
}

func TestCreateScanWithPresetName(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "--sast-preset-name", "High and Low")...)
}

func TestCreateScanExcludeGitFolder(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", "../..", "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "-f", "!.git")...)
}

func TestCreateScanExcludeGitFolderChildren(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", "../..", "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "-f", "!.git/HEAD")...)
}

func TestCreateScanBranches(t *testing.T) {
	// Test Missing branch either in flag and in the environment variable
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo)
	assert.Assert(t, err.Error() == errorMissingBranch)

	// Bind cx_branch environment variable
	_ = viper.BindEnv("cx_branch", "CX_BRANCH")
	viper.SetDefault("cx_branch", "branch_from_environment_variable")
	assert.Equal(t, viper.GetString("cx_branch"), "branch_from_environment_variable")

	// Test branch from environment variable. Since the cx_branch is bind the scan must run successfully without a branch flag defined
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo)

	// Test missing branch value
	err = execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b")
	assert.Assert(t, err.Error() == "flag needs an argument: 'b' in -b")

	// Test empty branch value
	err = execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "")
	assert.Assert(t, err.Error() == errorMissingBranch)

	// Test defined branch value
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "branch_defined")
}

func TestCreateScan_WhenProjectNotExistsAndInvalidGroup_ShouldFail(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan", "create", "--project-name", "newProject", "-s", ".", "--branch", "main", "--project-groups", "invalidGroup",
	)
	assert.Assert(t, err.Error() == "Failed updating a project: Failed finding groups: [invalidGroup]", "\n the received error is:", err.Error())
}

func TestCreateScan_WhenProjectNotExists_ShouldCreateProjectAndAssignGroup(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)

	baseArgs := []string{"scan", "create", "--project-name", "newProject", "-s", ".", "--branch", "main", "--project-groups", "existsGroup1", "--debug"}
	execCmdNilAssertion(
		t,
		baseArgs...,
	)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, "Updating project groups"), true, "Expected output: %s", "Updating project groups")
}

func TestCreateScan_WhenProjectNotExists_ShouldCreateProjectAndAssociateApplication(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)

	baseArgs := []string{"scan", "create", "--project-name", "newProject", "-s", ".", "--branch", "main", "--application-name", mock.ExistingApplication, "--debug"}
	execCmdNilAssertion(
		t,
		baseArgs...,
	)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, "application association done successfully"), true, "Expected output: %s", "application association done successfully")
}

func TestScanWorkflowMissingID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "workflow")
	assert.Error(t, err, "Please provide a scan ID", err.Error())
}

func TestCreateScanMissingSSHValue(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", "../..", "-b", "dummy_branch"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", " ")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())
}

func TestCreateScanInvalidSSHSource(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}

	// zip file with ssh
	err := execCmdNotNilAssertion(t, append(baseArgs, "-s", "data/sources.zip", "--ssh-key", "dummy_key")...)
	assert.Error(t, err, invalidSSHSource, err.Error())

	// directory with ssh
	err = execCmdNotNilAssertion(t, append(baseArgs, "-s", "../..", "--ssh-key", "dummy_key")...)
	assert.Error(t, err, invalidSSHSource, err.Error())

	// http url with ssh
	err = execCmdNotNilAssertion(t, append(baseArgs, "-s", dummyRepo, "--ssh-key", "dummy_key")...)
	assert.Error(t, err, invalidSSHSource, err.Error())
}

func TestCreateScanWrongSSHKeyPath(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "-s", dummySSHRepo, "--ssh-key", "dummy_key")...)

	expectedMessages := []string{
		"open dummy_key: The system cannot find the file specified.",
		"open dummy_key: no such file or directory",
	}

	assert.Assert(t, utils.Contains(expectedMessages, err.Error()))
}

func TestCreateScanWithSSHKey(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}

	execCmdNilAssertion(t, append(baseArgs, "-s", dummySSHRepo, "--ssh-key", "data/sources.zip")...)
}

func TestScanWorkFlowWithSastFilter(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "sastFilterMock", "-b", "dummy_branch", "-s", dummyRepo, "--sast-filter", "!*.go"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithKicsFilter(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsFilterMock", "-b", "dummy_branch", "-s", dummyRepo, "--iac-security-filter", "!Dockerfile"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithKicsFilterDeprecated(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsFilterMock", "-b", "dummy_branch", "-s", dummyRepo, "--kics-filter", "!Dockerfile"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithKicsPlatforms(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsPlatformsMock", "-b", "dummy_branch", "-s", dummyRepo, "--iac-security-platforms", "Dockerfile"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithKicsPlatformsDeprecated(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsPlatformsMock", "-b", "dummy_branch", "-s", dummyRepo, "--kics-platforms", "Dockerfile"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithKicsPresetID(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsPresetIDMock", "-b", "dummy_branch", "-s", dummyRepo, "--iac-security-preset-id", "4801dea3-b365-4934-a810-ebf481f646c3"}
	err := executeTestCommand(createASTTestCommand(), baseArgs...)
	assert.NilError(t, err)
}

func TestScanWorkFlowWithInvalidKicsPresetID(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "kicsPresetIDMock", "-b", "dummy_branch", "-s", dummyRepo, "--iac-security-preset-id", "invalid uuid"}
	err := executeTestCommand(createASTTestCommand(), baseArgs...)
	assert.Error(t, err, kicsPresetIDIncorrectValueError, err.Error())
}

func TestScanWorkFlowWithScaFilter(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "scaFilterMock", "-b", "dummy_branch", "-s", dummyRepo, "--sca-filter", "!jQuery"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateScanFilterZipFile(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch"}

	execCmdNilAssertion(t, append(baseArgs, "-s", "data/sources.zip", "--file-filter", "!.java")...)
}

func TestCreateRealtimeKics(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateRealtimeKicsMissingFile(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, fileSourceError, err.Error())
}

func TestCreateRealtimeKicsInvalidFile(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceIncorrectValue}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, fileSourceIncorrectValueError, err.Error())
}

func TestCreateRealtimeKicsWithEngine(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue, engineFlag, engineValue}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateRealtimeKicsInvalidEngine(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue, engineFlag, invalidEngineValue}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, InvalidEngineMessage, err.Error())
}

func TestCreateRealtimeKicsMissingEngine(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue, engineFlag}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, engineError, err.Error())
}

func TestCreateRealtimeKicsWithAdditionalParams(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue, engineFlag, engineValue, additionalParamsFlag, additionalParamsValue}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateRealtimeKicsMissingAdditionalParams(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValue, additionalParamsFlag}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, additionalParamsError, err.Error())
}

func TestCreateRealtimeKicsFailedScan(t *testing.T) {
	baseArgs := []string{scanCommand, kicsRealtimeCommand, fileSourceFlag, fileSourceValueEmpty}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateScanResubmit(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--resubmit")
}

func TestCreateScanResubmitWithScanTypes(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--scan-types", "sast,iac-security,sca", "--debug", "--resubmit")
}

func TestCreateScanWithPrimaryBranchFlag_Passed(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary")
}

func TestCreateScanWithPrimaryBranchFlagBooleanValueTrue_Failed(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()
	os.Args = []string{
		"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary=true",
	}
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary=true")
	assert.ErrorContains(t, err, "invalid value for --branch-primary flag", err.Error())
}

func TestCreateScanWithPrimaryBranchFlagBooleanValueFalse_Failed(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()
	os.Args = []string{
		"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary=false",
	}
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary=false")
	assert.ErrorContains(t, err, "invalid value for --branch-primary flag", err.Error())
}

func TestCreateScanWithPrimaryBranchFlagStringValue_Should_Fail(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--debug", "--branch-primary=string")
	assert.ErrorContains(t, err, "invalid argument \"string\"", err.Error())
}

func Test_parseThresholdSuccess(t *testing.T) {
	want := make(map[string]int)
	want["iac-security-low"] = 1
	threshold := " KICS - LoW=1"
	if got := parseThreshold(threshold); !reflect.DeepEqual(got, want) {
		t.Errorf("parseThreshold() = %v, want %v", got, want)
	}
}
func Test_parseThresholdsSuccess(t *testing.T) {
	want := make(map[string]int)
	want["sast-high"] = 1
	want["sast-medium"] = 1
	want["sca-high"] = 1
	threshold := "sast-high=1; sast-medium=1; sca-high=1"
	if got := parseThreshold(threshold); !reflect.DeepEqual(got, want) {
		t.Errorf("parseThreshold() = %v, want %v", got, want)
	}
}
func Test_parseThresholdParseError(t *testing.T) {
	want := make(map[string]int)
	threshold := " KICS - LoW=error"
	if got := parseThreshold(threshold); !reflect.DeepEqual(got, want) {
		t.Errorf("parseThreshold() = %v, want %v", got, want)
	}
}
func TestCreateScanProjectTags(t *testing.T) {
	execCmdNilAssertion(t, scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--project-tags", "test", "--debug")
}

func TestCreateScan_WhenProjectExists_ShouldIgnoreGroups(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)
	baseArgs := []string{scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--debug", "--project-groups", "anyProjectGroup"}
	execCmdNilAssertion(t, baseArgs...)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, noUpdatesForExistingProject), true, "Expected output: %s", noUpdatesForExistingProject)
}

func TestCreateScan_WhenProjectExists_ShouldIgnoreApplication(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)
	baseArgs := []string{scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--debug", "--application-name", "anyApplication"}
	execCmdNilAssertion(t, baseArgs...)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, noUpdatesForExistingProject), true, "Expected output: %s", noUpdatesForExistingProject)
}
func TestScanCreateLastSastScanTimeWithInvalidValue(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--sca-exploitable-path", "true", "--sca-last-sast-scan-time", "notaniteger"}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.ErrorContains(t, err, "Invalid value for --sca-last-sast-scan-time flag", err.Error())
}

func TestScanCreateExploitablePathWithWrongValue(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--sca-exploitable-path", "nottrueorfalse"}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.ErrorContains(t, err, "Invalid value for --sca-exploitable-path flag", err.Error())
}

func TestScanCreateExploitablePathWithoutSAST(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--scan-types", "sca", "--sca-exploitable-path", "true"}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.ErrorContains(t, err, "you must enable SAST scan type", err.Error())
}

func TestScanCreateExploitablePath(t *testing.T) {
	execCmdNilAssertion(t, scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--sca-exploitable-path", "true", "--sca-last-sast-scan-time", "1", "--debug")
}

func TestScanCreateProjectPrivatePackage(t *testing.T) {
	execCmdNilAssertion(t, scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--project-private-package", "true", "--debug")
}

func TestScanCreateProjectPrivatePackageWrongValue(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--project-private-package", "nottrueorfalse"}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.ErrorContains(t, err, "Invalid value for --project-private-package flag", err.Error())
}

func TestScanCreateScaPrivatePackageVersion(t *testing.T) {
	execCmdNilAssertion(t, scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--sca-private-package-version", "1.0.0", "--debug")
}
func TestAddScaScan(t *testing.T) {
	var resubmitConfig []wrappers.Config

	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.ScaFilterFlag, "", "Filter for SCA scan")
	cmdCommand.PersistentFlags().String(commonParams.LastSastScanTime, "", "Last SAST scan time")
	cmdCommand.PersistentFlags().String(commonParams.ScaPrivatePackageVersionFlag, "", "Private package version")
	cmdCommand.PersistentFlags().String(commonParams.ExploitablePathFlag, "", "Exploitable path")

	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.ScaFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.LastSastScanTime, "1")
	_ = cmdCommand.Flags().Set(commonParams.ScaPrivatePackageVersionFlag, "1.1.1")
	_ = cmdCommand.Flags().Set(commonParams.ExploitablePathFlag, "true")

	result := addScaScan(cmdCommand, resubmitConfig, false)
	scaConfig := wrappers.ScaConfig{
		Filter:                "test",
		ExploitablePath:       "true",
		LastSastScanTime:      "1",
		PrivatePackageVersion: "1.1.1",
		SBom:                  "false",
	}
	scaMapConfig := make(map[string]interface{})
	scaMapConfig[resultsMapType] = commonParams.ScaType
	scaMapConfig[resultsMapValue] = &scaConfig

	if !reflect.DeepEqual(result, scaMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scaMapConfig, result)
	}
}
func TestAddSCSScan_ResubmitWithOutScorecardFlags_ShouldPass(t *testing.T) {
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
	}
	cmdCommand.PersistentFlags().String(commonParams.ScanTypes, "", "Scan types")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "SCS Repo Token")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "SCS Repo URL")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.ScanTypes, commonParams.ScsType)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, "")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, "")

	resubmitConfig := []wrappers.Config{
		{
			Type: commonParams.ScsType,
			Value: map[string]interface{}{
				configTwoms:      trueString,
				ScsScoreCardType: falseString,
			},
		},
	}

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	expectedConfig := wrappers.SCSConfig{
		Twoms:     trueString,
		Scorecard: falseString,
	}

	expectedMapConfig := make(map[string]interface{})
	expectedMapConfig[resultsMapType] = commonParams.MicroEnginesType
	expectedMapConfig[resultsMapValue] = &expectedConfig

	if !reflect.DeepEqual(result, expectedMapConfig) {
		t.Errorf("Expected %+v, but got %+v", expectedMapConfig, result)
	}
}

func TestAddSCSScan_ResubmitWithScorecardFlags_ShouldPass(t *testing.T) {
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
	}
	cmdCommand.PersistentFlags().String(commonParams.ScanTypes, "", "Scan types")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "SCS Repo Token")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "SCS Repo URL")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.ScanTypes, commonParams.ScsType)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyRepo)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)

	resubmitConfig := []wrappers.Config{
		{
			Type: commonParams.ScsType,
			Value: map[string]interface{}{
				configTwoms:      trueString,
				ScsScoreCardType: trueString,
			},
		},
	}

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	expectedConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: trueString,
		RepoToken: dummyToken,
		RepoURL:   dummyRepo,
	}

	expectedMapConfig := make(map[string]interface{})
	expectedMapConfig[resultsMapType] = commonParams.MicroEnginesType
	expectedMapConfig[resultsMapValue] = &expectedConfig

	if !reflect.DeepEqual(result, expectedMapConfig) {
		t.Errorf("Expected %+v, but got %+v", expectedMapConfig, result)
	}
}

func TestAddSastScan_WithFastScanFlag_ShouldPass(t *testing.T) {
	var resubmitConfig []wrappers.Config

	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project with SAST fast scan configuration`,
	}

	cmdCommand.PersistentFlags().String(commonParams.PresetName, "", "Preset name")
	cmdCommand.PersistentFlags().String(commonParams.SastFilterFlag, "", "Filter for SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.IncrementalSast, false, "Incremental SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.SastFastScanFlag, false, "Enable SAST Fast Scan")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.PresetName, "test")
	_ = cmdCommand.Flags().Set(commonParams.SastFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.IncrementalSast, "true")
	_ = cmdCommand.Flags().Set(commonParams.SastFastScanFlag, "true")

	result := addSastScan(cmdCommand, resubmitConfig)

	sastConfig := wrappers.SastConfig{
		PresetName:   "test",
		Filter:       "test",
		Incremental:  "true",
		FastScanMode: "true",
	}
	sastMapConfig := make(map[string]interface{})
	sastMapConfig[resultsMapType] = commonParams.SastType
	sastMapConfig[resultsMapValue] = &sastConfig

	if !reflect.DeepEqual(result, sastMapConfig) {
		t.Errorf("Expected %+v, but got %+v", sastMapConfig, result)
	}
}

func TestAddSastScan_WithLightQueryAndRecommendedExclusions_ShouldPass(t *testing.T) {
	var resubmitConfig []wrappers.Config

	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project with SAST fast scan configuration`,
	}

	cmdCommand.PersistentFlags().String(commonParams.PresetName, "", "Preset name")
	cmdCommand.PersistentFlags().String(commonParams.SastFilterFlag, "", "Filter for SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.IncrementalSast, false, "Incremental SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.SastFastScanFlag, false, "Enable SAST Fast Scan")
	cmdCommand.PersistentFlags().Bool(commonParams.SastLightQueriesFlag, false, "Enable SAST Light Queries")
	cmdCommand.PersistentFlags().Bool(commonParams.SastRecommendedExclusionsFlags, false, "Enable SAST Recommended Exclusions")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.PresetName, "test")
	_ = cmdCommand.Flags().Set(commonParams.SastFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.IncrementalSast, "false")
	_ = cmdCommand.Flags().Set(commonParams.SastFastScanFlag, "false")
	_ = cmdCommand.Flags().Set(commonParams.SastLightQueriesFlag, "true")
	_ = cmdCommand.Flags().Set(commonParams.SastRecommendedExclusionsFlags, "true")

	result := addSastScan(cmdCommand, resubmitConfig)

	sastConfig := wrappers.SastConfig{
		PresetName:            "test",
		Filter:                "test",
		Incremental:           "false",
		FastScanMode:          "false",
		LightQueries:          "true",
		RecommendedExclusions: "true",
	}
	sastMapConfig := make(map[string]interface{})
	sastMapConfig[resultsMapType] = commonParams.SastType
	sastMapConfig[resultsMapValue] = &sastConfig

	if !reflect.DeepEqual(result, sastMapConfig) {
		t.Errorf("Expected %+v, but got %+v", sastMapConfig, result)
	}
}

func TestCreateScanWithFastScanFlagIncorrectCase(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "--branch", "b", "--scan-types", "sast", "--file-source", "."}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--SAST-FAST-SCAN", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --SAST-FAST-SCAN", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--Sast-Fast-Scan", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --Sast-Fast-Scan", err.Error())
}

func TestCreateScanWithLightQueryIncorrectCase(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "--branch", "b", "--scan-types", "sast", "--file-source", "."}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--SAST-LIGHT-QUERIES", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --SAST-LIGHT-QUERIES", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--Sast-Light-Queries", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --Sast-Light-Queries", err.Error())
}

func TestCreateScanWithRecommendedExclusionsIncorrectCase(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "--branch", "b", "--scan-types", "sast", "--file-source", "."}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--SAST-RECOMMENDED-EXCLUSIONS", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --SAST-RECOMMENDED-EXCLUSIONS", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--Sast-Recommended-Exclusions", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --Sast-Recommended-Exclusions", err.Error())
}

func TestAddSastScan(t *testing.T) {
	var resubmitConfig []wrappers.Config

	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}

	cmdCommand.PersistentFlags().String(commonParams.PresetName, "", "Preset name")
	cmdCommand.PersistentFlags().String(commonParams.SastFilterFlag, "", "Filter for SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.IncrementalSast, false, "Incremental SAST scan")
	cmdCommand.PersistentFlags().Bool(commonParams.SastFastScanFlag, false, "Enable SAST Fast Scan")
	cmdCommand.PersistentFlags().Bool(commonParams.SastLightQueriesFlag, false, "Enable SAST Light Queries")
	cmdCommand.PersistentFlags().Bool(commonParams.SastRecommendedExclusionsFlags, false, "Enable SAST Recommended Exclusions")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.PresetName, "test")
	_ = cmdCommand.Flags().Set(commonParams.SastFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.IncrementalSast, "true")
	_ = cmdCommand.Flags().Set(commonParams.SastLightQueriesFlag, "true")
	_ = cmdCommand.Flags().Set(commonParams.SastRecommendedExclusionsFlags, "true")

	result := addSastScan(cmdCommand, resubmitConfig)

	sastConfig := wrappers.SastConfig{
		PresetName:            "test",
		Filter:                "test",
		Incremental:           "true",
		FastScanMode:          "",
		LightQueries:          "true",
		RecommendedExclusions: "true",
	}
	sastMapConfig := make(map[string]interface{})
	sastMapConfig[resultsMapType] = commonParams.SastType

	sastMapConfig[resultsMapValue] = &sastConfig

	if !reflect.DeepEqual(result, sastMapConfig) {
		t.Errorf("Expected %+v, but got %+v", sastMapConfig, result)
	}
}

func TestAddKicsScan(t *testing.T) {
	var resubmitConfig []wrappers.Config

	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}

	cmdCommand.PersistentFlags().String(commonParams.KicsFilterFlag, "", "Filter for KICS scan")
	cmdCommand.PersistentFlags().Bool(commonParams.IacsPlatformsFlag, false, "IaC platforms")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.KicsFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.IacsPlatformsFlag, "true")

	result := addKicsScan(cmdCommand, resubmitConfig)

	kicsConfig := wrappers.KicsConfig{
		Filter: "test",
	}
	kicsMapConfig := make(map[string]interface{})
	kicsMapConfig[resultsMapType] = commonParams.KicsType

	kicsMapConfig[resultsMapValue] = &kicsConfig

	if !reflect.DeepEqual(result, kicsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", kicsMapConfig, result)
	}
}
func TestCreateScanProjectTagsCheckResendToScan(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "sastFilterMock", "-b", "dummy_branch", "-s", dummyRepo, "--project-tags", "SEG", "--debug"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateScan_WithSCSScorecard_ShouldFail(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan",
		"create",
		"--project-name",
		"MOCK",
		"-s",
		dummyRepo,
		"-b",
		"dummy_branch",
		"--scan-types",
		"scs",
		"--scs-engines",
		"scorecard",
	)
	assert.Assert(t, err.Error() == SCSScoreCardError)
}

func TestCreateScan_WithSCSSecretDetectionAndScorecard_scsMapHasBoth(t *testing.T) {
	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyRepo)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyRepo,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithoutSCSSecretDetection_scsMapNoSecretDetection(t *testing.T) {
	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyRepo)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, false)

	scsConfig := wrappers.SCSConfig{
		Twoms:     "",
		Scorecard: "true",
		RepoURL:   dummyRepo,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetection_scsMapHasSecretDetection(t *testing.T) {
	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection")

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	scsConfig := wrappers.SCSConfig{
		Twoms: "true",
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardWithScanTypesAndNoScorecardFlags_scsMapHasSecretDetection(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.ScanTypeFlag, "scs", "")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.ScanTypeFlag, "scs")

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	scsConfig := wrappers.SCSConfig{
		Twoms: "true",
	}

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, ScsRepoWarningMsg) {
		t.Errorf("Expected output to contain %q, but got %q", ScsRepoWarningMsg, output)
	}

	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepo_scsMapHasBoth(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyShortenedGithubRepo)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to not contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyShortenedGithubRepo,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepoWithTokenInURL_scsMapHasBoth(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyShortenedRepoWithToken)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to not contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyShortenedRepoWithToken,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardGithubRepoWithTokenInURL_scsMapHasBoth(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyRepoWithToken)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to not contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyRepoWithToken,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardGithubRepoWithTokenAndUsernameInURL_scsMapHasBoth(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyRepoWithTokenAndUsername)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to not contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyRepoWithTokenAndUsername,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepoWithTokenAndUsernameInURL_scsMapHasBoth(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyShortenedRepoWithTokenAndUsername)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to not contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyShortenedRepoWithTokenAndUsername,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardGitLabRepo_scsMapHasSecretDetection(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummyGitlabRepo)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "",
		RepoURL:   "",
		RepoToken: "",
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScan_WithSCSSecretDetectionAndScorecardGitSSHRepo_scsMapHasSecretDetection(t *testing.T) {
	// Create a pipe for capturing stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w // Redirecting stdout to the pipe

	var resubmitConfig []wrappers.Config
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoTokenFlag, "", "GitHub token to be used with SCS engines")
	cmdCommand.PersistentFlags().String(commonParams.SCSRepoURLFlag, "", "GitHub url to be used with SCS engines")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection,scorecard")
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoTokenFlag, dummyToken)
	_ = cmdCommand.Flags().Set(commonParams.SCSRepoURLFlag, dummySSHRepo)

	result, _ := addSCSScan(cmdCommand, resubmitConfig, true)

	// Close the writer to signal that we are done capturing the output
	w.Close()

	// Read from the pipe (stdout)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r) // Copy the captured output to a buffer
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, ScsScorecardUnsupportedHostWarningMsg) {
		t.Errorf("Expected output to contain %q, but got %q", ScsScorecardUnsupportedHostWarningMsg, output)
	}

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "",
		RepoURL:   "",
		RepoToken: "",
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func Test_isDirFiltered(t *testing.T) {
	type args struct {
		filename string
		filters  []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "WhenUserDefinedExcludedFolder_ReturnIsFilteredTrue",
			args: args{
				filename: "user-folder-to-exclude",
				filters:  append(commonParams.BaseExcludeFilters, "!user-folder-to-exclude"),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "WhenUserDefinedExcludedFolder_DoesNotAffectOtherFolders_ReturnIsFilteredFalse",
			args: args{
				filename: "some-folder",
				filters:  append(commonParams.BaseExcludeFilters, "!exclude-other-folder"),
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "WhenFolderIsNotExcluded_ReturnIsFilteredFalse",
			args: args{
				filename: "some-folder-name",
				filters:  commonParams.BaseExcludeFilters,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "WhenDefaultFolderIsExcluded_ReturnIsFilteredTrue",
			args: args{
				filename: ".vs",
				filters:  commonParams.BaseExcludeFilters,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "WhenNodeModulesExcluded_ReturnIsFilteredTrue",
			args: args{
				filename: "node_modules",
				filters:  commonParams.BaseExcludeFilters,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := isDirFiltered(ttt.args.filename, ttt.args.filters)
			if (err != nil) != ttt.wantErr {
				t.Errorf("isDirFiltered() error = %v, wantErr %v", err, ttt.wantErr)
				return
			}
			if got != ttt.want {
				t.Errorf("isDirFiltered() got = %v, want %v", got, ttt.want)
			}
		})
	}
}

func Test_parseThresholdLimit(t *testing.T) {
	type args struct {
		limit string
	}
	tests := []struct {
		name           string
		args           args
		wantEngineName string
		wantIntLimit   int
		wantErr        bool
	}{
		{
			name:           "Test parseThresholdLimit with valid limit Success",
			args:           args{limit: "sast-low=1"},
			wantEngineName: "sast-low",
			wantIntLimit:   1,
			wantErr:        false,
		},
		{
			name:           "Test parseThresholdLimit with invalid limit Fail",
			args:           args{limit: "kics-medium=error"},
			wantEngineName: "iac-security-medium",
			wantIntLimit:   0,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotEngineName, gotIntLimit, err := parseThresholdLimit(tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseThresholdLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEngineName != tt.wantEngineName {
				t.Errorf("parseThresholdLimit() gotEngineName = %v, want %v", gotEngineName, tt.wantEngineName)
			}
			if gotIntLimit != tt.wantIntLimit {
				t.Errorf("parseThresholdLimit() gotIntLimit = %v, want %v", gotIntLimit, tt.wantIntLimit)
			}
		})
	}
}

func Test_validateThresholds(t *testing.T) {
	tests := []struct {
		name         string
		thresholdMap map[string]int
		wantErr      bool
	}{
		{
			name: "Valid Thresholds",
			thresholdMap: map[string]int{
				"sast-medium": 5,
				"sast-high":   10,
			},
			wantErr: false,
		},
		{
			name: "Invalid Threshold - Negative Limit",
			thresholdMap: map[string]int{
				"sca-medium": -3,
			},
			wantErr: true,
		},
		{
			name: "Invalid Threshold - Zero Limit",
			thresholdMap: map[string]int{
				"sca-high": 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validateThresholds(tt.thresholdMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateThresholds() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContainerImageFormat(t *testing.T) {
	var errMessage = "Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag> or <image-name>.tar"

	testCases := []struct {
		name           string
		containerImage string
		expectedError  error
	}{
		{
			name:           "Valid container image format",
			containerImage: "nginx:latest",
			expectedError:  nil,
		},
		{
			name:           "Valid compressed container image format",
			containerImage: "nginx.tar",
			expectedError:  nil,
		},
		{
			name:           "Missing image name",
			containerImage: ":latest",
			expectedError:  errors.New(errMessage),
		},
		{
			name:           "Missing image tag",
			containerImage: "nginx:",
			expectedError:  errors.New(errMessage),
		},
		{
			name:           "Empty image name and tag",
			containerImage: ":",
			expectedError:  errors.New(errMessage),
		},
		{
			name:           "Extra colon",
			containerImage: "nginx:latest:extra",
			expectedError:  errors.New(errMessage),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateContainerImageFormat(tc.containerImage)
			if err != nil && tc.expectedError == nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if err != nil && tc.expectedError != nil && err.Error() != tc.expectedError.Error() {
				t.Errorf("Expected error %v, but got %v", tc.expectedError, err)
			}
			if err == nil && tc.expectedError != nil {
				t.Errorf("Expected error %v, but got nil", tc.expectedError)
			}
		})
	}
}

func TestAddContainersScan_WithCustomImages_ShouldSetUserCustomImages(t *testing.T) {
	// Setup
	var resubmitConfig []wrappers.Config

	// Create command with container flag
	cmdCommand := &cobra.Command{}
	cmdCommand.Flags().String(commonParams.ContainerImagesFlag, "", "Container images")

	// Set test values for container images (comma-separated private artifactory images)
	expectedImages := "artifactory.company.com/repo/image1:latest,artifactory.company.com/repo/image2:1.0.3"
	_ = cmdCommand.Flags().Set(commonParams.ContainerImagesFlag, expectedImages)

	// Enable container scan type
	originalScanTypes := actualScanTypes
	actualScanTypes = commonParams.ContainersType // Use string instead of slice
	defer func() {
		actualScanTypes = originalScanTypes // Restore original value instead of nil
	}()

	// Execute
	result, err := addContainersScan(cmdCommand, resubmitConfig)

	// Verify no error occurred
	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected result to not be nil")

	// Verify
	containerMapConfig, ok := result[resultsMapValue].(*wrappers.ContainerConfig)
	assert.Assert(t, ok, "Expected result to contain a ContainerConfig")

	// Check that the UserCustomImages field was correctly set
	assert.Equal(t, containerMapConfig.UserCustomImages, expectedImages,
		"Expected UserCustomImages to be set to '%s', but got '%s'",
		expectedImages, containerMapConfig.UserCustomImages)
}

func TestInitializeContainersConfigWithResubmitValues_UserCustomImages(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name                    string
		resubmitConfig          []wrappers.Config
		containerResolveLocally bool
		expectedCustomImages    string
	}{
		{
			name: "When UserCustomImages is valid string and ContainerResolveLocally is false, it should be set in containerConfig",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: "image1:tag1,image2:tag2",
					},
				},
			},
			containerResolveLocally: false,
			expectedCustomImages:    "image1:tag1,image2:tag2",
		},
		{
			name: "When UserCustomImages is valid string and ContainerResolveLocally is true, it should not be set in containerConfig",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: "image1:tag1,image2:tag2",
					},
				},
			},
			containerResolveLocally: true,
			expectedCustomImages:    "",
		},
		{
			name: "When UserCustomImages is empty string, containerConfig should not be updated",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: "",
					},
				},
			},
			containerResolveLocally: false,
			expectedCustomImages:    "",
		},
		{
			name: "When UserCustomImages is nil, containerConfig should not be updated",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: nil,
					},
				},
			},
			containerResolveLocally: false,
			expectedCustomImages:    "",
		},
		{
			name: "When config.Value doesn't have UserCustomImages key, containerConfig should not be updated",
			resubmitConfig: []wrappers.Config{
				{
					Type:  commonParams.ContainersType,
					Value: map[string]interface{}{},
				},
			},
			containerResolveLocally: false,
			expectedCustomImages:    "",
		},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize containerConfig
			containerConfig := &wrappers.ContainerConfig{}

			// Call the function under test
			initializeContainersConfigWithResubmitValues(tc.resubmitConfig, containerConfig, tc.containerResolveLocally)

			// Assert the result
			assert.Equal(t, tc.expectedCustomImages, containerConfig.UserCustomImages,
				"Expected UserCustomImages to be %q but got %q", tc.expectedCustomImages, containerConfig.UserCustomImages)
		})
	}
}

func Test_WhenScaResolverAndResultsFileExist_ThenAddScaResultsShouldRemoveThemAfterAddingToZip(t *testing.T) {
	// Step 1: Create a temporary file to  simulate the SCA results file and check for errors.
	tempFile, err := os.CreateTemp("", "sca_results_test")
	assert.NilError(t, err)

	// Step 2: Schedule deletion of the temporary file after the test completes.
	defer os.Remove(tempFile.Name())

	// Step 3: Define the path for scaResolverResultsFile, adding ".json" extension.
	scaResolverResultsFile = tempFile.Name() + ".json"

	// Step 4: Create scaResolverResultsFile on disk to simulate its existence before running addScaResults.
	_, err = os.Create(scaResolverResultsFile)
	assert.NilError(t, err, "Expected scaResolverResultsFile to be created")

	// Step 5: Define and create scaResultsFile (without ".json" extension) to simulate another required file.
	scaResultsFile := strings.TrimSuffix(scaResolverResultsFile, ".json")
	_, err = os.Create(scaResultsFile)
	assert.NilError(t, err, "Expected scaResultsFile to be created")

	// Step 6: Set up a buffer to collect the zip file's contents.
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	// Step 7: Redirect log output to logBuffer to capture logs for validation.
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	// Step 8 : Ensure log output is reset to standard error after the test completes.
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	// Step 9: Call addScaResults, which should add results to the zipWriter and delete temporary files.
	err = addScaResults(zipWriter)
	assert.NilError(t, err)

	// Step 10: Close the zip writer to complete the writing process.
	zipWriter.Close()

	// Step 11: Check if scaResolverResultsFile was successfully deleted after addScaResults ran.
	_, err = os.Stat(scaResolverResultsFile)
	assert.Assert(t, os.IsNotExist(err), "Expected scaResolverResultsFile to be deleted")

	// Step 12: Check if scaResultsFile was successfully deleted as well.
	_, err = os.Stat(scaResultsFile)
	assert.Assert(t, os.IsNotExist(err), "Expected scaResultsFile to be deleted")

	// Step 13: Validate log output to confirm the success message for file removal is present.
	logOutput := logBuffer.String()
	t.Logf("Log output:\n%s", logOutput)
	assert.Assert(t, strings.Contains(logOutput, "Successfully removed file"), "Expected success log for file removal")
}

func TestFilterMatched(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		fileName string
		expected bool
	}{
		{
			name:     "whenFileMatchesInclusionFilter_shouldReturnTrue",
			filters:  []string{"*.go"},
			fileName: "main.go",
			expected: true,
		},
		{
			name:     "whenFileNoMatchesInclusionFilter_shouldReturnFalse",
			filters:  []string{"*.go"},
			fileName: "main.py",
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			result := filterMatched(tt.filters, tt.fileName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createOutputFile(t *testing.T, fileName string) *os.File {
	file, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	logger.SetOutput(file)
	return file
}

func deleteOutputFile(file *os.File) {
	file.Close()
	err := os.Remove(file.Name())
	if err != nil {
		logger.Printf("Failed to remove log file: %v", err)
	}
}

func TestResubmitConfig_ProjectDoesNotExist_ReturnedEmptyConfig(t *testing.T) {
	scanWrapper := mock.ScansMockWrapper{}
	projectID := "non-existent-project"
	userScanTypes := ""
	cmd := createASTTestCommand()
	cmd.PersistentFlags().String("project-name", "non-existent-project", "project name")
	config, err := getResubmitConfiguration(&scanWrapper, projectID, userScanTypes)
	assert.NilError(t, err)
	assert.Equal(t, len(config), 0)
}

func TestUploadZip_whenUserProvideZip_shouldReturnEmptyZipFilePathInSuccessCase(t *testing.T) {
	uploadWrapper := mock.UploadsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	_, zipPath, err := uploadZip(&uploadWrapper, "test.zip", false, true, featureFlagsWrapper)
	assert.NilError(t, err)
	assert.Equal(t, zipPath, "")
}

func TestUploadZip_whenUserProvideZip_shouldReturnEmptyZipFilePathInFailureCase(t *testing.T) {
	uploadWrapper := mock.UploadsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	_, zipPath, err := uploadZip(&uploadWrapper, "failureCase.zip", false, true, featureFlagsWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "error from UploadFile"), err.Error())
	assert.Equal(t, zipPath, "")
}

func TestUploadZip_whenUserNotProvideZip_shouldReturnZipFilePathInSuccessCase(t *testing.T) {
	uploadWrapper := mock.UploadsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	_, zipPath, err := uploadZip(&uploadWrapper, "test.zip", false, false, featureFlagsWrapper)
	assert.NilError(t, err)
	assert.Equal(t, zipPath, "test.zip")
}

func TestUploadZip_whenUserNotProvideZip_shouldReturnZipFilePathInFailureCase(t *testing.T) {
	uploadWrapper := mock.UploadsMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
	_, zipPath, err := uploadZip(&uploadWrapper, "failureCase.zip", false, false, featureFlagsWrapper)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "error from UploadFile"), err.Error())
	assert.Equal(t, zipPath, "failureCase.zip")
}

func TestAddSastScan_ScanFlags(t *testing.T) {
	var resubmitConfig []wrappers.Config

	tests := []struct {
		name                             string
		requiredIncrementalSet           bool
		requiredFastScanSet              bool
		requiredLightQueriesSet          bool
		requiredRecommendedExclusionsSet bool
		fastScanFlag                     string
		incrementalFlag                  string
		lightQueriesFlag                 string
		recommendedExclusionsFlag        string
		expectedConfig                   wrappers.SastConfig
	}{
		{
			name:                             "Fast scan, Incremental scan, Light Queries and Recommended Exclusion are false",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "false",
			incrementalFlag:                  "false",
			lightQueriesFlag:                 "false",
			recommendedExclusionsFlag:        "false",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "false",
				Incremental:           "false",
				LightQueries:          "false",
				RecommendedExclusions: "false",
			},
		},
		{
			name:                             "Fast scan, Incremental scan, Light Queries and Recommended Exclusion are true",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "true",
			incrementalFlag:                  "true",
			lightQueriesFlag:                 "true",
			recommendedExclusionsFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "true",
				Incremental:           "true",
				LightQueries:          "true",
				RecommendedExclusions: "true",
			},
		},
		{
			name:                             "Fast scan, Incremental scan, Light Queries and Recommended Exclusion not set",
			requiredIncrementalSet:           false,
			requiredFastScanSet:              false,
			requiredLightQueriesSet:          false,
			requiredRecommendedExclusionsSet: false,
			expectedConfig:                   wrappers.SastConfig{},
		},
		{
			name:                             "Fast scan is true, Incremental is false, Light Queries is false and Recommended Exclusions is false",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "true",
			incrementalFlag:                  "false",
			lightQueriesFlag:                 "false",
			recommendedExclusionsFlag:        "false",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "true",
				Incremental:           "false",
				LightQueries:          "false",
				RecommendedExclusions: "false",
			},
		},
		{
			name:                             "Fast scan is false, Incremental is true, Light Queries is false and Recommended Exclusions is false",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "false",
			incrementalFlag:                  "true",
			lightQueriesFlag:                 "false",
			recommendedExclusionsFlag:        "false",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "false",
				Incremental:           "true",
				LightQueries:          "false",
				RecommendedExclusions: "false",
			},
		},
		{
			name:                             "Fast scan is false, Incremental is false, Light Queries is true and Recommended Exclusions is false",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "false",
			incrementalFlag:                  "false",
			lightQueriesFlag:                 "true",
			recommendedExclusionsFlag:        "false",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "false",
				Incremental:           "false",
				LightQueries:          "true",
				RecommendedExclusions: "false",
			},
		},
		{
			name:                             "Fast scan is false, Incremental is false, Light Queries is false and Recommended Exclusions is true",
			requiredIncrementalSet:           true,
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "false",
			incrementalFlag:                  "false",
			lightQueriesFlag:                 "false",
			recommendedExclusionsFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "false",
				Incremental:           "false",
				LightQueries:          "false",
				RecommendedExclusions: "true",
			},
		},
		{
			name:                             "Fast scan is not set , Incremental is true , Light Queries is true and Recommended Exclusion is true",
			requiredIncrementalSet:           true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			incrementalFlag:                  "true",
			lightQueriesFlag:                 "true",
			recommendedExclusionsFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				Incremental:           "true",
				LightQueries:          "true",
				RecommendedExclusions: "true",
			},
		},
		{
			name:                             "Fast scan is true , Incremental is not set , Light Queries is true and Recommended Exclusion is true",
			requiredFastScanSet:              true,
			requiredLightQueriesSet:          true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "true",
			lightQueriesFlag:                 "true",
			recommendedExclusionsFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "true",
				LightQueries:          "true",
				RecommendedExclusions: "true",
			},
		},
		{
			name:                             "Fast scan is true , Incremental is true , Light Queries is not set and Recommended Exclusion is true",
			requiredFastScanSet:              true,
			requiredIncrementalSet:           true,
			requiredRecommendedExclusionsSet: true,
			fastScanFlag:                     "true",
			incrementalFlag:                  "true",
			recommendedExclusionsFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				FastScanMode:          "true",
				Incremental:           "true",
				RecommendedExclusions: "true",
			},
		},
		{
			name:                    "Fast scan is true , Incremental is true , Light Queries is true and Recommended Exclusion is not set",
			requiredFastScanSet:     true,
			requiredIncrementalSet:  true,
			requiredLightQueriesSet: true,
			fastScanFlag:            "true",
			incrementalFlag:         "true",
			lightQueriesFlag:        "true",
			expectedConfig: wrappers.SastConfig{
				FastScanMode: "true",
				Incremental:  "true",
				LightQueries: "true",
			},
		},
	}

	oldActualScanTypes := actualScanTypes

	defer func() {
		actualScanTypes = oldActualScanTypes
	}()

	for _, tt := range tests {
		actualScanTypes = "sast,sca,kics,scs"
		t.Run(tt.name, func(t *testing.T) {
			cmdCommand := &cobra.Command{
				Use:   "scan",
				Short: "Scan a project",
				Long:  `Scan a project`,
			}
			cmdCommand.PersistentFlags().Bool(commonParams.SastFastScanFlag, false, "Fast scan flag")
			cmdCommand.PersistentFlags().Bool(commonParams.IncrementalSast, false, "Incremental scan flag")
			cmdCommand.PersistentFlags().Bool(commonParams.SastLightQueriesFlag, false, "Enable SAST Light Queries")
			cmdCommand.PersistentFlags().Bool(commonParams.SastRecommendedExclusionsFlags, false, "Enable SAST Recommended Exclusions")

			_ = cmdCommand.Execute()

			if tt.requiredFastScanSet {
				_ = cmdCommand.PersistentFlags().Set(commonParams.SastFastScanFlag, tt.fastScanFlag)
			}
			if tt.requiredIncrementalSet {
				_ = cmdCommand.PersistentFlags().Set(commonParams.IncrementalSast, tt.incrementalFlag)
			}

			if tt.requiredLightQueriesSet {
				_ = cmdCommand.PersistentFlags().Set(commonParams.SastLightQueriesFlag, tt.lightQueriesFlag)
			}

			if tt.requiredRecommendedExclusionsSet {
				_ = cmdCommand.PersistentFlags().Set(commonParams.SastRecommendedExclusionsFlags, tt.recommendedExclusionsFlag)
			}

			result := addSastScan(cmdCommand, resubmitConfig)

			actualSastConfig := wrappers.SastConfig{}
			for key, value := range result {
				if key == resultsMapType {
					assert.Equal(t, commonParams.SastType, value)
				} else if key == resultsMapValue {
					actualSastConfig = *value.(*wrappers.SastConfig)
				}
			}

			if !reflect.DeepEqual(actualSastConfig, tt.expectedConfig) {
				t.Errorf("Expected %+v, but got %+v", tt.expectedConfig, actualSastConfig)
			}
		})
	}
}

func TestValidateScanTypes(t *testing.T) {
	tests := []struct {
		name             string
		userScanTypes    string
		userSCSScanTypes string
		allowedEngines   map[string]bool
		expectedError    string
	}{
		{
			name:             "No licenses available",
			userScanTypes:    "scs",
			userSCSScanTypes: "sast,secret-detection",
			allowedEngines:   map[string]bool{"scs": false, "enterprise-secrets": false},
			expectedError:    "It looks like the \"scs\" scan type does",
		},
		{
			name:             "SCS license available, secret-detection not available",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": false},
			expectedError:    "It looks like the \"secret-detection\" scan type does not exist",
		},
		{
			name:             "All licenses available",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": true},
			expectedError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String(commonParams.ScanTypes, tt.userScanTypes, "")
			cmd.Flags().String(commonParams.SCSEnginesFlag, tt.userSCSScanTypes, "")

			jwtWrapper := &mock.JWTMockWrapper{
				CustomGetAllowedEngines: func(featureFlagsWrapper wrappers.FeatureFlagsWrapper) (map[string]bool, error) {
					return tt.allowedEngines, nil
				},
			}
			featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}
			err := validateScanTypes(cmd, jwtWrapper, featureFlagsWrapper)
			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}

func TestCreateScanWith_ScaResolver_Source_as_Zip(t *testing.T) {
	clearFlags()
	baseArgs := []string{
		"scan",
		"create",
		"--project-name",
		"MOCK",
		"-s",
		"data/sources.zip",
		"-b",
		"dummy_branch",
		"--sca-resolver",
		"ScaResolver.exe",
	}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Assert(t, strings.Contains(err.Error(), ScaResolverZipNotSupportedErr), err.Error())
}

func Test_parseArgs(t *testing.T) {
	tests := []struct {
		inputString string
		lenOfArgs   int
	}{
		{"--log-level Debug --break-on-manifest-failure", 3},
		{`test test1`, 2},
		{"--gradle-parameters='-Prepository.proxy.url=123 -Prepository.proxy.username=123 -Prepository.proxy.password=123' --log-level Debug", 3},
	}

	for _, test := range tests {
		fmt.Println("test ::", test)
		result := parseArgs(test.inputString)
		if len(result) != test.lenOfArgs {
			t.Errorf(" test case failed for params %v", test)
		}
	}
}

func Test_isValidJSONOrXML(t *testing.T) {
	tests := []struct {
		description string
		inputPath   string
		output      bool
	}{
		{"wrong extension", "somefile.txt", false},
		{"wrong json file", "wrongfilepath.json", false},
		{"wrong xml file", "wrongfilepath.xml", false},
		{"correct file", "data/package.json", true},
	}

	for _, test := range tests {
		isValid, _ := isValidJSONOrXML(test.inputPath)
		if isValid != test.output {
			t.Errorf(" test case failed for params %v", test)
		}
	}
}

func Test_CreateScanWithSbomFlag(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan", "create", "--project-name", "newProject", "-s", "data/sbom.json", "--branch", "dummy_branch", "--sbom-only",
	)

	assert.ErrorContains(t, err, "Failed creating a scan: Input in bad format: failed to read file:")
}
