// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert/cmp"

	"gotest.tools/assert"
)

func TestSastResourcesScansCommand(t *testing.T) {
	err := executeASTTestCommand("sast-rm", "scans", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "scans")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "scans", "--format", "table")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "scans", "-h")
	assert.NilError(t, err)
}

func TestSastResourcesEnginesCommand(t *testing.T) {
	err := executeASTTestCommand("sast-rm", "engines", "--format", "table")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "engines")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "engines", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "engines", "-h")
	assert.NilError(t, err)
}

func TestSastResourcesStatsCommand(t *testing.T) {
	err := executeASTTestCommand("sast-rm", "stats", "-v", "-r", "hour")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "stats", "-v", "--resolution", "hour")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "stats", "-v", "-r", "day")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "stats", "-v", "--resolution", "minute")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "-v", "stats", "--format", "json")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "stats", "-v", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("sast-rm", "stats", "-h")
	assert.NilError(t, err)

	err = executeASTTestCommand("sast-rm", "stats", "-r", "sdfsd")
	assert.Assert(t, cmp.Equal(err.Error(), "unknown resolution sdfsd"))
}

func executeASTTestCommand(args ...string) error {
	cmd := createASTTestCommand()
	return executeTestCommand(cmd, args...)
}
