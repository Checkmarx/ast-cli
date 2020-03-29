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

func TestCreateProject(t *testing.T) {
	_ = createProjectFromInputFile(t)
	_ = createProjectFromInput(t, RandomizeString(5))
}

func TestDeleteProject(t *testing.T) {
	projectID := createProjectFromInput(t, RandomizeString(5))
	deleteProject(t, projectID)
}

func createProjectFromInputFile(t *testing.T) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand()
	createProjCommand.SetOut(b)
	err := execute(createProjCommand, "-v", "project", "create", "--inputFile", "project_payload.json")
	return executeCreateProject(t, err, b)
}

func createProjectFromInput(t *testing.T, projectID string) string {
	b := bytes.NewBufferString("")
	createProjCommand := createASTIntegrationTestCommand()
	createProjCommand.SetOut(b)
	payload := fmt.Sprintf("{\"id\":\"integration_test_%s\", \"tags\":[]}", projectID)
	err := execute(createProjCommand, "-v", "project", "create", "--input", payload)
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
