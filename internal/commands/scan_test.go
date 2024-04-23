//go:build !integration

package commands

import (
	"reflect"
	"strings"
	"testing"

	applicationErrors "github.com/checkmarx/ast-cli/internal/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	unknownFlag                   = "unknown flag: --chibutero"
	blankSpace                    = " "
	errorMissingBranch            = "Failed creating a scan: Please provide a branch"
	dummyRepo                     = "https://github.com/dummyuser/dummy_project.git"
	dummyToken                    = "dummyToken"
	dummySSHRepo                  = "git@github.com:dummyRepo/dummyProject.git"
	errorSourceBadFormat          = "Failed creating a scan: Input in bad format: Sources input has bad format: "
	scaPathError                  = "ScaResolver error: exec: \"resolver\": executable file not found in "
	fileSourceFlag                = "--file"
	fileSourceValueEmpty          = "data/empty.Dockerfile"
	fileSourceValue               = "data/Dockerfile"
	fileSourceIncorrectValue      = "data/source.zip"
	fileSourceIncorrectValueError = "data/source.zip. Provided file is not supported by kics"
	fileSourceError               = "flag needs an argument: --file"
	engineFlag                    = "--engine"
	engineValue                   = "docker"
	invalidEngineValue            = "invalidengine"
	engineError                   = "flag needs an argument: --engine"
	additionalParamsFlag          = "--additional-params"
	additionalParamsValue         = "-v"
	additionalParamsError         = "flag needs an argument: --additional-params"
	scanCommand                   = "scan"
	kicsRealtimeCommand           = "kics-realtime"
	InvalidEngineMessage          = "Please verify if engine is installed"
	SCSScoreCardError             = "SCS Repo Token and SCS Repo URL are required, if scorecard is enabled"
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

func TestScanCreate_ExistingApplicationAndProject_CreateProjectUnderApplicationSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch")
}

func TestScanCreate_ApplicationNameIsNotExactMatch_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", "MOC", "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_ExistingProjectAndApplicationWithNoPermission_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.ApplicationDoesntExist, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_ExistingApplication_CreateNewProjectUnderApplicationSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "NewProject", "--application-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch")
}

func TestScanCreate_ExistingApplicationWithNoPermission_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "NewProject", "--application-name", mock.NoPermissionApp, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.ApplicationDoesntExistOrNoPermission)
}

func TestScanCreate_OnReceivingHttpBadRequestStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.FakeHTTPStatusBadRequest, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.FailedToGetApplication)
}

func TestScanCreate_OnReceivingHttpInternalServerErrorStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.FakeHTTPStatusInternalServerError, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.FailedToGetApplication)
}

func TestCreateScanInsideApplicationProjectExistNoPermissions(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "--application-name", mock.NoPermissionApp, "-s", dummyRepo, "-b", "dummy_branch")
	assert.Assert(t, err.Error() == applicationErrors.ApplicationDoesntExistOrNoPermission)
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

func TestCreateScanWithScanTypes(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch"}
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sast")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "iac-security")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sca")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sast,api-security")...)

	scsArgs := append(baseArgs, flag(commonParams.ScanTypes), "scs",
		flag(commonParams.SCSRepoURLFlag), "dummyURL",
		flag(commonParams.SCSRepoTokenFlag), "dummyToken")
	execCmdNilAssertion(t, scsArgs...)
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

func TestCreateScanWithProjectGroup(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan", "create", "--project-name", "invalidGroup", "-s", ".", "--project-groups", "invalidGroup",
	)
	assert.Assert(t, err.Error() == "Failed finding groups: [invalidGroup]", "\n the received error is:", err.Error())
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

	assert.Assert(t, util.Contains(expectedMessages, err.Error()))
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

func Test_parseThresholdSuccess(t *testing.T) {
	want := make(map[string]int)
	want["iac-security-low"] = 1
	threshold := " KICS - LoW=1"
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

func TestCreateScanProjecGroupsError(t *testing.T) {
	baseArgs := []string{scanCommand, "create", "--project-name", "MOCK", "-s", dummyRepo, "-b", "dummy_branch",
		"--debug", "--project-groups", "err"}
	err := execCmdNotNilAssertion(t, baseArgs...)
	assert.Error(t, err, "Failed updating a project: Failed finding groups: [err]", err.Error())
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

	result := addScaScan(cmdCommand, resubmitConfig)
	scaConfig := wrappers.ScaConfig{
		Filter:                "test",
		ExploitablePath:       "true",
		LastSastScanTime:      "1",
		PrivatePackageVersion: "1.1.1",
	}
	scaMapConfig := make(map[string]interface{})
	scaMapConfig[resultsMapType] = commonParams.ScaType
	scaMapConfig[resultsMapValue] = &scaConfig

	if !reflect.DeepEqual(result, scaMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scaMapConfig, result)
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

func TestCreateScanWithFastScanFlagIncorrectCase(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "--branch", "b", "--scan-types", "sast", "--file-source", "."}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--SAST-FAST-SCAN", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --SAST-FAST-SCAN", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--Sast-Fast-Scan", "true")...)
	assert.ErrorContains(t, err, "unknown flag: --Sast-Fast-Scan", err.Error())
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
	cmdCommand.PersistentFlags().Bool(commonParams.SastFastScanFlag, true, "Enable SAST Fast Scan")

	_ = cmdCommand.Execute()

	_ = cmdCommand.Flags().Set(commonParams.PresetName, "test")
	_ = cmdCommand.Flags().Set(commonParams.SastFilterFlag, "test")
	_ = cmdCommand.Flags().Set(commonParams.IncrementalSast, "true")

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

func TestCreateScanWithSCSScorecardShouldFail(t *testing.T) {
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

func TestCreateScanWithSCSSecretDetectionAndScorecard(t *testing.T) {
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

	result, _ := addSCSScan(cmdCommand)

	scsConfig := wrappers.SCSConfig{
		Twoms:     "true",
		Scorecard: "true",
		RepoURL:   dummyRepo,
		RepoToken: dummyToken,
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.ScsType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}

func TestCreateScanWithSCSSecretDetection(t *testing.T) {
	cmdCommand := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project",
		Long:  `Scan a project`,
	}
	cmdCommand.PersistentFlags().String(commonParams.SCSEnginesFlag, "", "SCS Engine flag")
	_ = cmdCommand.Execute()
	_ = cmdCommand.Flags().Set(commonParams.SCSEnginesFlag, "secret-detection")

	result, _ := addSCSScan(cmdCommand)

	scsConfig := wrappers.SCSConfig{
		Twoms: "true",
	}
	scsMapConfig := make(map[string]interface{})
	scsMapConfig[resultsMapType] = commonParams.ScsType
	scsMapConfig[resultsMapValue] = &scsConfig

	if !reflect.DeepEqual(result, scsMapConfig) {
		t.Errorf("Expected %+v, but got %+v", scsMapConfig, result)
	}
}
