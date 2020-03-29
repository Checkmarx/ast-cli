// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	projectsRESTApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
	"gotest.tools/assert"
)

func TestE2E(t *testing.T) {
	projectFromFile := createProjectFromInputFile(t)
	projectID := createProjectFromInput(t, RandomizeString(5), []string{})
	deleteProject(t, projectID)
	getAllProjects(t, projectFromFile)
	getProjectByID(t, projectFromFile)
	_ = createProjectFromInput(t, RandomizeString(5), []string{"A", "B", "D"})
	getTags(t)
	// TODO Get tags and check for [A, B, C, D]
	deleteProject(t, projectFromFile)
}
func getTags(t *testing.T) {
	b := bytes.NewBufferString("")
	tagsCommand := createASTIntegrationTestCommand()
	tagsCommand.SetOut(b)
	err := execute(tagsCommand, "-v", "project", "tags")
	assert.NilError(t, err, "Getting tags should pass")
	// Read response from buffer
	var tagsJSON []byte
	tagsJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading tags JSON should pass")
	tags := []string{}
	err = json.Unmarshal(tagsJSON, &tags)
	assert.NilError(t, err, "Parsing tags JSON should pass")
	assert.Assert(t, tags != nil)
	assert.Assert(t, len(tags) == 4)
}

func getProjectByID(t *testing.T, projectID string) {
	b := bytes.NewBufferString("")
	getProjectCommand := createASTIntegrationTestCommand()
	getProjectCommand.SetOut(b)
	err := execute(getProjectCommand, "-v", "project", "get", projectID)
	assert.NilError(t, err, "Getting a project should pass")
	// Read response from buffer
	var projectJSON []byte
	projectJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading project response JSON should pass")
	project := projectsRESTApi.ProjectResponseModel{}
	err = json.Unmarshal(projectJSON, &project)
	assert.NilError(t, err, "Parsing project response JSON should pass")
	assert.Assert(t, projectID == project.ID)
	assert.Assert(t, project.Tags != nil)
	assert.Assert(t, len(project.Tags) == 3)
	assert.Assert(t, project.Tags[0] == "A")
	assert.Assert(t, project.Tags[1] == "B")
	assert.Assert(t, project.Tags[2] == "C")
}

func createProjectFromInputFile(t *testing.T) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand()
	createProjCommand.SetOut(b)
	err := execute(createProjCommand, "-v", "project", "create", "--inputFile", "project_payload.json")
	return executeCreateProject(t, err, b)
}

func createProjectFromInput(t *testing.T, projectID string, tags []string) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand()
	createProjCommand.SetOut(b)
	tagsJSON, err := json.Marshal(tags)
	assert.NilError(t, err, "Marshaling tags should pass")
	payload := fmt.Sprintf("{\"id\":\"integration_test_%s\", \"tags\":%s}", projectID, string(tagsJSON))
	err = execute(createProjCommand, "-v", "project", "create", "--input", payload)
	return executeCreateProject(t, err, b)
}

func executeCreateProject(t *testing.T, err error, b *bytes.Buffer) string {
	assert.NilError(t, err, "Creating a project should pass")
	// Read response from buffer
	var createdProjectJSON []byte
	createdProjectJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading project response JSON should pass")
	createdProject := projectsRESTApi.ProjectResponseModel{}
	err = json.Unmarshal(createdProjectJSON, &createdProject)
	assert.NilError(t, err, "Parsing project response JSON should pass")
	log.Printf("Project ID %s created in test", createdProject.ID)
	return createdProject.ID
}

func deleteProject(t *testing.T, projectID string) {
	deleteProjCommand := createASTIntegrationTestCommand()
	err := execute(deleteProjCommand, "-v", "project", "delete", projectID)
	assert.NilError(t, err, "Deleting a project should pass")
}

func getAllProjects(t *testing.T, projectID string) {
	b := bytes.NewBufferString("")
	getAllCommand := createASTIntegrationTestCommand()
	getAllCommand.SetOut(b)
	err := execute(getAllCommand, "-v", "project", "get-all")
	assert.NilError(t, err, "Getting all projects should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all projects response JSON should pass")
	allProjects := projectsRESTApi.SlicedProjectsResponseModel{}
	err = json.Unmarshal(getAllJSON, &allProjects)
	assert.NilError(t, err, "Parsing all projects response JSON should pass")
	assert.Assert(t, allProjects.Limit == 20, "Limit should be 20")
	assert.Assert(t, allProjects.Offset == 0, "Offset should be 0")
	assert.Assert(t, allProjects.TotalCount == 1, "Total should be 1")
	assert.Assert(t, len(allProjects.Projects) == 1, "Total should be 1")
	assert.Assert(t, allProjects.Projects[0].ID == projectID)
}
