// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestBFLHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "bfl")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIDCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "bfl")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed getting BFL: Please provide a scan ID")
}

func TestRunGetBFLByScanIDCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "bfl", "list", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIDCommandPretty(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "bfl", "--format", "list", "list", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIDCommandFilters(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "bfl", "--format", "table", "list", "MOCK", "--filter")
	assert.Assert(t, err != nil)
	cmd = createASTTestCommand()
	err = executeTestCommand(cmd, "-v", "bfl", "--format", "list", "list", "MOCK", "--filter", "a=b=c")
	assert.Assert(t, err != nil)
	cmd = createASTTestCommand()
	err = executeTestCommand(cmd, "-v", "bfl", "--format", "table", "list", "MOCK", "--filter", "a")
	assert.Assert(t, err != nil)
	cmd = createASTTestCommand()
	err = executeTestCommand(cmd, "-v", "bfl", "--format", "list", "list", "MOCK", "--filter", "a=b")
	assert.NilError(t, err)
}
