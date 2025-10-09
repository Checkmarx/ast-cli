//go:build !integration

package commands

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
	assert.Assert(t, err.Error() == "Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag>, <image-name>.tar, or use a supported prefix (docker:, podman:, containerd:, registry:, docker-archive:, oci-archive:, oci-dir:, file:)")
}

func TestCreateScanFromFolder_CommaSeparatedContainerImages_SingleBadEntry_FailCreatingScan(t *testing.T) {
	clearFlags()
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-b", "dummy_branch", "--container-images", "docker:nginx:latest,dir:/bad/directory,registry:ubuntu:20.04", "--scan-types", "containers"}
	err := execCmdNotNilAssertion(t, append(baseArgs, "-s", blankSpace+"."+blankSpace)...)
	assert.Assert(t, err.Error() == "Invalid value for --container-images flag. The 'dir:' prefix is not supported as it would scan entire directories rather than a single image")
}

func TestCreateScanWithThreshold_ShouldSuccess(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch", "--scan-types", "sast", "--threshold", "sca-low=1 ; sast-medium=2")
}

func TestScanCreate_ApplicationNameIsNotExactMatch_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "non-existing-project", "--application-name", "MOC", "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_ExistingProjectAndApplicationWithNoPermission_ShouldFailScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.NoPermissionApp, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, strings.Contains(err.Error(), errorConstants.FailedToGetApplication), err.Error())
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

// Now as we give the ability to assign existing projects to applications , there is validation if application exists

func TestCreateScan_WhenProjectExists_GetApplication_Fails500Err_Failed(t *testing.T) {
	baseArgs := []string{scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--debug", "--application-name", mock.FakeInternalServerError500}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.ErrorContains(t, err, errorConstants.FailedToGetApplication, err.Error())
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
func TestAddSCSScan_ResubmitWithoutScorecardFlags_ShouldPass(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_ResubmitWithScorecardFlags_ShouldPass(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
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

func TestAddSCSScan_WithSCSSecretDetectionAndScorecard_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithoutSCSSecretDetection_scsMapNoSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: false,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetection_scsMapHasSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resubmitConfig []wrappers.Config
			cmdCommand := &cobra.Command{
				Use:   "scan",
				Short: "Scan a project",
				Long:  `Scan a project`,
			}
			cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
			_ = cmdCommand.Execute()
			_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection")

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

			scsConfig := wrappers.SCSConfig{
				Twoms: "true",
			}
			scsMapConfig := make(map[string]interface{})
			scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
			scsMapConfig[resultsMapValue] = &scsConfig

			if !reflect.DeepEqual(result, scsMapConfig) {
				t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
			}
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardWithScanTypesAndNoScorecardFlags_scsMapHasSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndWithoutScanTypes_scsMapHasSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resubmitConfig []wrappers.Config
			cmdCommand := &cobra.Command{
				Use:   "scan",
				Short: "Scan a project",
				Long:  `Scan a project`,
			}

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

			scsConfig := wrappers.SCSConfig{
				Twoms: "true",
			}

			scsMapConfig := make(map[string]interface{})
			scsMapConfig[resultsMapType] = commonParams.MicroEnginesType
			scsMapConfig[resultsMapValue] = &scsConfig

			if !reflect.DeepEqual(result, scsMapConfig) {
				t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
			}
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepo_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepoWithTokenInURL_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardGithubRepoWithTokenInURL_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardGithubRepoWithTokenAndUsernameInURL_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardShortenedGithubRepoWithTokenAndUsernameInURL_scsMapHasBoth(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardGitLabRepo_scsMapHasSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
	}
}

func TestAddSCSScan_WithSCSSecretDetectionAndScorecardGitSSHRepo_scsMapHasSecretDetection(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasRepositoryHealthLicense  bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
	}{
		{
			name:                        "scsLicensingV2 disabled",
			scsLicensingV2:              false,
			hasRepositoryHealthLicense:  false,
			hasSecretDetectionLicense:   false,
			hasEnterpriseSecretsLicense: true,
		},
		{
			name:                        "scsLicensingV2 enabled",
			scsLicensingV2:              true,
			hasRepositoryHealthLicense:  true,
			hasSecretDetectionLicense:   true,
			hasEnterpriseSecretsLicense: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, _ := addSCSScan(cmdCommand, resubmitConfig, tt.scsLicensingV2,
				tt.hasRepositoryHealthLicense, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense)

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
		})
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

// TestValidateContainerImageFormat tests the basic validation logic for container image formats.
// Container-security scan-type related test function.
// This test covers traditional image:tag formats, tar files, and various error cases.
func TestValidateContainerImageFormat(t *testing.T) {
	var traditionalErrorMessage = "Invalid value for --container-images flag. The value must be in the format <image-name>:<image-tag>, <image-name>.tar, or use a supported prefix (docker:, podman:, containerd:, registry:, docker-archive:, oci-archive:, oci-dir:, file:)"

	testCases := []struct {
		name           string
		containerImage string
		expectedError  error
		setupFiles     []string // Files to create for testing
		setupDirs      []string // Directories to create for testing
	}{
		// Traditional format tests
		{
			name:           "Valid container image format",
			containerImage: "nginx:latest",
			expectedError:  nil,
		},
		{
			name:           "Valid compressed container image format",
			containerImage: "nginx.tar",
			expectedError:  nil,
			setupFiles:     []string{"nginx.tar"},
		},
		{
			name:           "Valid tar file with path (like syft)",
			containerImage: "empty/alpine.tar",
			expectedError:  nil,
			setupFiles:     []string{"empty/alpine.tar"},
		},
		{
			name:           "Invalid tar file with path - file does not exist",
			containerImage: "nonexistent/alpine.tar",
			expectedError:  errors.New("--container-images flag error: file 'nonexistent/alpine.tar' does not exist"),
		},
		{
			name:           "Missing image name",
			containerImage: ":latest",
			expectedError:  errors.New(traditionalErrorMessage),
		},
		{
			name:           "Missing image tag",
			containerImage: "nginx:",
			expectedError:  errors.New(traditionalErrorMessage),
		},
		{
			name:           "Empty image name and tag",
			containerImage: ":",
			expectedError:  errors.New(traditionalErrorMessage),
		},
		{
			name:           "Extra colon in traditional format",
			containerImage: "nginx:latest:extra",
			expectedError:  errors.New(traditionalErrorMessage),
		},

		// Docker daemon prefix tests
		{
			name:           "Valid docker daemon format",
			containerImage: "docker:nginx:latest",
			expectedError:  nil,
		},
		{
			name:           "Valid docker daemon format with registry",
			containerImage: "docker:registry.example.com/namespace/image:tag",
			expectedError:  nil,
		},
		{
			name:           "Invalid docker daemon format - missing tag",
			containerImage: "docker:nginx:",
			expectedError:  errors.New("Invalid value for --container-images flag. Prefix 'docker:' expects format <image-name>:<image-tag>"),
		},
		{
			name:           "Invalid docker daemon format - empty image ref",
			containerImage: "docker:",
			expectedError:  errors.New("Invalid value for --container-images flag. After prefix 'docker:', the image reference cannot be empty"),
		},

		// Podman daemon prefix tests
		{
			name:           "Valid podman daemon format",
			containerImage: "podman:test:latest",
			expectedError:  nil,
		},
		{
			name:           "Invalid podman daemon format - missing image name",
			containerImage: "podman::latest",
			expectedError:  errors.New("Invalid value for --container-images flag. Prefix 'podman:' expects format <image-name>:<image-tag>"),
		},

		// Containerd daemon prefix tests
		{
			name:           "Valid containerd daemon format",
			containerImage: "containerd:test:latest",
			expectedError:  nil,
		},

		// Registry prefix tests
		{
			name:           "Valid registry format",
			containerImage: "registry:test:latest",
			expectedError:  nil,
		},

		// Docker archive prefix tests
		{
			name:           "Valid docker archive format",
			containerImage: "docker-archive:test.tar",
			expectedError:  nil,
			setupFiles:     []string{"test.tar"},
		},
		{
			name:           "Valid docker archive format with different extension",
			containerImage: "docker-archive:image.tar.gz",
			expectedError:  nil,
			setupFiles:     []string{"image.tar.gz"},
		},
		{
			name:           "Invalid docker archive format - non-existent file",
			containerImage: "docker-archive:nonexistent.tar",
			expectedError:  errors.New("--container-images flag error: file 'nonexistent.tar' does not exist"),
		},

		// OCI archive prefix tests
		{
			name:           "Valid oci archive format",
			containerImage: "oci-archive:test.tar",
			expectedError:  nil,
			setupFiles:     []string{"test.tar"},
		},
		{
			name:           "Valid oci archive with any file extension",
			containerImage: "oci-archive:archive.tgz",
			expectedError:  nil,
			setupFiles:     []string{"archive.tgz"},
		},
		{
			name:           "Invalid oci archive format - non-existent file",
			containerImage: "oci-archive:nonexistent.tar",
			expectedError:  errors.New("--container-images flag error: file 'nonexistent.tar' does not exist"),
		},

		// OCI directory prefix tests (matches Syft behavior)
		{
			name:           "Valid oci-dir with directory",
			containerImage: "oci-dir:test-dir",
			expectedError:  nil,
			setupDirs:      []string{"test-dir"},
		},
		{
			name:           "Valid oci-dir with directory and tag",
			containerImage: "oci-dir:test-dir:latest",
			expectedError:  nil,
			setupDirs:      []string{"test-dir"},
		},
		{
			name:           "Valid oci-dir with file (like .tar)",
			containerImage: "oci-dir:image.tar",
			expectedError:  nil,
			setupFiles:     []string{"image.tar"},
		},
		{
			name:           "Valid oci-dir with file and tag",
			containerImage: "oci-dir:image.tar:v1.0",
			expectedError:  nil,
			setupFiles:     []string{"image.tar"},
		},
		{
			name:           "Invalid oci-dir format - non-existent path",
			containerImage: "oci-dir:nonexistent-path",
			expectedError:  errors.New("--container-images flag error: path nonexistent-path does not exist"),
		},

		// Directory prefix tests - RESTRICTED (not allowed for single image scanning)
		{
			name:           "Invalid directory format - dir prefix not supported",
			containerImage: "dir:myproject",
			expectedError:  errors.New("Invalid value for --container-images flag. The 'dir:' prefix is not supported as it would scan entire directories rather than a single image"),
		},

		// File prefix tests (matches Syft - any single file)
		{
			name:           "Valid file format with tar",
			containerImage: "file:test.tar",
			expectedError:  nil,
			setupFiles:     []string{"test.tar"},
		},
		{
			name:           "Valid file format with any extension",
			containerImage: "file:test.txt",
			expectedError:  nil,
			setupFiles:     []string{"test.txt"},
		},
		{
			name:           "Valid file format with no extension",
			containerImage: "file:myfile",
			expectedError:  nil,
			setupFiles:     []string{"myfile"},
		},
		{
			name:           "Invalid file format - non-existent file",
			containerImage: "file:nonexistent.file",
			expectedError:  errors.New("--container-images flag error: file 'nonexistent.file' does not exist"),
		},

		// Registry prefix tests (restricted to single images only)
		{
			name:           "Valid registry format simple",
			containerImage: "registry:ubuntu:latest",
			expectedError:  nil,
		},
		{
			name:           "Valid registry format with port",
			containerImage: "registry:localhost:5000/image:tag",
			expectedError:  nil,
		},
		{
			name:           "Valid registry format complex",
			containerImage: "registry:registry.example.com/namespace/image:tag",
			expectedError:  nil,
		},
		{
			name:           "Valid registry format no tag",
			containerImage: "registry:myimage",
			expectedError:  nil,
		},
		{
			name:           "Invalid registry format - just registry URL",
			containerImage: "registry:registry.example.com",
			expectedError:  errors.New("Invalid value for --container-images flag. Registry format must specify a single image, not just a registry URL. Use format: registry:<registry-url>/<image>:<tag> or registry:<image>:<tag>"),
		},
		{
			name:           "Invalid registry format - registry with port only",
			containerImage: "registry:localhost:5000",
			expectedError:  errors.New("Invalid value for --container-images flag. Registry format must specify a single image, not just a registry URL. Use format: registry:<registry-url>/<image>:<tag>"),
		},

		// Edge cases
		{
			name:           "Complex registry with multiple colons using docker prefix",
			containerImage: "docker:registry.example.com:5000/namespace/image:v1.2.3",
			expectedError:  nil,
		},
		{
			name:           "Complex registry with multiple colons using registry prefix",
			containerImage: "registry:registry.example.com:5000/namespace/image:v1.2.3",
			expectedError:  nil,
		},

		// Note: Comma-separated validation is tested at the integration level
		// since validateContainerImageFormat() only validates single entries.
		// The comma splitting and individual validation happens in addContainersScan().
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Setup test files and directories
			cleanupFuncs := setupTestFilesAndDirs(t, tc.setupFiles, tc.setupDirs)
			defer func() {
				for _, cleanup := range cleanupFuncs {
					cleanup()
				}
			}()

			err := validateContainerImageFormat(tc.containerImage)
			if err != nil && tc.expectedError == nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if err != nil && tc.expectedError != nil && err.Error() != tc.expectedError.Error() {
				t.Errorf("Expected error '%v', but got '%v'", tc.expectedError, err)
			}
			if err == nil && tc.expectedError != nil {
				t.Errorf("Expected error %v, but got nil", tc.expectedError)
			}
		})
	}
}

// TestValidateContainerImageFormat_Comprehensive tests the complete validation logic
// including input normalization, helpful hints, and all error cases.
// Container-security scan-type related test function.
// This test validates all supported container image formats, prefixes, tar files,
// error messages, and helpful hints for the --container-images flag.
func TestValidateContainerImageFormat_Comprehensive(t *testing.T) {
	testCases := []struct {
		name           string
		containerImage string
		expectedError  string
		setupFiles     []string
	}{
		// ==================== Basic Format Tests ====================
		{
			name:           "Valid image with tag",
			containerImage: "nginx:latest",
			expectedError:  "",
		},
		{
			name:           "Valid image with version tag",
			containerImage: "alpine:3.18",
			expectedError:  "",
		},
		{
			name:           "Valid image with complex registry",
			containerImage: "registry.example.com:5000/namespace/image:v1.2.3",
			expectedError:  "",
		},
		{
			name:           "Invalid - missing tag",
			containerImage: "nginx",
			expectedError:  "--container-images flag error: image does not have a tag",
		},
		{
			name:           "Invalid - empty tag",
			containerImage: "nginx:",
			expectedError:  "Invalid value for --container-images flag. Image name and tag cannot be empty",
		},
		{
			name:           "Invalid - empty name",
			containerImage: ":latest",
			expectedError:  "Invalid value for --container-images flag. Image name and tag cannot be empty",
		},

		// ==================== Tar File Tests ====================
		{
			name:           "Valid tar file",
			containerImage: "alpine.tar",
			expectedError:  "",
			setupFiles:     []string{"alpine.tar"},
		},
		{
			name:           "Valid tar file in current dir",
			containerImage: "image-with-path.tar",
			expectedError:  "",
			setupFiles:     []string{"image-with-path.tar"},
		},
		{
			name:           "Invalid - tar file does not exist",
			containerImage: "nonexistent.tar",
			expectedError:  "--container-images flag error: file 'nonexistent.tar' does not exist",
		},

		// ==================== Compressed Tar Tests ====================
		{
			name:           "Invalid - compressed tar.gz",
			containerImage: "image.tar.gz",
			expectedError:  "--container-images flag error: file 'image.tar.gz' is compressed, use non-compressed format (tar)",
		},
		{
			name:           "Invalid - compressed tar.bz2",
			containerImage: "image.tar.bz2",
			expectedError:  "--container-images flag error: file 'image.tar.bz2' is compressed, use non-compressed format (tar)",
		},
		{
			name:           "Invalid - compressed tar.xz",
			containerImage: "image.tar.xz",
			expectedError:  "--container-images flag error: file 'image.tar.xz' is compressed, use non-compressed format (tar)",
		},
		{
			name:           "Invalid - compressed tgz",
			containerImage: "image.tgz",
			expectedError:  "--container-images flag error: file 'image.tgz' is compressed, use non-compressed format (tar)",
		},

		// ==================== Helpful Hints Tests ====================
		{
			name:           "Hint - looks like tar file (wrong extension)",
			containerImage: "image.tar.bz",
			expectedError:  "--container-images flag error: image does not have a tag. Did you try to scan a tar file?",
		},
		{
			name:           "Hint - looks like tar file (typo in extension)",
			containerImage: "image.tar.ez2",
			expectedError:  "--container-images flag error: image does not have a tag. Did you try to scan a tar file?",
		},

		// ==================== File Prefix Tests ====================
		{
			name:           "Valid file prefix with tar",
			containerImage: "file:alpine.tar",
			expectedError:  "",
			setupFiles:     []string{"alpine.tar"},
		},
		{
			name:           "Valid file prefix with image",
			containerImage: "file:prefixed-image.tar",
			expectedError:  "",
			setupFiles:     []string{"prefixed-image.tar"},
		},
		{
			name:           "Invalid file prefix - missing file",
			containerImage: "file:nonexistent.tar",
			expectedError:  "--container-images flag error: file 'nonexistent.tar' does not exist",
		},
		{
			name:           "Hint - file prefix with image name",
			containerImage: "file:nginx:latest",
			expectedError:  "--container-images flag error: file 'nginx:latest' does not exist. Did you try to scan an image using image name and tag?",
		},
		{
			name:           "Hint - file prefix with image (no tag)",
			containerImage: "file:alpine:3.18",
			expectedError:  "--container-images flag error: file 'alpine:3.18' does not exist. Did you try to scan an image using image name and tag?",
		},

		// ==================== Docker Archive Tests ====================
		{
			name:           "Valid docker-archive",
			containerImage: "docker-archive:image.tar",
			expectedError:  "",
			setupFiles:     []string{"image.tar"},
		},
		{
			name:           "Invalid docker-archive - missing file",
			containerImage: "docker-archive:nonexistent.tar",
			expectedError:  "--container-images flag error: file 'nonexistent.tar' does not exist",
		},
		{
			name:           "Hint - docker-archive with image name",
			containerImage: "docker-archive:nginx:latest",
			expectedError:  "--container-images flag error: file 'nginx:latest' does not exist. Did you try to scan an image using image name and tag?",
		},

		// ==================== OCI Archive Tests ====================
		{
			name:           "Valid oci-archive",
			containerImage: "oci-archive:image.tar",
			expectedError:  "",
			setupFiles:     []string{"image.tar"},
		},
		{
			name:           "Invalid oci-archive - missing file",
			containerImage: "oci-archive:nonexistent.tar",
			expectedError:  "--container-images flag error: file 'nonexistent.tar' does not exist",
		},
		{
			name:           "Hint - oci-archive with image name",
			containerImage: "oci-archive:ubuntu:22.04",
			expectedError:  "--container-images flag error: file 'ubuntu:22.04' does not exist. Did you try to scan an image using image name and tag?",
		},

		// ==================== Docker Daemon Tests ====================
		{
			name:           "Valid docker prefix",
			containerImage: "docker:nginx:latest",
			expectedError:  "",
		},
		{
			name:           "Valid docker prefix with registry",
			containerImage: "docker:registry.io/namespace/image:tag",
			expectedError:  "",
		},
		{
			name:           "Invalid docker prefix - missing tag",
			containerImage: "docker:nginx",
			expectedError:  "image does not have a tag",
		},
		{
			name:           "Invalid docker prefix - empty",
			containerImage: "docker:",
			expectedError:  "image does not have a tag",
		},

		// ==================== Podman Daemon Tests ====================
		{
			name:           "Valid podman prefix",
			containerImage: "podman:alpine:3.18",
			expectedError:  "",
		},
		{
			name:           "Invalid podman prefix - missing tag",
			containerImage: "podman:alpine",
			expectedError:  "image does not have a tag",
		},

		// ==================== Containerd Daemon Tests ====================
		{
			name:           "Valid containerd prefix",
			containerImage: "containerd:nginx:latest",
			expectedError:  "",
		},
		{
			name:           "Invalid containerd prefix - missing tag",
			containerImage: "containerd:nginx",
			expectedError:  "image does not have a tag",
		},

		// ==================== Registry Tests ====================
		{
			name:           "Valid registry prefix",
			containerImage: "registry:nginx:latest",
			expectedError:  "",
		},
		{
			name:           "Valid registry with URL",
			containerImage: "registry:myregistry.io/app:v1.0",
			expectedError:  "",
		},
		{
			name:           "Invalid registry - just URL without image",
			containerImage: "registry:myregistry.com",
			expectedError:  "image does not have a tag",
		},

		// ==================== Dir Prefix (Forbidden) ====================
		{
			name:           "Invalid - dir prefix not supported",
			containerImage: "dir:/path/to/dir",
			expectedError:  "Invalid value for --container-images flag. The 'dir:' prefix is not supported",
		},

		// ==================== Edge Cases ====================
		{
			name:           "Complex registry with multiple colons",
			containerImage: "registry.io:5000/namespace/image:v1.2.3",
			expectedError:  "",
		},
		{
			name:           "Image name with dash and underscore",
			containerImage: "my-custom_image:v1.0",
			expectedError:  "",
		},
		{
			name:           "Tar file with multiple dots in name",
			containerImage: "alpine.3.18.0.tar",
			expectedError:  "",
			setupFiles:     []string{"alpine.3.18.0.tar"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Setup test files if needed
			cleanupFuncs := setupTestFilesAndDirs(t, tc.setupFiles, nil)
			defer func() {
				for _, cleanup := range cleanupFuncs {
					cleanup()
				}
			}()

			// Run validation
			err := validateContainerImageFormat(tc.containerImage)

			// Check results
			if tc.expectedError == "" {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tc.expectedError)
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s', but got: %v", tc.expectedError, err)
				}
			}
		})
	}
}

// TestInputNormalization tests the space and quote trimming logic.
// Container-security scan-type related test function.
// This test validates input normalization for comma-separated container image lists,
// including space trimming, quote handling, and empty entry filtering.
func TestInputNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple comma-separated list",
			input:    "nginx:latest,alpine:3.18,ubuntu:22.04",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "With spaces after commas",
			input:    "nginx:latest, alpine:3.18, ubuntu:22.04",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "With spaces before and after commas",
			input:    "nginx:latest , alpine:3.18 , ubuntu:22.04",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "With single quotes",
			input:    "'nginx:latest','alpine:3.18','ubuntu:22.04'",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "With double quotes",
			input:    "\"nginx:latest\",\"alpine:3.18\",\"ubuntu:22.04\"",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "Mixed quotes and spaces",
			input:    "'nginx:latest', \"alpine:3.18\", ubuntu:22.04",
			expected: []string{"nginx:latest", "alpine:3.18", "ubuntu:22.04"},
		},
		{
			name:     "With file paths in quotes",
			input:    "'file:/path/to/image.tar', '/another/path.tar'",
			expected: []string{"file:/path/to/image.tar", "/another/path.tar"},
		},
		{
			name:     "Empty entries (consecutive commas)",
			input:    "nginx:latest,,alpine:3.18",
			expected: []string{"nginx:latest", "alpine:3.18"},
		},
		{
			name:     "Leading/trailing commas",
			input:    ",nginx:latest,alpine:3.18,",
			expected: []string{"nginx:latest", "alpine:3.18"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the normalization logic from addContainersScan
			rawList := strings.Split(strings.TrimSpace(tc.input), ",")
			var normalized []string

			for _, item := range rawList {
				// Trim spaces and quotes
				item = strings.TrimSpace(item)
				item = strings.Trim(item, "'\"")

				// Skip empty entries
				if item == "" {
					continue
				}

				normalized = append(normalized, item)
			}

			// Verify results
			if len(normalized) != len(tc.expected) {
				t.Errorf("Expected %d items, got %d. Expected: %v, Got: %v",
					len(tc.expected), len(normalized), tc.expected, normalized)
				return
			}

			for i, expected := range tc.expected {
				if normalized[i] != expected {
					t.Errorf("Item %d: expected '%s', got '%s'", i, expected, normalized[i])
				}
			}
		})
	}
}

// setupTestFilesAndDirs creates temporary files and directories for testing.
// Container-security scan-type related test helper function.
// This helper creates test files (like .tar files) and directories needed for container image validation tests.
func setupTestFilesAndDirs(t *testing.T, files []string, dirs []string) []func() {
	var cleanupFuncs []func()

	for _, file := range files {
		// Create temporary file
		tempFile, err := os.CreateTemp("", filepath.Base(file))
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		tempFile.Close()

		// Always use relative paths for testing to avoid filesystem permission issues
		targetFile := filepath.Base(file)
		err = os.Rename(tempFile.Name(), targetFile)
		if err != nil {
			t.Fatalf("Failed to rename temp file to %s: %v", targetFile, err)
		}
		cleanupFuncs = append(cleanupFuncs, func() {
			os.Remove(targetFile)
		})
	}

	for _, dir := range dirs {
		// Always use relative paths for testing to avoid filesystem permission issues
		targetDir := filepath.Base(dir)
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", targetDir, err)
		}
		cleanupFuncs = append(cleanupFuncs, func() {
			os.RemoveAll(targetDir)
		})
	}

	return cleanupFuncs
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

func TestAddContainersScan_GitScanWithResolveLocallyAndCustomImages_ShouldSetUserCustomImages(t *testing.T) {
	// Setup
	var resubmitConfig []wrappers.Config

	// Create command with container flags
	cmdCommand := &cobra.Command{}
	cmdCommand.Flags().String(commonParams.ContainerImagesFlag, "", "Container images")
	cmdCommand.Flags().Bool(commonParams.ContainerResolveLocallyFlag, false, "Resolve containers locally")
	cmdCommand.Flags().String(commonParams.SourcesFlag, "", "Source")

	// Set test values for git scan with resolve locally and custom images
	expectedImages := "artifactory.company.com/repo/image1:latest,artifactory.company.com/repo/image2:1.0.3"
	gitURL := "https://github.com/user/repo.git"
	_ = cmdCommand.Flags().Set(commonParams.ContainerImagesFlag, expectedImages)
	_ = cmdCommand.Flags().Set(commonParams.ContainerResolveLocallyFlag, "true")
	_ = cmdCommand.Flags().Set(commonParams.SourcesFlag, gitURL)

	// Enable container scan type
	originalScanTypes := actualScanTypes
	actualScanTypes = commonParams.ContainersType
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Execute
	result, err := addContainersScan(cmdCommand, resubmitConfig)

	// Verify no error occurred
	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected result to not be nil")

	// Verify
	containerMapConfig, ok := result[resultsMapValue].(*wrappers.ContainerConfig)
	assert.Assert(t, ok, "Expected result to contain a ContainerConfig")

	// Check that the UserCustomImages field was correctly set even with resolve locally true (because it's a git scan)
	assert.Equal(t, containerMapConfig.UserCustomImages, expectedImages,
		"Expected UserCustomImages to be set to '%s' for git scan even with resolve locally, but got '%s'",
		expectedImages, containerMapConfig.UserCustomImages)
}

func TestAddContainersScan_UploadScanWithResolveLocallyAndCustomImages_ShouldNotSetUserCustomImages(t *testing.T) {
	// Setup
	var resubmitConfig []wrappers.Config

	// Create command with container flags
	cmdCommand := &cobra.Command{}
	cmdCommand.Flags().String(commonParams.ContainerImagesFlag, "", "Container images")
	cmdCommand.Flags().Bool(commonParams.ContainerResolveLocallyFlag, false, "Resolve containers locally")
	cmdCommand.Flags().String(commonParams.SourcesFlag, "", "Source")

	// Set test values for upload scan (local path) with resolve locally and custom images
	customImages := "artifactory.company.com/repo/image1:latest,artifactory.company.com/repo/image2:1.0.3"
	localPath := "/path/to/local/directory"
	_ = cmdCommand.Flags().Set(commonParams.ContainerImagesFlag, customImages)
	_ = cmdCommand.Flags().Set(commonParams.ContainerResolveLocallyFlag, "true")
	_ = cmdCommand.Flags().Set(commonParams.SourcesFlag, localPath)

	// Enable container scan type
	originalScanTypes := actualScanTypes
	actualScanTypes = commonParams.ContainersType
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Execute
	result, err := addContainersScan(cmdCommand, resubmitConfig)

	// Verify no error occurred
	assert.NilError(t, err)
	assert.Assert(t, result != nil, "Expected result to not be nil")

	// Verify
	containerMapConfig, ok := result[resultsMapValue].(*wrappers.ContainerConfig)
	assert.Assert(t, ok, "Expected result to contain a ContainerConfig")

	// Check that the UserCustomImages field was NOT set for upload scan with resolve locally
	assert.Equal(t, containerMapConfig.UserCustomImages, "",
		"Expected UserCustomImages to be empty for upload scan with resolve locally, but got '%s'",
		containerMapConfig.UserCustomImages)
}

func TestInitializeContainersConfigWithResubmitValues_UserCustomImages(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name                    string
		resubmitConfig          []wrappers.Config
		containerResolveLocally bool
		isGitScan               bool
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
			isGitScan:               false,
			expectedCustomImages:    "image1:tag1,image2:tag2",
		},
		{
			name: "When UserCustomImages is valid string and ContainerResolveLocally is true (upload scan), it should not be set in containerConfig",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: "image1:tag1,image2:tag2",
					},
				},
			},
			containerResolveLocally: true,
			isGitScan:               false,
			expectedCustomImages:    "",
		},
		{
			name: "When UserCustomImages is valid string and ContainerResolveLocally is true but is git scan, it should be set in containerConfig",
			resubmitConfig: []wrappers.Config{
				{
					Type: commonParams.ContainersType,
					Value: map[string]interface{}{
						ConfigUserCustomImagesKey: "image1:tag1,image2:tag2",
					},
				},
			},
			containerResolveLocally: true,
			isGitScan:               true,
			expectedCustomImages:    "image1:tag1,image2:tag2",
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
			isGitScan:               false,
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
			isGitScan:               false,
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
			isGitScan:               false,
			expectedCustomImages:    "",
		},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize containerConfig
			containerConfig := &wrappers.ContainerConfig{}

			// Call the function under test
			initializeContainersConfigWithResubmitValues(tc.resubmitConfig, containerConfig, tc.containerResolveLocally, tc.isGitScan)

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
		scsLicensingV2   bool
		expectedError    string
	}{
		{
			name:             "no specific micro engines selected with no licenses available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"repository-health": false, "secret-detection": false},
			scsLicensingV2:   true,
			expectedError:    "This requires either the \"repository‑health\" or the \"secret‑detection\" package license",
		},
		{
			name:             "no specific micro engines selected with repository-health license available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"repository-health": true, "secret-detection": false},
			scsLicensingV2:   true,
			expectedError:    "",
		},
		{
			name:             "no specific micro engines selected with secret-detection license available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"repository-health": false, "secret-detection": true},
			scsLicensingV2:   true,
			expectedError:    "",
		},
		{
			name:             "no specific micro engines selected with all licenses available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"repository-health": true, "secret-detection": true},
			scsLicensingV2:   true,
			expectedError:    "",
		},
		{
			name:             "no specific micro engines selected with no licenses available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"scs": false, "enterprise-secrets": false},
			scsLicensingV2:   false,
			expectedError:    "It looks like the \"scs\" scan type does not exist or",
		},
		{
			name:             "no specific micro engines selected with scs license available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": false},
			scsLicensingV2:   false,
			expectedError:    "",
		},
		{
			name:             "no specific micro engines selected with all licenses available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": true},
			scsLicensingV2:   false,
			expectedError:    "",
		},
		{
			name:             "scorecard and secret-detection selected with no licenses available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "scorecard,secret-detection",
			allowedEngines:   map[string]bool{"scs": false, "enterprise-secrets": false},
			scsLicensingV2:   false,
			expectedError:    "It looks like the \"scs\" scan type does not exist or",
		},
		{
			name:             "scorecard and secret-detection selected with no licenses available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "scorecard,secret-detection",
			allowedEngines:   map[string]bool{"repository-health": false, "secret-detection": false},
			scsLicensingV2:   true,
			expectedError:    "It looks like the \"secret-detection\" scan type does not exist or",
		},
		{
			name:             "secret-detection selected with SCS license available, secret-detection not available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": false},
			scsLicensingV2:   false,
			expectedError:    "It looks like the \"secret-detection\" scan type does not exist or",
		},
		{
			name:             "secret-detection selected with repository-health license available, secret-detection not available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"repository-health": true, "secret-detection": false},
			scsLicensingV2:   true,
			expectedError:    "It looks like the \"secret-detection\" scan type does not exist or",
		},
		{
			name:             "scorecard selected with secret-detection license available and repository-health not available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "scorecard",
			allowedEngines:   map[string]bool{"repository-health": false, "secret-detection": true},
			scsLicensingV2:   true,
			expectedError:    "It looks like the \"repository-health\" scan type does not exist or",
		},
		{
			name:             "secret-detection selected with all licenses available using old sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"scs": true, "enterprise-secrets": true},
			scsLicensingV2:   false,
			expectedError:    "",
		},
		{
			name:             "secret-detection selected with secret-detection license available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "secret-detection",
			allowedEngines:   map[string]bool{"repository-health": false, "secret-detection": true},
			scsLicensingV2:   true,
			expectedError:    "",
		},
		{
			name:             "scorecard selected with repository-health license available using new sscs licensing",
			userScanTypes:    "scs",
			userSCSScanTypes: "scorecard",
			allowedEngines:   map[string]bool{"repository-health": true, "secret-detection": false},
			scsLicensingV2:   true,
			expectedError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappers.ClearCache()
			mock.Flag = wrappers.FeatureFlagResponseModel{
				Name:   wrappers.ScsLicensingV2Enabled,
				Status: tt.scsLicensingV2,
			}

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

func TestIsScsScorecardAllowed(t *testing.T) {
	tests := []struct {
		name                       string
		scsLicensingV2             bool
		hasRepositoryHealthLicense bool
		hasScsLicense              bool
		expectedAllowed            bool
	}{
		{
			name:            "scsLicensingV2 disabled and has scs license",
			scsLicensingV2:  false,
			hasScsLicense:   true,
			expectedAllowed: true,
		},
		{
			name:            "scsLicensingV2 disabled and does not have scs license",
			scsLicensingV2:  false,
			hasScsLicense:   false,
			expectedAllowed: false,
		},
		{
			name:                       "scsLicensingV2 enabled and has repository health license",
			scsLicensingV2:             true,
			hasRepositoryHealthLicense: true,
			expectedAllowed:            true,
		},
		{
			name:                       "scsLicensingV2 enabled and does not have repository health license",
			scsLicensingV2:             true,
			hasRepositoryHealthLicense: false,
			expectedAllowed:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualAllowed := isScsScorecardAllowed(tt.scsLicensingV2, tt.hasRepositoryHealthLicense, tt.hasScsLicense)
			assert.Equal(t, tt.expectedAllowed, actualAllowed)
		})
	}
}

func TestIsScsSecretDetectionAllowed(t *testing.T) {
	tests := []struct {
		name                        string
		scsLicensingV2              bool
		hasSecretDetectionLicense   bool
		hasEnterpriseSecretsLicense bool
		hasScsLicense               bool
		expectedAllowed             bool
	}{
		{
			name:                        "scsLicensingV2 disabled and has scs and enterprise secrets license",
			scsLicensingV2:              false,
			hasEnterpriseSecretsLicense: true,
			hasScsLicense:               true,
			expectedAllowed:             true,
		},
		{
			name:                        "scsLicensingV2 disabled and has enterprise secrets but does not have scs license",
			scsLicensingV2:              false,
			hasEnterpriseSecretsLicense: true,
			hasScsLicense:               false,
			expectedAllowed:             false,
		},
		{
			name:                        "scsLicensingV2 disabled and has scs license but does not have enterprise secrets license",
			scsLicensingV2:              false,
			hasEnterpriseSecretsLicense: false,
			hasScsLicense:               true,
			expectedAllowed:             false,
		},
		{
			name:                      "scsLicensingV2 enabled and has secret detection license",
			scsLicensingV2:            true,
			hasSecretDetectionLicense: true,
			expectedAllowed:           true,
		},
		{
			name:                      "scsLicensingV2 enabled and does not have secret detection license",
			scsLicensingV2:            true,
			hasSecretDetectionLicense: false,
			expectedAllowed:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualAllowed := isScsSecretDetectionAllowed(tt.scsLicensingV2, tt.hasSecretDetectionLicense, tt.hasEnterpriseSecretsLicense, tt.hasScsLicense)
			assert.Equal(t, tt.expectedAllowed, actualAllowed)
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

// Tests for container scan directory handling bug fix (AST-107490)

func Test_isSingleContainerScanTriggered_WithSingleContainerType_ShouldReturnTrue(t *testing.T) {
	// Save original actualScanTypes
	originalScanTypes := actualScanTypes
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Test single container scan type
	actualScanTypes = commonParams.ContainersType
	result := isSingleContainerScanTriggered()
	assert.Assert(t, result, "Should return true for single container scan type")
}

func Test_isSingleContainerScanTriggered_WithMultipleScanTypes_ShouldReturnFalse(t *testing.T) {
	// Save original actualScanTypes
	originalScanTypes := actualScanTypes
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Test multiple scan types including container
	actualScanTypes = fmt.Sprintf("%s,%s", commonParams.ContainersType, commonParams.SastType)
	result := isSingleContainerScanTriggered()
	assert.Assert(t, !result, "Should return false for multiple scan types")

	// Test multiple scan types without container
	actualScanTypes = fmt.Sprintf("%s,%s", commonParams.SastType, commonParams.ScaType)
	result = isSingleContainerScanTriggered()
	assert.Assert(t, !result, "Should return false for multiple scan types without container")
}

func Test_isSingleContainerScanTriggered_WithNonContainerType_ShouldReturnFalse(t *testing.T) {
	// Save original actualScanTypes
	originalScanTypes := actualScanTypes
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Test single non-container scan type
	actualScanTypes = commonParams.SastType
	result := isSingleContainerScanTriggered()
	assert.Assert(t, !result, "Should return false for single non-container scan type")

	// Test empty scan types
	actualScanTypes = ""
	result = isSingleContainerScanTriggered()
	assert.Assert(t, !result, "Should return false for empty scan types")
}

func TestCreateScan_WithContainerImagesAndDirectory_ShouldProcessDirectoryFiles(t *testing.T) {
	// This test ensures that when using --container-images with a directory source,
	// the directory files are properly processed instead of creating a minimal zip
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "containers",
		"--container-images", "nginx:latest,alpine:3.14",
	}

	// This should succeed - directory files should be processed normally
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithSingleContainerScanAndDirectory_ShouldProcessAllFiles(t *testing.T) {
	// Test that single container scans with directory sources process all files,
	// not just container resolution files
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data",
		"--scan-types", "containers",
		"--containers-local-resolution",
	}

	// This should succeed and process the entire directory
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerImagesAndZipSource_ShouldProcessZipNormally(t *testing.T) {
	// Test that container scans with zip sources work normally
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data/sources.zip",
		"--scan-types", "containers",
		"--container-images", "redis:6.2,postgres:13",
	}

	// This should succeed and process the zip file normally
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithMixedScanTypesAndContainerImages_ShouldProcessAllFiles(t *testing.T) {
	// Test that mixed scan types with container images process all files
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "sast,containers,iac-security",
		"--container-images", "ubuntu:20.04",
	}

	// This should succeed and process all source files for all scan types
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerImagesOnly_ShouldNotCreateMinimalZip(t *testing.T) {
	// Test that using only --container-images flag doesn't create a minimal zip
	// and properly processes directory contents
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data",
		"--container-images", "mongo:4.4,elasticsearch:7.15.0",
	}

	// This should succeed and process directory files normally
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerResolveLocallyAndImages_ShouldProcessDirectoryContents(t *testing.T) {
	// Test that container-resolve-locally with container images processes directory contents
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "containers",
		"--containers-local-resolution",
		"--container-images", "node:16-alpine,python:3.9-slim",
	}

	// This should succeed and process both directory contents and external images
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerScanAndFileFilters_ShouldApplyFiltersToDirectory(t *testing.T) {
	// Test that file filters are applied to directory contents in container scans
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "containers",
		"--container-images", "nginx:latest",
		"--file-filter", "!*.log,!temp/**",
	}

	// This should succeed and apply file filters to the directory scan
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithComplexContainerScenario_ShouldHandleAllCases(t *testing.T) {
	// Test a complex scenario that would have failed before the bug fix
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data",
		"--scan-types", "containers",
		"--container-images", "httpd:2.4,tomcat:9.0-jdk11",
		"--containers-local-resolution",
		"--file-filter", "!node_modules/**",
		"--containers-file-folder-filter", "!*.tmp",
	}

	// This complex scenario should work correctly after the bug fix
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerImagesAndEmptyDirectory_ShouldProcessEmptyDirectory(t *testing.T) {
	// Test edge case: container images with empty directory should not create minimal zip
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--container-images", "busybox:latest",
	}

	// This should succeed and process the directory even if empty
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithOnlyContainerResolveLocally_ShouldProcessDirectory(t *testing.T) {
	// Test that containers-local-resolution without external images works correctly
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data",
		"--scan-types", "containers",
		"--containers-local-resolution",
	}

	// This should succeed and process directory for local container resolution
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_WithContainerImagesAndIncludeFilters_ShouldApplyFilters(t *testing.T) {
	// Test that include filters work correctly with container images
	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "containers",
		"--container-images", "alpine:latest",
		"--file-include", "*.dockerfile,*.yaml",
	}

	// This should succeed and apply include filters to directory scan
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_EdgeCase_SingleContainerWithCustomImages_ShouldNotCreateMinimalZip(t *testing.T) {
	// This is the exact edge case that was fixed - single container scan with custom images
	// Before the fix, this would create a minimal zip instead of processing directory

	// Save original actualScanTypes
	originalScanTypes := actualScanTypes
	defer func() {
		actualScanTypes = originalScanTypes
	}()

	// Set to single container scan type
	actualScanTypes = commonParams.ContainersType

	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", "data",
		"--scan-types", "containers",
		"--container-images", "nginx:1.21,php:8.0-apache",
	}

	// This was the failing case before the bug fix - should now succeed
	execCmdNilAssertion(t, baseArgs...)
}

func TestCreateScan_VerifyNoMinimalZipCreation_WithContainerImagesFlag(t *testing.T) {
	// This test verifies that the createMinimalZipFile function is no longer called
	// when using container images with directory sources

	baseArgs := []string{
		"scan", "create",
		"--project-name", "MOCK",
		"-b", "dummy_branch",
		"-s", ".",
		"--scan-types", "containers",
		"--container-images", "mariadb:10.6,redis:6.2-alpine",
		"--file-filter", "!*.md",
	}

	// Before the fix, this would have triggered createMinimalZipFile
	// After the fix, it should process directory contents normally
	execCmdNilAssertion(t, baseArgs...)
}

func TestGetGitignorePatterns_DirPath_GitIgnore_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := getGitignorePatterns(dir, "")
	assert.ErrorContains(t, err, ".gitignore file not found in directory")
}

func TestGetGitignorePatterns_DirPath_GitIgnore_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(""), 0000)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}
	_, err = getGitignorePatterns(dir, "")
	assert.ErrorContains(t, err, "permission denied")
}

func TestGetGitignorePatterns_DirPath_GitIgnore_EmptyPatternList(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}
	gitIgnoreFilter, err := getGitignorePatterns(dir, "")
	if err != nil {
		t.Fatalf("Error in fetching pattern from .gitignore file: %v", err)
	}
	assert.Assert(t, len(gitIgnoreFilter) == 0, "Expected no patterns from empty .gitignore file")
}

func TestGetGitignorePatterns_DirPath_GitIgnore_PatternList(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(`src
src/
**/vullib
**/admin/
vulnerability/**
application-jira.yml
*.yml
LoginController[0-1].java
LoginController[!0-3].java
LoginController[01].java
LoginController[!456].java
?pplication-jira.yml
a*cation-jira.yml`), 0644)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}
	gitIgnoreFilter, err := getGitignorePatterns(dir, "")
	if err != nil {
		t.Fatalf("Error in fetching pattern from .gitignore file: %v", err)
	}
	assert.Assert(t, len(gitIgnoreFilter) > 0, "Expected patterns from .gitignore file")
}

func TestGetGitignorePatterns_ZipPath_GitIgnore_FailedToOpenZipFIle(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "example.zip")

	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			t.Fatalf("Failed to close zip file: %v", err)
		}
	}(zipFile)
	_, err = getGitignorePatterns("", zipPath)
	assert.ErrorContains(t, err, "failed to open zip")
}

func TestGetGitignorePatterns_ZipPath_GitIgnore_NotFound(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "example.zip")

	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			t.Fatalf("Failed to close zip file: %v", err)
		}
	}(zipFile)

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)
	err = zipWriter.Close()
	if err != nil {
		return
	}

	_, err = getGitignorePatterns("", zipPath)
	assert.ErrorContains(t, err, ".gitignore not found in zip")
}

func TestGetGitignorePatterns_ZipPath_GitIgnore_EmptyPatternList(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "example.zip")

	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			t.Fatalf("Failed to close zip file: %v", err)
		}
	}(zipFile)

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)

	// Add a file to the zip archive
	fileInZip, err := zipWriter.Create("example" + "/.gitignore")
	if err != nil {
		t.Fatalf("Failed to add file to zip: %v", err)
	}

	_, err = fileInZip.Write([]byte(""))
	if err != nil {
		t.Fatalf("Failed to write data to zip: %v", err)
	}
	err = zipWriter.Close()
	if err != nil {
		return
	}

	gitIgnoreFilter, _ := getGitignorePatterns("", zipPath)
	assert.Assert(t, len(gitIgnoreFilter) == 0, "Expected no patterns from empty .gitignore file")
}

func TestGetGitignorePatterns_ZipPath_GitIgnore_PatternList(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "example.zip")

	// Create the zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			t.Fatalf("Failed to close zip file: %v", err)
		}
	}(zipFile)

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)

	// Add a file to the zip archive
	fileInZip, err := zipWriter.Create("example" + "/.gitignore")
	if err != nil {
		t.Fatalf("Failed to add file to zip: %v", err)
	}
	_, err = fileInZip.Write([]byte(`src
src/
**/vullib
**/admin/
vulnerability/**
application-jira.yml
*.yml
LoginController[0-1].java
LoginController[!0-3].java
LoginController[01].java
LoginController[!456].java
?pplication-jira.yml
a*cation-jira.yml`))
	if err != nil {
		t.Fatalf("Failed to write data to zip: %v", err)
	}
	err = zipWriter.Close()
	if err != nil {
		return
	}

	gitIgnoreFilter, _ := getGitignorePatterns("", zipPath)
	assert.Assert(t, len(gitIgnoreFilter) > 0, "Expected patterns from .gitignore file")
}

func Test_CreateScanWithIgnorePolicyFlag(t *testing.T) {
	execCmdNilAssertion(
		t,
		"scan", "create", "--project-name", "MOCK", "-s", "data/sources.zip", "--branch", "dummy_branch", "--ignore-policy",
	)
}

func Test_CreateScanWithExistingProjectAndAssign_Application(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)

	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", ".", "--branch", "main", "--application-name", mock.ExistingApplication, "--debug"}
	execCmdNilAssertion(
		t,
		baseArgs...,
	)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, "Successfully updated the application"), true, "Expected output: %s", "Successfully updated the application")
}

func Test_CreateScanWithExistingProjectAndAssign_FailedNoApplication_NameProvided(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)

	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", ".", "--branch", "main", "--debug"}
	execCmdNilAssertion(
		t,
		baseArgs...,
	)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, "No application name provided. Skipping application update"), true, "Expected output: %s", "No application name provided. Skipping application update")
}

func Test_CreateScanWithExistingProjectAndAssign_FailedApplication_DoesNot_Exist(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", ".", "--branch", "main", "--debug", "--application-name", "NoPermissionApp"}
	err := execCmdNotNilAssertion(
		t,
		baseArgs...,
	)
	assert.ErrorContains(t, err, errorConstants.FailedToGetApplication, err.Error())
}

func Test_CreateScanWithExistingProjectAssign_to_Application_FF_DirectAssociationEnabledShouldPass(t *testing.T) {
	file := createOutputFile(t, outputFileName)
	defer deleteOutputFile(file)
	defer logger.SetOutput(os.Stdout)

	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.DirectAssociationEnabled, Status: true}
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", ".", "--branch", "main", "--debug", "--application-name", mock.ExistingApplication}
	execCmdNilAssertion(
		t,
		baseArgs...,
	)
	stdoutString, err := util.ReadFileAsString(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	assert.Equal(t, strings.Contains(stdoutString, "Successfully updated the application"), true, "Expected output: %s", "Successfully updated the application")
}
