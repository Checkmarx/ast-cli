// +build integration

package integration

import (
	"fmt"
	"github.com/google/uuid"
	"testing"

	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
	"gotest.tools/assert"
)

// End to end test of project handling.
// 1. Create a project
// 2. Get and assert the project exists
// 3. Delete the created project
// 4. Get and assert the project was deleted
func TestProjectsE2E(t *testing.T) {
	projectID := createProject(t)

	response := getProjectsByID(t, projectID)

	assert.Equal(t, len(response), 1, "Total projects should be 1")
	assert.Equal(t, response[0].ID, projectID, "Project ID should match the created project")

	deleteProject(t, projectID)

	response = getProjectsByID(t, projectID)

	assert.Equal(t, len(response), 0, "Total projects should be 0 as the project was deleted")
}

func createProject(t *testing.T) string {
	projectName := fmt.Sprintf("integration_test_%s", uuid.New().String())
	createProjCommand, outBuffer := createRedirectedTestCommand(t)

	err := execute(createProjCommand, "project", "create", "--format", "json", "--project-name", projectName)
	assert.NilError(t, err, "Creating a project should pass")

	createdProject := projectsRESTApi.ProjectResponseModel{}
	createdProjectJSON := unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("CREATED PROJECT PAYLOAD IS ", string(createdProjectJSON))
	fmt.Printf("Project ID %s created\n", createdProject.ID)

	return createdProject.ID
}

func deleteProject(t *testing.T, projectID string) {
	deleteProjCommand := createASTIntegrationTestCommand(t)
	err := execute(deleteProjCommand, "project", "delete", "--project-id", projectID)
	assert.NilError(t, err, "Deleting a project should pass")
}

func getProjectsByID(t *testing.T, projectID string) []projectsRESTApi.ProjectResponseModel {
	idFilter := fmt.Sprintf("ids=%s", projectID)
	getAllCommand, outputBuffer := createRedirectedTestCommand(t)

	err := execute(getAllCommand, "project", "list", "--format", "json", "--filter", idFilter)
	assert.NilError(t, err, "Getting the project should pass")

	var projects []projectsRESTApi.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")

	return projects
}
