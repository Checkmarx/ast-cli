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

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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
	projectID, _ := createProject(t, Tags, Groups)

	response := listProjectByID(t, projectID)

	assert.Equal(t, len(response), 1, "Total projects should be 1")
	assert.Equal(t, response[0].ID, projectID, "Project ID should match the created project")

	project := showProject(t, projectID)
	assert.Equal(t, project.ID, projectID, "Project ID should match the created project")

	assertTagsAndGroups(t, project, Groups)

	deleteProject(t, projectID)

	response = listProjectByID(t, projectID)

	assert.Equal(t, len(response), 0, "Total projects should be 0 as the project was deleted")
}

func TestGetProjectByTagsFilter_whenProjectHasNoneTags_shouldReturnProjectWithNoTags(t *testing.T) {
	projectID, _ := createProject(t, nil, nil)
	defer deleteProject(t, projectID)

	projects := listProjectByTagsAndLimit(t, "NONE", "NONE", "5")
	fmt.Println("Projects length: ", len(projects))
	for _, project := range projects {
		assert.Equal(t, len(project.Tags), 0, "Project should have no tags")
	}
}

// Assert project contains created tags and groups
func assertTagsAndGroups(t *testing.T, project wrappers.ProjectResponseModel, groups []string) {

	allTags := getAllTags(t, "project")

	for key := range Tags {
		_, ok := allTags[key]
		assert.Assert(t, ok, "Get all tags response should contain all created tags. Missing %s", key)

		val, ok := project.Tags[key]
		assert.Assert(t, ok, "Project should contain all created tags. Missing %s", key)
		assert.Equal(t, val, Tags[key], "Tag value should be equal")
		fmt.Println("The project.Groups-->", project.Groups)
		fmt.Println("The Groups assigned are --->", groups)
	}
	// todo: current used grps are created by another users, as ACCESSMGMT FF is on, grps will not be assigned
	assert.Assert(t, len(project.Groups) >= len(groups), "The project must contain at least %d groups", len(groups))
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

func TestProjectCreate_ApplicationDoesntExist_FailAndReturnErrorMessage(t *testing.T) {

	err, _ := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), projectNameRandom,
		flag(params.ApplicationName), "application-that-doesnt-exist",
	)

	assertError(t, err, errorConstants.ApplicationDoesntExistOrNoPermission)
}

func TestProjectCreate_ApplicationExists_CreateProjectSuccessfully(t *testing.T) {

	err, outBuffer := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), projectNameRandom,
		flag(params.ApplicationName), "my-application",
	)
	createdProject := wrappers.ProjectResponseModel{}
	unmarshall(t, outBuffer, &createdProject, "Reading project create response JSON should pass")
	defer deleteProject(t, createdProject.ID)
	assert.NilError(t, err)
	assert.Assert(t, createdProject.ID != "", "Project ID should not be empty")
	assert.Assert(t, len(createdProject.ApplicationIds) == 1, "The project must be connected to the application")
}

func TestCreateWithInvalidGroup(t *testing.T) {
	err, _ := executeCommand(
		t, "project", "create", flag(params.FormatFlag),
		printer.FormatJSON, flag(params.ProjectName), "project", flag(params.GroupList), "invalidGroup",
	)
	assertError(t, err, "Failed finding groups: [invalidGroup]")
}

func TestProjectCreate_WhenCreatingProjectWithExistingName_FailProjectCreation(t *testing.T) {
	err, _ := executeCommand(
		t, "project", "create",
		flag(params.FormatFlag),
		printer.FormatJSON,
		flag(params.ProjectName), "project",
	)
	assertError(t, err, "Failed creating a project: CODE: 208, Failed to create a project, project name 'project' already exists\n")
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

func createProject(t *testing.T, tags map[string]string, groups []string) (string, string) {
	projectName := getProjectNameForTest() + "_for_project"
	return createNewProject(t, tags, groups, projectName)
}

func createNewProject(t *testing.T, tags map[string]string, groups []string, projectName string) (string, string) {
	tagsStr := formatTags(tags)
	groupsStr := formatGroups(groups)

	fmt.Printf("Creating project : %s \n", projectName)
	outBuffer := executeCmdNilAssertion(
		t, "Creating a project should pass",
		"project", "create",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), "master",
		flag(params.TagList), tagsStr,
		flag(params.GroupList), groupsStr,
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

func deleteProjectByName(t *testing.T, projectName string) {
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(viper.GetString(params.ProjectsPathKey))
	projectModel, _, err := projectsWrapper.GetByName(projectName)
	if err == nil && projectModel != nil {
		deleteProject(t, projectModel.ID)
	}
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
	fmt.Println("Listing project for id output buffer-->", outputBuffer)
	var projects []wrappers.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")
	fmt.Println("Listing project for id projects length: ", len(projects))

	return projects
}

func listProjectByTagsAndLimit(t *testing.T, tagsKeys string, tagsValues string, limit string) []wrappers.ProjectResponseModel {
	tagsFilter := fmt.Sprintf("%s=%s,%s=%s", params.TagsKeyQueryParam, tagsKeys, params.TagsValueQueryParam, tagsValues)
	limitFilter := fmt.Sprintf("%s=%s", params.LimitQueryParam, limit)
	fmt.Println("Listing project for filters: ", tagsFilter, ",", limitFilter)
	filters := tagsFilter + "," + limitFilter
	outputBuffer := executeCmdNilAssertion(
		t,
		"Getting the project should pass",
		"project", "list",
		flag(params.FormatFlag), printer.FormatJSON, flag(params.FilterFlag), filters,
	)
	var projects []wrappers.ProjectResponseModel
	_ = unmarshall(t, outputBuffer, &projects, "Reading all projects response JSON should pass")
	fmt.Println("Listing project for tags projects length: ", len(projects))

	return projects
}

func showProject(t *testing.T, projectID string) wrappers.ProjectResponseModel {
	assertRequiredParameter(t, "Failed getting a project: Please provide a project ID", "project", "show")

	outputBuffer := executeCmdNilAssertion(
		t, "Getting the project should pass", "project", "show",
		flag(params.FormatFlag), printer.FormatJSON,
		flag(params.ProjectIDFlag), projectID,
	)
	fmt.Println("Got output Buffered..")
	fmt.Println("Output Buffered from show project is --> ", outputBuffer)

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
