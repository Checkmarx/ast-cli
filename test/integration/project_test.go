//go:build integration

package integration

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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
func assertTags(t *testing.T, project wrappers.ProjectResponseModel) {

	allTags := getAllTags(t, "project")

	for key := range Tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := project.Tags[key]
		assert.Assert(t, ok, "Project should contain all created tags. Missing %s", key)
		assert.Equal(t, val, Tags[key], "Tag value should be equal")
	}
}

// Create a project with empty project name should fail
func TestCreateEmptyProjectName(t *testing.T) {

	err, _ := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), "",
	)
	assertError(t, err, "Project name is required")
}

// Create the same project twice and assert that it fails
func TestCreateAlreadyExistingProject(t *testing.T) {
	assertRequiredParameter(t, "Project name is required", "project", "create")

	_, projectName := getRootProject(t)

	err, _ := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), projectName,
	)
	assertError(t, err, "Failed creating a project: CODE: 208, Failed to create a project, project name")
}

func TestCreateWithInvalidGroup(t *testing.T) {
	err, _ := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), "project", flag(params.GroupList), "invalidGroup",
	)
	assertError(t, err, "Failed finding groups: [invalidGroup]")
}

// Test list project's branches
func TestProjectBranches(t *testing.T) {

	assertRequiredParameter(
		t,
		"Failed getting branches for project: Please provide a project ID",
		"project",
		"branches",
	)

	projectID, _ := getRootProject(t)

	buffer := executeCmdNilAssertion(t, "Branches should be listed", "project", "branches", "--project-id", projectID)

	result, readingError := io.ReadAll(buffer)
	assert.NilError(t, readingError, "Reading result should pass")
	assert.Assert(t, strings.Contains(string(result), "[]"))
}

func createProject(t *testing.T, tags map[string]string) (string, string) {
	projectName := getProjectNameForTest() + "_for_project"
	tagsStr := formatTags(tags)

	fmt.Printf("Creating project : %s \n", projectName)
	outBuffer := executeCmdNilAssertion(
		t, "Creating a project should pass",
		"project", "create",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
	)

	createdProject := wrappers.ProjectResponseModel{}
	createdProjectJSON := unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("Response after project is created : ", string(createdProjectJSON))
	fmt.Printf("New project created with id: %s \n", createdProject.ID)

	return createdProject.ID, projectName
}

func deleteProject(t *testing.T, projectID string) {
	log.Println("Deleting the project with id ", projectID)
	fmt.Println("Deleting the project with id ", projectID)
	executeCmdNilAssertion(
		t,
		"Deleting a project should pass",
		"project",
		"delete",
		flag(params.ProjectIDFlag),
		projectID,
	)
}

func listProjectByID(t *testing.T, projectID string) []wrappers.ProjectResponseModel {
	idFilter := fmt.Sprintf("ids=%s", projectID)
	fmt.Println("Listing project for id ", projectID)
	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting the project should pass",
		"project", "list",
		flag(params.FormatFlag), printer.FormatJSON, flag(params.FilterFlag), idFilter,
	)
	fmt.Println("Listing project for id output buffer", outputBuffer)
	var projects []wrappers.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")
	fmt.Println("Listing project for id projects length: ", len(projects))

	return projects
}

func showProject(t *testing.T, projectID string) wrappers.ProjectResponseModel {
	assertRequiredParameter(t, "Failed getting a project: Please provide a project ID", "project", "show")

	outputBuffer := executeCmdNilAssertion(
		t, "Getting the project should pass", "project", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
	)

	var project wrappers.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &project, "Reading project JSON should pass")

	return project
}

func TestCreateProjectWithSSHKey(t *testing.T) {
	projectName := fmt.Sprintf("ast-cli-tests_%s", uuid.New().String()) + "_for_project"
	tagsStr := formatTags(Tags)

	_ = viper.BindEnv("CX_SCAN_SSH_KEY")
	sshKey := viper.GetString("CX_SCAN_SSH_KEY")

	_ = ioutil.WriteFile(SSHKeyFilePath, []byte(sshKey), 0644)
	defer func() { _ = os.Remove(SSHKeyFilePath) }()

	cmd := createASTIntegrationTestCommand(t)
	err := execute(
		cmd,
		"project", "create",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
		flag(params.RepoURLFlag), SSHRepo,
		flag(params.SSHKeyFlag), "",
	)
	assert.Assert(t, err != nil)

	err = execute(
		cmd,
		"project", "create",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
		flag(params.RepoURLFlag), "",
		flag(params.SSHKeyFlag), SSHKeyFilePath,
	)
	assert.Assert(t, err != nil)

	fmt.Printf("Creating project : %s \n", projectName)
	outBuffer := executeCmdNilAssertion(
		t, "Creating a project with ssh key should pass",
		"project", "create",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
		flag(params.RepoURLFlag), SSHRepo,
		flag(params.SSHKeyFlag), SSHKeyFilePath,
	)

	createdProject := wrappers.ProjectResponseModel{}
	createdProjectJSON := unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")

	fmt.Println("Response after project is created : ", string(createdProjectJSON))
	fmt.Printf("New project created with id: %s \n", createdProject.ID)

	deleteProject(t, createdProject.ID)
}
