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

func TestSastRMPoolCommand(t *testing.T) {
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "list"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "create", "-d", "the-pool"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "create", "--description", "the-pool"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "delete", "-v", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "projects", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "project-tags", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "engines", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "engine-tags", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "projects", "set", "--pool-id", "some-pool-id", "project1"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "project-tags", "set", "--pool-id", "some-pool-id",
		"tag1=value1", "tag2=value2"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "engines", "set", "--pool-id", "some-pool-id", "engine1", "engine2"))
	assert.NilError(t, executeASTTestCommand("sast-rm", "pools", "engine-tags", "set", "--pool-id", "some-pool-id", "tag1=value1"))
}

func executeASTTestCommand(args ...string) error {
	cmd := createASTTestCommand()
	return executeTestCommand(cmd, args...)
}
