package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestSastResourcesScansCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-resources", "scans", "--format", "pretty")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "scans")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "scans", "--format", "table")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sast-resources", "scans", "-h")
	assert.NilError(t, err)
}

func TestSastResourcesEnginesCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-resources", "engines", "--format", "table")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "engines")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "engines", "--format", "pretty")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "engines", "-h")
	assert.NilError(t, err)
}

func TestSastResourcesStatsCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-resources", "stats", "--format", "table")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "--format", "pretty")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "-h")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "resolution", "hour", "m", "running")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "r", "day", "metric", "scan-queued")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "resolution", "week")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "sr", "stats", "metric", "engine", "engine-waiting")
	assert.NilError(t, err)
}
