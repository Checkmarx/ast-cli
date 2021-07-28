// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

const (
	unknownFlag = "unknown flag: --chibutero"
)

func TestScanHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "scan")
	assert.NilError(t, err)
}

func TestScanNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "scan")
	assert.Assert(t, err == nil)
}

func TestRunGetScanByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "show", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetScanByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "show", "--scan-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunDeleteScanByIdCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed deleting a scan: Please provide at least one scan ID")
}

func TestRunDeleteByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteScanByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "--scan-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunCancelScanByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "cancel", "--scan-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetAllCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list")
	assert.NilError(t, err)
}

func TestRunGetAllCommandList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list", "--format", "list")
	assert.NilError(t, err)
}

func TestRunGetAllCommandLimitList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list", "--format", "list", "--filter", "limit=40")
	assert.NilError(t, err)
}

func TestRunGetAllCommandOffsetList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list", "--format", "list", "--filter", "offset=0")
	assert.NilError(t, err)
}

func TestRunGetAllCommandStatusesList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list", "--format", "list", "--filter", "statuses=Failed;Completed;Running,limit=500")
	assert.NilError(t, err)
}

func TestRunGetAllCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "list", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunTagsCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "tags")
	assert.NilError(t, err)
}
