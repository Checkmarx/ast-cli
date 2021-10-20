//go:build !integration

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
	execCmdNilAssertion(t, "project", "create", "--project-name", "test_project")
}

func TestRunCreateProjectCommandWithNoInput(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "create")
	assert.Assert(t, err.Error() == "Project name is required")
}

func TestRunCreateProjectCommandWithInvalidFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "--format", "non-sense", "project", "create", "--project-name", "test_project")
	assert.Assert(t, err.Error() == "Failed creating a project: Invalid format non-sense")
}

func TestRunCreateProjectCommandWithInputList(t *testing.T) {
	args := []string{"--format", "list", "project", "create", "--project-name", "test_project", "--branch", "dummy-branch"}
	execCmdNilAssertion(t, args...)
}

func TestRunGetProjectByIdCommandNoScanID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "show")
	assert.Assert(t, err.Error() == "Failed getting a project: Please provide a project ID")
}

func TestRunGetProjectByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "get", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetProjectByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "project", "show", "--project-id", "MOCK")
}

func TestRunDeleteProjectByIdCommandNoProjectID(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "delete")
	assert.Assert(t, err.Error() == "Failed deleting a project: Please provide a project ID")
}

func TestRunDeleteProjectByIdCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "scan", "project", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunDeleteProjectByIdCommand(t *testing.T) {
	execCmdNilAssertion(t, "project", "delete", "--project-id", "MOCK")
}

func TestRunGetAllProjectsCommand(t *testing.T) {
	execCmdNilAssertion(t, "project", "list")
}

func TestRunGetAllProjectsCommandFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "list", "--chibutero")
	assert.Assert(t, err.Error() == unknownFlag)
}

func TestRunGetAllProjectsCommandWithLimit(t *testing.T) {
	execCmdNilAssertion(t, "project", "list", "--filter", "limit=40")
}

func TestRunGetAllProjectsCommandWithLimitList(t *testing.T) {
	execCmdNilAssertion(t, "project", "list", "--format", "list", "--filter", "--limit=40")
}

func TestRunGetAllProjectsCommandWithOffset(t *testing.T) {
	execCmdNilAssertion(t, "project", "list", "--filter", "offset=150")
}

func TestRunGetProjectTagsCommand(t *testing.T) {
	execCmdNilAssertion(t, "project", "tags")
}

func TestRunGetProjectBranchesCommand(t *testing.T) {
	execCmdNilAssertion(t, "project", "branches", "--project-id", "MOCK")
}

func TestRunGetProjectBranchesCommandWithFilter(t *testing.T) {
	execCmdNilAssertion(t, "project", "branches", "--project-id", "MOCK", "--filter", "branch-name=ma,offset=1")
}
