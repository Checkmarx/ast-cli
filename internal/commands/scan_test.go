//go:build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"

	"github.com/spf13/viper"
)

const (
	unknownFlag = "unknown flag: --chibutero"
	blankSpace  = " "
)

func TestScanHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "scan")
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
	execCmdNilAssertion(t, "scan", "list", "--format", "list", "--filter", "statuses=Failed;Completed;Running,limit=500")
}

func TestRunGetAllCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "list", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunTagsCommand(t *testing.T) {
	execCmdNilAssertion(t, "scan", "tags")
}

func TestCreateScan(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "-b", "dummy_branch")
}

func TestCreateScanSourceDirectory(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "data", "--file-filter", "!.java", "-b", "dummy_branch")
}

func TestCreateScanSourceFile(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "data/sources.zip", "-b", "dummy_branch")
}

func TestCreateScanWithTrimmedSources(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", blankSpace+"."+blankSpace, "-b", "dummy_branch")
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", blankSpace+"data/"+blankSpace, "-b", "dummy_branch")
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", blankSpace+"https://www.dummy-repo.com"+blankSpace, "-b", "dummy_branch")
}

func TestCreateScanWrongFormatSource(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "invalidSource", "-b", "dummy_branch")
	assert.Assert(t, err.Error() == "Failed creating a scan: Input in bad format: Sources input has bad format: invalidSource")

	err = execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "http:", "-b", "dummy_branch")
	assert.Assert(t, err.Error() == "Failed creating a scan: Input in bad format: Sources input has bad format: http:")
}

func TestCreateScanWithScaResolver(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "data", "--sca-resolver", "nop", "-f", "!ScaResolver-win64", "-b", "dummy_branch")
}

func TestCreateScanWithScanTypes(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "--scan-types", "sast", "-b", "dummy_branch")
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "--scan-types", "kics", "-b", "dummy_branch")
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "--scan-types", "sca", "-b", "dummy_branch")
}

func TestCreateScanWithNoFilteredProjects(t *testing.T) {
	// Cover "createProject" when no project is filtered when finding the provided project
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK-NO-FILTERED-PROJECTS", "-s", "https://www.dummy-repo.com", "-b", "dummy_branch")
}

func TestCreateScanWithTags(t *testing.T) {
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "--tags", "dummy_tag:sub_dummy_tag", "-b", "dummy_branch")
}

func TestCreateScanBranches(t *testing.T) {
	// Test Missing branch either in flag and in the environment variable
	err := execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com")
	assert.Assert(t, err.Error() == "Failed creating a scan: Please provide a branch")

	// Bind cx_branch environment variable
	_ = viper.BindEnv("cx_branch", "CX_BRANCH")
	viper.SetDefault("cx_branch", "branch_from_environment_variable")

	// Test branch from environment variable. Since the cx_branch is bind the scan must run successfully without a branch flag defined
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com")

	// Test missing branch value
	err = execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "-b")
	assert.Assert(t, err.Error() == "flag needs an argument: 'b' in -b")

	// Test empty branch value
	err = execCmdNotNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "-b", "")
	assert.Assert(t, err.Error() == "Failed creating a scan: Please provide a branch")

	// Test defined branch value
	execCmdNilAssertion(t, "scan", "create", "--project-name", "MOCK", "-s", "https://www.dummy-repo.com", "-b", "branch_defined")
}
