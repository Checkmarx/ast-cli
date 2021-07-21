// +build integration

package integration

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	"github.com/google/uuid"
	"testing"

	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
	"gotest.tools/assert"
)

// End to end test of project handling.
// - Create a project
// - Get and assert the project exists
// - Get all tags
// - Assert assigned tags exist
// - Assert project contains the tags
// - Assert project contains the groups
// - Delete the created project
// - Get and assert the project was deleted
func TestProjectsE2E(t *testing.T) {

	projectID := createProject(t, Tags, Groups)

	response := listProjectByID(t, projectID)

	assert.Equal(t, len(response), 1, "Total projects should be 1")
	assert.Equal(t, response[0].ID, projectID, "Project ID should match the created project")

	project := showProject(t, projectID)
	assert.Equal(t, project.ID, projectID, "Project ID should match the created project")

	allTags := getAllTags(t, "project")
	for key := range Tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := project.Tags[key]
		assert.Assert(t, ok, "Project should contain all created tags. Missing %s", key)
		assert.Equal(t, val, Tags[key], "Tag value should be equal")
	}

	for _, group := range Groups {
		assert.Assert(t, contains(project.Groups, group), "Project should contain group %s", group)
	}

	deleteProject(t, projectID)

	response = listProjectByID(t, projectID)

	assert.Equal(t, len(response), 0, "Total projects should be 0 as the project was deleted")
}

// Create the same project twice and assert that it fails
func TestCreateAlreadyExisting(t *testing.T) {
	projectID := createProject(t, map[string]string{}, []string{})

	defer deleteProject(t, projectID)

	err := execute(createASTIntegrationTestCommand(t),
		"project", "create",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.ProjectName), showProject(t, projectID).Name,
	)
	assert.Assert(t, err != nil, "Creating a project with the same name should fail")
}

func createProject(t *testing.T, tags map[string]string, groups []string) string {
	projectName := fmt.Sprintf("integration_test_project_%s", uuid.New().String())
	createProjCommand, outBuffer := createRedirectedTestCommand(t)

	tagsStr := formatTags(tags)
	groupsStr := formatGroups(groups)

	err := execute(createProjCommand,
		"project", "create",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.ProjectName), projectName,
		flag(commands.TagList), tagsStr,
		flag(commands.GroupList), groupsStr,
	)
	assert.NilError(t, err, "Creating a project should pass")

	createdProject := projectsRESTApi.ProjectResponseModel{}
	createdProjectJSON := unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("CREATED PROJECT PAYLOAD IS ", string(createdProjectJSON))
	fmt.Printf("Project ID %s created\n", createdProject.ID)

	return createdProject.ID
}

func deleteProject(t *testing.T, projectID string) {
	deleteProjCommand := createASTIntegrationTestCommand(t)
	err := execute(deleteProjCommand,
		"project", "delete",
		flag(commands.ProjectIDFlag), projectID,
	)
	assert.NilError(t, err, "Deleting a project should pass")
}

func listProjectByID(t *testing.T, projectID string) []projectsRESTApi.ProjectResponseModel {
	idFilter := fmt.Sprintf("ids=%s", projectID)
	getAllCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(getAllCommand,
		"project", "list",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.FilterFlag), idFilter,
	)
	assert.NilError(t, err, "Getting the project should pass")

	var projects []projectsRESTApi.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")

	return projects
}

func showProject(t *testing.T, projectID string) projectsRESTApi.ProjectResponseModel {
	showCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(showCommand,
		"project", "show",
		flag(commands.FormatFlag), util.FormatJSON,
		flag(commands.ProjectIDFlag), projectID,
	)
	assert.NilError(t, err, "Getting the project should pass")

	var project projectsRESTApi.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &project, "Reading project JSON should pass")

	return project
}
