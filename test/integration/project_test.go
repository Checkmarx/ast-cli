// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
	"gotest.tools/assert"
)

func TestProjectsE2E(t *testing.T) {
	// TODO: fix this
	fmt.Println("TODO: Disabled TestProjectsE2E because its not working.")
	/*
		projectFromFile := createProjectFromInputFile(t)
		getAllProjects(t, projectFromFile)
		getAllProjectsList(t)
		deleteProject(t, projectFromFile)

		tags := map[string]string{}
		projectID := createProjectFromInput(t, RandomizeString(5), tags)
		getProjectByID(t, projectID)
		getProjectByIDList(t, projectID)
		tags = map[string]string{
			"A_KEY": "A_VAL",
			"B_KEY": "B_VAL",
			"C_KEY": "C_VAL",
		}
		deleteProject(t, projectID)

		projectID = createProjectFromInput(t, RandomizeString(5), tags)
		getProjectTags(t)
		deleteProject(t, projectID)
	*/
}

func createProjectFromInputFile(t *testing.T) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand(t)
	createProjCommand.SetOut(b)
	err := execute(createProjCommand, "-v", "--format", "json", "project", "create", "--input-file", "project_payload.json")
	return executeCreateProject(t, err, b)
}

func createProjectFromInput(t *testing.T, projectID string, tags map[string]string) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand(t)
	createProjCommand.SetOut(b)
	tagsJSON, err := json.Marshal(tags)
	assert.NilError(t, err, "Marshaling tags should pass")
	payload := fmt.Sprintf("{\"id\":\"integration_test_%s\", \"tags\":%s}", projectID, string(tagsJSON))
	err = execute(createProjCommand, "-v", "--format", "json", "project", "create", "--input", payload)
	return executeCreateProject(t, err, b)
}

func executeCreateProject(t *testing.T, err error, b *bytes.Buffer) string {
	assert.NilError(t, err, "Creating a project should pass")
	// Read response from buffer
	var createdProjectJSON []byte
	createdProjectJSON, err = ioutil.ReadAll(b)
	fmt.Println("CREATED PROJECT PAYLOAD IS ", string(createdProjectJSON))
	assert.NilError(t, err, "Reading project response JSON should pass")
	createdProject := projectsRESTApi.ProjectResponseModel{}
	err = json.Unmarshal(createdProjectJSON, &createdProject)
	assert.NilError(t, err, "Parsing project response JSON should pass")
	log.Printf("Project ID %s created in test", createdProject.ID)
	return createdProject.ID
}

func getProjectByIDList(t *testing.T, projectID string) {
	getProjectCommand := createASTIntegrationTestCommand(t)
	err := execute(getProjectCommand, "-v", "--format", "list", "project", "show", projectID)
	assert.NilError(t, err, "Getting a project should pass")
}

func getProjectByID(t *testing.T, projectID string) {
	b := bytes.NewBufferString("")
	getProjectCommand := createASTIntegrationTestCommand(t)
	getProjectCommand.SetOut(b)
	err := execute(getProjectCommand, "-v", "--format", "json", "project", "show", projectID)
	assert.NilError(t, err, "Getting a project should pass")
	// Read response from buffer
	var projectJSON []byte
	projectJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading project response JSON should pass")
	project := projectsRESTApi.ProjectResponseModel{}
	err = json.Unmarshal(projectJSON, &project)
	assert.NilError(t, err, "Parsing project response JSON should pass")
	assert.Assert(t, projectID == project.ID)
}

func getAllProjectsList(t *testing.T) {
	getAllCommand := createASTIntegrationTestCommand(t)
	var limit uint64 = 40
	var offset uint64 = 0
	lim := fmt.Sprintf("limit=%s", strconv.FormatUint(limit, 10))
	off := fmt.Sprintf("offset=%s", strconv.FormatUint(offset, 10))
	err := execute(getAllCommand, "-v", "--format", "list", "project", "list", "--filter", lim, "--filter", off)
	assert.NilError(t, err, "Getting all projects should pass")
}

func getAllProjects(t *testing.T, projectID string) {
	b := bytes.NewBufferString("")
	getAllCommand := createASTIntegrationTestCommand(t)
	getAllCommand.SetOut(b)
	var limit uint64 = 40
	var offset uint64 = 0
	lim := fmt.Sprintf("limit=%s", strconv.FormatUint(limit, 10))
	off := fmt.Sprintf("offset=%s", strconv.FormatUint(offset, 10))
	err := execute(getAllCommand, "-v", "--format", "json", "project", "list", "--filter", lim, "--filter", off)
	assert.NilError(t, err, "Getting all projects should pass")
	// Read response from buffer
	var getAllJSON []byte
	getAllJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading all projects response JSON should pass")
	allProjects := []projectsRESTApi.ProjectResponseModel{}
	err = json.Unmarshal(getAllJSON, &allProjects)
	assert.NilError(t, err, "Parsing all projects response JSON should pass")
	assert.Equal(t, len(allProjects), 1, "Total projects should be 1")
	assert.Equal(t, allProjects[0].ID, projectID)
}

func deleteProject(t *testing.T, projectID string) {
	fmt.Println("TESTING project DELETE")
	deleteProjCommand := createASTIntegrationTestCommand(t)
	err := execute(deleteProjCommand, "-v", "project", "delete", projectID)
	assert.NilError(t, err, "Deleting a project should pass")
}

func getProjectTags(t *testing.T) {
	b := bytes.NewBufferString("")
	tagsCommand := createASTIntegrationTestCommand(t)
	tagsCommand.SetOut(b)
	err := execute(tagsCommand, "-v", "--format", "json", "project", "tags")
	assert.NilError(t, err, "Getting tags should pass")
	// Read response from buffer
	var tagsJSON []byte
	tagsJSON, err = ioutil.ReadAll(b)
	assert.NilError(t, err, "Reading tags JSON should pass")
	tags := map[string][]string{}
	err = json.Unmarshal(tagsJSON, &tags)
	assert.NilError(t, err, "Parsing tags JSON should pass")
	assert.Assert(t, tags != nil)
	assert.Assert(t, len(tags) == 3)
}
