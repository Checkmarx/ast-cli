//go:build !integration

package commands

import (
	"reflect"
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/spf13/viper"
)

const (
	unknownFlag                   = "unknown flag: --chibutero"
	blankSpace                    = " "
	errorMissingBranch            = "Failed creating a scan: Please provide a branch"
	dummyRepo                     = "https://github.com/dummyuser/dummy_project.git"
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
	assert.Assert(t, err.Error() == "Failed finding groups: [invalidGroup]")
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
	want[" kics - low"] = 1
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
