// +build !integration

package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

func TestQueryHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "query")
	assert.NilError(t, err)
}

func TestQueryNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "query")
	assert.NilError(t, err)
}

func TestRunImportCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "import", "--repo", "./payloads/nonsense.json")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "-v", "query", "import", "--repo", "./payloads/uploads.json", "--name", "mock")
	assert.NilError(t, err)
}

func TestRunImportCommandWithNoRepo(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "import")
	assert.Assert(t, err != nil)
}

func TestRunCLoneCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "clone", "mock")
	assert.NilError(t, err)
	wd, _ := os.Getwd()
	mockRepoFile, err := os.Open(filepath.Join(wd, queriesRepoDistFileName))
	assert.NilError(t, err, "failed to open repo mock file")
	bytes, err := ioutil.ReadAll(mockRepoFile)
	assert.NilError(t, err, "failed to read repo mock file")
	assert.Assert(t, string(bytes) == "mock")
}

func TestRunListCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "list")
	assert.NilError(t, err)
}

func TestRunActivateCommandBlank(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "activate")
	assert.Assert(t, err != nil)
}

func TestRunActivateCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "activate", "mock")
	assert.NilError(t, err)
}

func TestRunDeleteCommandBlank(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "delete")
	assert.Assert(t, err != nil)
}

func TestRunDeleteCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "query", "delete", "mock")
	assert.NilError(t, err)
}
