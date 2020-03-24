package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestProjectHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "project")
	assert.NilError(t, err)
}

func TestProjectNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "project")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == subcommandRequired)
}

func TestRunCreateProjectCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create", "--inputFile", "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
	err = executeTestCommand(cmd, "-v", "project", "create", "--inputFile", "./payloads/projects.json")
	assert.NilError(t, err)
}

func TestRunCreateProjectCommandWithNoInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a project: no input was given\n")
}

func TestRunCreateProjectCommandWithInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create", "--input", "{\"id\": \"test_project\"}")
	assert.NilError(t, err)
}

func TestRunCreateProjectCommandWithInputBadFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "create", "--input", "[]")
	assert.Assert(t, err != nil)
}

func TestRunGetProjectByIdCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "get")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed getting a project: Please provide a project ID")
}

func TestRunGetProjectByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "get", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetProjectByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "get", "MOCK")
	assert.NilError(t, err)
}
func TestRunDeleteProjectByIdCommandNoProjectID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "delete")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed deleting a project: Please provide a project ID")
}

func TestRunDeleteProjectByIdCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "scan", "project", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteProjectByIdCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "delete", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetAllProjectsCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "get-all")
	assert.NilError(t, err)
}

func TestRunGetAllProjectsCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "get-all", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetProjectTagsCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "tags")
	assert.NilError(t, err)
}
