// +build !integration

package commands

//import (
//	"testing"
//
//	"gotest.tools/assert/cmp"
//
//	"gotest.tools/assert"
//)
/**
func TestSastResourcesScansCommand(t *testing.T) {
	err := executeASTTestCommand("utils", "sast-rm", "scans", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "scans")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "scans", "--format", "table")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "scans", "-h")
	assert.NilError(t, err)
}

func TestSastResourcesEnginesCommand(t *testing.T) {
	err := executeASTTestCommand("utils", "sast-rm", "engines", "--format", "table")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "engines")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "engines", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "engines", "-h")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "engines", "set-tags", "-i", "12234", "kuku=riku")
	assert.NilError(t, err)

	err = executeASTTestCommand("utils", "sast-rm", "engines", "set-tags", "kuku=riku")
	assert.Error(t, err, MissingEngineIDFlagError)
}

func TestSastResourcesStatsCommand(t *testing.T) {
	err := executeASTTestCommand("utils", "sast-rm", "stats", "-v", "-r", "hour")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "stats", "-v", "--resolution", "hour")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "stats", "-v", "-r", "day")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "stats", "-v", "--resolution", "minute")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "-v", "stats", "--format", "json")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "stats", "-v", "--format", "list")
	assert.NilError(t, err)
	err = executeASTTestCommand("utils", "sast-rm", "stats", "-h")
	assert.NilError(t, err)

	err = executeASTTestCommand("utils", "sast-rm", "stats", "-r", "sdfsd")
	assert.Assert(t, cmp.Equal(err.Error(), "unknown resolution sdfsd"))
}

func TestSastRMPoolCommand(t *testing.T) {
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "list"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "create", "-d", "the-pool"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "create", "--description", "the-pool"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "delete", "-v", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "projects", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "project-tags", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "engines", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "engine-tags", "get", "--pool-id", "some-pool-id"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "projects", "set", "--pool-id", "some-pool-id", "project1"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "project-tags", "set", "--pool-id", "some-pool-id",
		"tag1=value1", "tag2=value2"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "engines", "set", "--pool-id", "some-pool-id", "engine1", "engine2"))
	assert.NilError(t, executeASTTestCommand("utils", "sast-rm", "pools", "engine-tags", "set", "--pool-id", "some-pool-id", "tag1=value1"))

	assert.Error(t, executeASTTestCommand("utils", "sast-rm", "pools", "projects", "get"), MissingPoolIDFlagError)
	assert.Error(t, executeASTTestCommand("utils", "sast-rm", "pools", "project-tags", "get"), MissingPoolIDFlagError)
	assert.Error(t, executeASTTestCommand("utils", "sast-rm", "pools", "engines", "set"), MissingPoolIDFlagError)
	assert.Error(t, executeASTTestCommand("utils", "sast-rm", "pools", "engine-tags", "set"), MissingPoolIDFlagError)
}

func executeASTTestCommand(args ...string) error {
	cmd := createASTTestCommand()
	return executeTestCommand(cmd, args...)
}
*/
