//go:build projectstest

package projectstest

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/test/integration"
	"github.com/google/uuid"
	"github.com/spf13/viper"

	"gotest.tools/assert"
)

const SSHKeyFilePath = "ssh-key-file.txt"

// End-to-end test of project handling.
// - Create a project
// - Get and assert the project exists
// - Get all tags
// - Assert assigned tags exist
// - Assert project contains the tags
// - Assert project contains the groups
// - Delete the created project
// - Get and assert the project was deleted
func TestProjectsE2E(t *testing.T) {
	projectID, _ := integration.CreateProject(t, integration.Tags, integration.Groups)

	response := listProjectByID(t, projectID)

	assert.Equal(t, len(response), 1, "Total projects should be 1")
	assert.Equal(t, response[0].ID, projectID, "Project ID should match the created project")

	project := showProject(t, projectID)
	assert.Equal(t, project.ID, projectID, "Project ID should match the created project")

	assertTagsAndGroups(t, project, integration.Groups)

	integration.DeleteProject(t, projectID)

	response = listProjectByID(t, projectID)

	assert.Equal(t, len(response), 0, "Total projects should be 0 as the project was deleted")
}

// Assert project contains created tags and groups
func assertTagsAndGroups(t *testing.T, project wrappers.ProjectResponseModel, groups []string) {

	allTags := integration.getAllTags(t, "project")

	for key := range integration.Tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := project.Tags[key]
		assert.Assert(t, ok, "Project should contain all created tags. Missing %s", key)
		assert.Equal(t, val, integration.Tags[key], "Tag value should be equal")
	}

	assert.Assert(t, len(project.Groups) >= len(groups), "The project must contain at least %d groups", len(groups))
}

// Create a project with empty project name should fail
func TestCreateEmptyProjectName(t *testing.T) {

	err, _ := integration.executeCommand(
		t, "project", "create", integration.flag(params.FormatFlag),
		printer.FormatJSON, integration.flag(params.ProjectName), "",
	)
	integration.assertError(t, err, "Project name is required")
}

// Create the same project twice and assert that it fails
func TestCreateAlreadyExistingProject(t *testing.T) {
	integration.assertRequiredParameter(t, "Project name is required", "project", "create")

	_, projectName := integration.getRootProject(t)

	err, _ := integration.executeCommand(
		t, "project", "create", integration.flag(params.FormatFlag),
		printer.FormatJSON, integration.flag(params.ProjectName), projectName,
	)
	integration.assertError(t, err, "Failed creating a project: CODE: 208, Failed to create a project, project name")
}

func TestProjectCreate_ApplicationDoesntExist_FailAndReturnErrorMessage(t *testing.T) {

	err, _ := integration.executeCommand(
		t, "project", "create", integration.flag(params.FormatFlag),
		printer.FormatJSON, integration.flag(params.ProjectName), integration.GenerateProjectName(),
		integration.flag(params.ApplicationName), "application-that-doesnt-exist",
	)

	integration.assertError(t, err, errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestProjectCreate_ApplicationExists_CreateProjectSuccessfully(t *testing.T) {

	err, outBuffer := integration.executeCommand(
		t, "project", "create", integration.flag(params.FormatFlag),
		printer.FormatJSON, integration.flag(params.ProjectName), integration.GenerateProjectName(),
		integration.flag(params.ApplicationName), "my-application",
	)
	createdProject := wrappers.ProjectResponseModel{}
	integration.unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")
	defer integration.DeleteProject(t, createdProject.ID)
	assert.NilError(t, err)
	assert.Assert(t, createdProject.ID != "", "Project ID should not be empty")
	assert.Assert(t, len(createdProject.ApplicationIds) == 1, "The project must be connected to the application")
}

func TestCreateWithInvalidGroup(t *testing.T) {
	err, _ := integration.executeCommand(
		t, "project", "create", integration.flag(params.FormatFlag),
		printer.FormatJSON, integration.flag(params.ProjectName), "project", integration.flag(params.GroupList), "invalidGroup",
	)
	integration.assertError(t, err, "Failed finding groups: [invalidGroup]")
}

func TestProjectCreate_WhenCreatingProjectWithExistingName_FailProjectCreation(t *testing.T) {
	err, _ := integration.executeCommand(
		t, "project", "create",
		integration.flag(params.FormatFlag),
		printer.FormatJSON,
		integration.flag(params.ProjectName), "project",
	)
	integration.assertError(t, err, "Failed creating a project: CODE: 208, Failed to create a project, project name 'project' already exists\n")
}

// Test list project's branches
func TestProjectBranches(t *testing.T) {

	integration.assertRequiredParameter(
		t,
		"Failed getting branches for project: Please provide a project ID",
		"project",
		"branches",
	)

	projectID, _ := integration.getRootProject(t)

	buffer := integration.executeCmdNilAssertion(t, "Branches should be listed", "project", "branches", "--project-id", projectID)

	result, readingError := io.ReadAll(buffer)
	assert.NilError(t, readingError, "Reading result should pass")
	assert.Assert(t, strings.Contains(string(result), "[]"))
}

func listProjectByID(t *testing.T, projectID string) []wrappers.ProjectResponseModel {
	idFilter := fmt.Sprintf("ids=%s", projectID)
	fmt.Println("Listing project for id ", projectID)
	outputBuffer := integration.executeCmdNilAssertion(
		t,
		"Getting the project should pass",
		"project", "list",
		integration.flag(params.FormatFlag), printer.FormatJSON, integration.flag(params.FilterFlag), idFilter,
	)
	fmt.Println("Listing project for id output buffer", outputBuffer)
	var projects []wrappers.ProjectResponseModel
	_ = integration.unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")
	fmt.Println("Listing project for id projects length: ", len(projects))

	return projects
}

func showProject(t *testing.T, projectID string) wrappers.ProjectResponseModel {
	integration.assertRequiredParameter(t, "Failed getting a project: Please provide a project ID", "project", "show")

	outputBuffer := integration.executeCmdNilAssertion(
		t, "Getting the project should pass", "project", "show",
		integration.flag(params.FormatFlag), printer.FormatJSON,
		integration.flag(params.ProjectIDFlag), projectID,
	)

	var project wrappers.ProjectResponseModel
	_ = integration.unmarshall(t, outputBuffer, &project, "Reading project JSON should pass")

	return project
}

func TestCreateProjectWithSSHKey(t *testing.T) {
	projectName := fmt.Sprintf("ast-cli-tests_%s", uuid.New().String()) + "_for_project"
	tagsStr := integration.formatTags(integration.Tags)

	_ = viper.BindEnv("CX_SCAN_SSH_KEY")
	sshKey := viper.GetString("CX_SCAN_SSH_KEY")

	_ = ioutil.WriteFile(SSHKeyFilePath, []byte(sshKey), 0644)
	defer func() { _ = os.Remove(SSHKeyFilePath) }()

	cmd := integration.createASTIntegrationTestCommand(t)
	err := integration.execute(
		cmd,
		"project", "create",
		integration.flag(params.FormatFlag), printer.FormatJSON,
		integration.flag(params.ProjectName), projectName,
		integration.flag(params.BranchFlag), "master",
		integration.flag(params.TagList), tagsStr,
		integration.flag(params.RepoURLFlag), integration.SSHRepo,
		integration.flag(params.SSHKeyFlag), "",
	)
	assert.Assert(t, err != nil)

	err = integration.execute(
		cmd,
		"project", "create",
		integration.flag(params.FormatFlag), printer.FormatJSON,
		integration.flag(params.ProjectName), projectName,
		integration.flag(params.BranchFlag), "master",
		integration.flag(params.TagList), tagsStr,
		integration.flag(params.RepoURLFlag), "",
		integration.flag(params.SSHKeyFlag), SSHKeyFilePath,
	)
	assert.Assert(t, err != nil)

	fmt.Printf("Creating project : %s \n", projectName)
	outBuffer := integration.executeCmdNilAssertion(
		t, "Creating a project with ssh key should pass",
		"project", "create",
		integration.flag(params.FormatFlag), printer.FormatJSON,
		integration.flag(params.ProjectName), projectName,
		integration.flag(params.BranchFlag), "master",
		integration.flag(params.TagList), tagsStr,
		integration.flag(params.RepoURLFlag), integration.SSHRepo,
		integration.flag(params.SSHKeyFlag), SSHKeyFilePath,
	)

	createdProject := wrappers.ProjectResponseModel{}
	createdProjectJSON := integration.unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("Response after project is created : ", string(createdProjectJSON))
	fmt.Printf("New project created with id: %s \n", createdProject.ID)

	integration.DeleteProject(t, createdProject.ID)
}
