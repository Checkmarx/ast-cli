//go:build !integration

package commands

import (
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/spf13/viper"
)

const (
	unknownFlag          = "unknown flag: --chibutero"
	blankSpace           = " "
	errorMissingBranch   = "Failed creating a scan: Please provide a branch"
	dummyRepo            = "https://www.dummy-repo.com"
	errorSourceBadFormat = "Failed creating a scan: Input in bad format: Sources input has bad format: "
	scaPathError         = "ScaResolver error: exec: \"resolver\": executable file not found in "
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
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "kics")...)
	execCmdNilAssertion(t, append(baseArgs, "--scan-types", "sca")...)
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
