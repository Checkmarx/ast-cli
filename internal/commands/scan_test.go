package commands

import (
	"testing"

	"gotest.tools/assert"
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
	assert.Assert(t, err.Error() == "Failed creating a scan: Failed to open input file: open "+
		"./payloads/nonsense.json: The system cannot find the file specified.")
	err = executeTestCommand(cmd, "-v", "scan", "create", "--inputFile", "./payloads/uploads.json", "--sources", "./payloads/sources.zip")
	assert.NilError(t, err)
}

func TestRunCreateCommandWithNoInput(t *testing.T) {
	cmd := createASTCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a scan: no input was given\n")
}

func TestRunGetScanByIdCommand(t *testing.T) {
}
func TestRunGetScanByIdCommandNoArg(t *testing.T) {
}
