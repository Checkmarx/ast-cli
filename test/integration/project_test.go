//go:build integration

package integration

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"

	"gotest.tools/assert"
)

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
	projectID, _ := createProject(t, Tags)

	response := listProjectByID(t, projectID)

	assert.Equal(t, len(response), 1, "Total projects should be 1")
	assert.Equal(t, response[0].ID, projectID, "Project ID should match the created project")

	project := showProject(t, projectID)
	assert.Equal(t, project.ID, projectID, "Project ID should match the created project")

	assertTags(t, project)

	deleteProject(t, projectID)

	response = listProjectByID(t, projectID)

	assert.Equal(t, len(response), 0, "Total projects should be 0 as the project was deleted")
}

// Assert project contains created tags and groups
func assertTags(t *testing.T, project ProjectResponseModel) {

	allTags := getAllTags(t, "project")

	for key := range Tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := project.Tags[key]
		assert.Assert(t, ok, "Project should contain all created tags. Missing %s", key)
		assert.Equal(t, val, Tags[key], "Tag value should be equal")
	}
}

// Create the same project twice and assert that it fails
func TestCreateAlreadyExisting(t *testing.T) {

	assertRequiredParameter(t, "Project name is required", "project", "create")

	_, projectName := getRootProject(t)

	err, _ := executeCommand(t, "project", "create", flag(params.FormatFlag), util.FormatJSON, flag(params.ProjectName), projectName)
	assertError(t, err, "Failed creating a project: CODE: 208, Failed to create a project, project name")
}

func TestCreateWithInvalidGroup(t *testing.T) {
	err, _ := executeCommand(t, "project", "create", flag(params.FormatFlag),
		util.FormatJSON, flag(params.ProjectName), "project", flag(params.GroupList), "invalidGroup")
	assertError(t, err, "Failed finding groups: [invalidGroup]")
}

// Test list project's branches
func TestProjectBranches(t *testing.T) {

	assertRequiredParameter(t, "Failed getting branches for project: Please provide a project ID", "project", "branches")

	projectId, _ := getRootProject(t)

	buffer := executeCmdNilAssertion(t, "Branches should be listed", "project", "branches", "--project-id", projectId)

	result, readingError := io.ReadAll(buffer)
	assert.NilError(t, readingError, "Reading result should pass")
	assert.Assert(t, strings.Contains(string(result), "[]"))
}

func createProject(t *testing.T, tags map[string]string) (string, string) {
	projectName := fmt.Sprintf("integration_test_project_%s", uuid.New().String())
	tagsStr := formatTags(tags)

	outBuffer := executeCmdNilAssertion(t, "Creating a project should pass",
		"project", "create",
		flag(params.FormatFlag), util.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
	)

	createdProject := ProjectResponseModel{}
	createdProjectJSON := unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("CREATED PROJECT PAYLOAD IS ", string(createdProjectJSON))
	fmt.Printf("Project ID %s created\n", createdProject.ID)

	return createdProject.ID, projectName
}

func deleteProject(t *testing.T, projectID string) {
	executeCmdNilAssertion(t, "Deleting a project should pass", "project", "delete", flag(params.ProjectIDFlag), projectID)
}

func listProjectByID(t *testing.T, projectID string) []ProjectResponseModel {
	idFilter := fmt.Sprintf("ids=%s", projectID)

	outputBuffer := executeCmdNilAssertion(t,
		"Getting the project should pass",
		"project", "list",
		flag(params.FormatFlag), util.FormatJSON, flag(params.FilterFlag), idFilter,
	)

	var projects []ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")

	return projects
}

func showProject(t *testing.T, projectID string) ProjectResponseModel {
	assertRequiredParameter(t, "Failed getting a project: Please provide a project ID", "project", "show")

	outputBuffer := executeCmdNilAssertion(t, "Getting the project should pass", "project", "show",
		flag(params.FormatFlag), util.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
	)

	var project ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &project, "Reading project JSON should pass")

	return project
}
