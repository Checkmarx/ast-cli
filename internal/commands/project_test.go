// +build !integration

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
	assert.Assert(t, err == nil)
}

func TestRunCreateProjectCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create", "--project-name", "test_project")
	assert.NilError(t, err)
}

func TestRunCreateProjectCommandWithNoInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Project name is required")
}

/* Test functionality removed from CLI
func TestRunCreateProjectCommandWithInput(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "create", "--input", "{\"id\": \"test_project\"}")
	assert.NilError(t, err)
}
*/

func TestRunCreateProjectCommandWithInvalidFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "--format", "non-sense", "-v", "project", "create", "--project-name", "test_project")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed creating a project: Invalid format non-sense")
}

func TestRunCreateProjectCommandWithInputList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "--format", "list", "-v", "project", "create", "--project-name", "test_project")
	assert.NilError(t, err)
}

func TestRunGetProjectByIdCommandNoScanID(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "show")
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
	err := executeTestCommand(cmd, "-v", "project", "show", "MOCK")
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
	err := executeTestCommand(cmd, "-v", "project", "list")
	assert.NilError(t, err)
}

func TestRunGetAllProjectsCommandFlagNonExist(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "list", "--chibutero")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetAllProjectsCommandWithLimit(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "list", "--filter", "limit=40")
	assert.NilError(t, err)
}

func TestRunGetAllProjectsCommandWithLimitList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "list", "--format", "list", "--filter", "--limit=40")
	assert.NilError(t, err)
}

func TestRunGetAllProjectsCommandWithOffset(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "list", "--filter", "offset=150")
	assert.NilError(t, err)
}

func TestRunGetProjectTagsCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "project", "tags")
	assert.NilError(t, err)
}
