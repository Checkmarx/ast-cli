//go:build !integration
// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestProjectHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "project")
}

func TestProjectNoSub(t *testing.T) {
	execCmdNilAssertion(t, "project")
}

func TestRunCreateProjectCommandWithFile(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "create", "--project-name", "test_project")
}

func TestRunCreateProjectCommandWithNoInput(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "project", "create")
	assert.Assert(t, err.Error() == "Project name is required")
}

func TestRunCreateProjectCommandWithInvalidFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "--format", "non-sense", "-v", "project", "create", "--project-name", "test_project")
	assert.Assert(t, err.Error() == "Failed creating a project: Invalid format non-sense")
}

func TestRunCreateProjectCommandWithInputList(t *testing.T) {
	args := []string{"--format", "list", "-v", "project", "create", "--project-name", "test_project", "--branch", "dummy-branch"}
	execCmdNilAssertion(t, args...)
}

func TestRunGetProjectByIdCommandNoScanID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "project", "show")
	assert.Assert(t, err.Error() == "Failed getting a project: Please provide a project ID")
}

func TestRunGetProjectByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "project", "get", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetProjectByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "show", "--project-id", "MOCK")
}

func TestRunDeleteProjectByIdCommandNoProjectID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "project", "delete")
	assert.Assert(t, err.Error() == "Failed deleting a project: Please provide a project ID")
}

func TestRunDeleteProjectByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "scan", "project", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteProjectByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "delete", "--project-id", "MOCK")
}

func TestRunGetAllProjectsCommand(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "list")
}

func TestRunGetAllProjectsCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "-v", "project", "list", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetAllProjectsCommandWithLimit(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "list", "--filter", "limit=40")
}

func TestRunGetAllProjectsCommandWithLimitList(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "list", "--format", "list", "--filter", "--limit=40")
}

func TestRunGetAllProjectsCommandWithOffset(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "list", "--filter", "offset=150")
}

func TestRunGetProjectTagsCommand(t *testing.T) {
	execCmdNilAssertion(t, "-v", "project", "tags")
}
