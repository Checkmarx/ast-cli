package commands

import (
	"testing"

	"gotest.tools/assert"
)

const (
	unknownFlag = "unknown flag: --Chibutero"
)

func TestScanHelp(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "help", "scan")
	assert.NilError(t, err)
}

func TestScanNoSub(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "scan")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "subcommand is required")
}

func TestRunCreateCommandWithFile(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--inputFile", "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
	err = executeTestCommand(cmd, "-v", "scan", "create", "--inputFile", "./payloads/uploads.json", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateCommandWithNoInput(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a scan: no input was given\n")
}

func TestRunCreateCommandWithInput(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "{\"project\":{\"id\":\"test\",\"type\":\"upload\",\"handler\":"+
		"{\"url\":\"MOSHIKO\"},\"tags\":{}},\"config\":"+
		"[{\"type\":\"sast\",\"value\":{\"presetName\":\"Default\"}}],\"tags\":{}}", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateCommandWithInputBadFormat(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "[]", "--sources", "./payloads/sources.zip")
	assert.Assert(t, err != nil)
}

func TestRunGetScanByIdCommandNoScanID(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "get")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed getting a scan: Please provide a scan ID")
}

func TestRunGetScanByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "get", "--Chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetScanByIdCommand(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "get", "MOCK")
	assert.NilError(t, err)
}
func TestRunDeleteScanByIdCommandNoScanID(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed deleting a scan: Please provide a scan ID")
}

func TestRunGetDeleteByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "--Chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteScanByIdCommand(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "delete", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetAllCommand(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "get-all")
	assert.NilError(t, err)
}

func TestRunGetAllCommandFlagNonExist(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "get-all", "--Chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}
