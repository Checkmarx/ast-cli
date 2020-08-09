// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

const (
	unknownFlag        = "unknown flag: --chibutero"
	subcommandRequired = "subcommand is required"
)

func TestScanHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "scan")
	assert.NilError(t, err)
}

func TestScanNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "scan")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == subcommandRequired)
}

func TestRunCreateScanCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input-file", "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
	err = executeTestCommand(cmd, "-v", "scan", "create", "--input-file", "./payloads/uploads.json", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateScanCommandWithNoInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a scan: no input was given\n")
}

func TestRunCreateScanCommandWithInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "{\"project\":{\"id\":\"test\",\"type\":\"upload\",\"handler\":"+
		"{\"url\":\"MOSHIKO\"},\"tags\":{}},\"config\":"+
		"[{\"type\":\"sast\",\"value\":{\"presetName\":\"Default\"}}]}", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateScanCommandWithInputAndTags(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "{\"project\":{\"id\":\"test\",\"type\":\"upload\",\"handler\":"+
		"{\"url\":\"MOSHIKO\"},\"tags\":{}},\"config\":"+
		"[{\"type\":\"sast\",\"value\":{\"presetName\":\"Default\"}}], \"tags\": {\"test\": \"test1\"}}",
		"--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateScanCommandWithInputPretty(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "--format", "list", "scan",
		"create", "--input", "{\"project\":{\"id\":\"test\",\"type\":\"upload\",\"handler\":"+
			"{\"url\":\"MOSHIKO\"},\"tags\":{}},\"config\":"+
			"[{\"type\":\"sast\",\"value\":{\"presetName\":\"Default\"}}],\"tags\":{}}", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateScanCommandWithInputBadFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "[]", "--sources", "./payloads/sources.zip")
	assert.Assert(t, err != nil)
}

func TestRunGetScanByIdCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "show")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed showing a scan: Please provide a scan ID")
}

func TestRunGetScanByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "show", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetScanByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "show", "MOCK")
	assert.NilError(t, err)
}
func TestRunDeleteScanByIdCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed deleting a scan: Please provide a scan ID")
}

func TestRunDeleteByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteScanByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "MOCK")
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
