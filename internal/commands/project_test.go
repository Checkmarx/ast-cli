//go:build !integration

package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"

	"gotest.tools/assert"

	"github.com/checkmarx/ast-cli/internal/commands/util"
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

func TestProjectCreate_ExistingApplication_CreateProjectUnderApplicationSuccessfully(t *testing.T) {
	execCmdNilAssertion(t, "project", "create", "--project-name", "test_project", "--application-name", "MOCK")
}

func TestProjectCreate_ExistingApplicationWithNoPermission_FailToCreateProject(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "create", "--project-name", "test_project", "--application-name", mock.NoPermissionApp)
	assert.Assert(t, err.Error() == errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestProjectCreate_OnReceivingHttpBadRequestStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "create", "--project-name", "test_project", "--application-name", mock.FakeBadRequest400)
	assert.Assert(t, err.Error() == errorConstants.FailedToGetApplication)
}

func TestProjectCreate_OnReceivingHttpInternalServerErrorStatusCode_FailedToCreateScan(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "create", "--project-name", "test_project", "--application-name", mock.FakeInternalServerError500)
	assert.Assert(t, err.Error() == errorConstants.FailedToGetApplication)
}

func TestRunCreateProjectCommandWithNoInput(t *testing.T) {
	err := execCmdNotNilAssertion(t, "project", "create")
	assert.Assert(t, err.Error() == errorConstants.ProjectNameIsRequired)
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

func TestRunProjectCreateInvalidGroup(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"project", "create", "--project-name", "invalidprj", "--groups", "invalidgroup")
	assert.Assert(t, err.Error() == "Failed finding groups: [invalidgroup]")
}

func TestCreateProjectMissingSSHValue(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", " ")...)
	assert.Error(t, err, "flag needs an argument: --ssh-key", err.Error())
}

func TestCreateProjectMissingRepoURLWithSSHValue(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key", "--repo-url")...)
	assert.Error(t, err, "flag needs an argument: --repo-url", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key", "--repo-url", "")...)
	assert.Error(t, err, "flag needs an argument: --repo-url", err.Error())

	err = execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key", "--repo-url", " ")...)
	assert.Error(t, err, "flag needs an argument: --repo-url", err.Error())
}

func TestCreateProjectMandatoryRepoURLWhenSSHKeyProvided(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key")...)

	assert.Error(t, err, mandatoryRepoURLError)
}

func TestCreateProjectInvalidRepoURLWithSSHKey(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key", "--repo-url", "https://github.com/dummyuser/dummy_project.git")...)

	assert.Error(t, err, invalidRepoURL)
}

func TestCreateProjectWrongSSHKeyPath(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	err := execCmdNotNilAssertion(t, append(baseArgs, "--ssh-key", "dummy_key", "--repo-url", "git@github.com:dummyRepo/dummyProject.git")...)

	expectedMessages := []string{
		"open dummy_key: The system cannot find the file specified.",
		"open dummy_key: no such file or directory",
	}

	assert.Assert(t, util.Contains(expectedMessages, err.Error()))
}

func TestCreateProjectWithSSHKey(t *testing.T) {
	baseArgs := []string{"project", "create", "--project-name", "MOCK"}

	execCmdNilAssertion(t, append(baseArgs, "--ssh-key", "data/Dockerfile", "--repo-url", "git@github.com:dummyRepo/dummyProject.git")...)
}
