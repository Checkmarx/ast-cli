// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestResultHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "result")
	assert.NilError(t, err)
}

func TestRunGetResultsByScanIDCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "result", "list")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed listing results: Please provide a scan ID")
}
func TestRunGetResultsByScanIDCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "result", "list", "MOCK")
	assert.NilError(t, err)
}
