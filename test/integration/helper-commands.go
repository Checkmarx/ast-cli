//go:build integration

package integration

import (
	"fmt"
	"log"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func GenerateProjectName() string {
	return fmt.Sprintf("ast-cli-tests_%s", uuid.New().String())
}

func DeleteProject(t *testing.T, projectID string) {
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

func DeleteProjectByName(t *testing.T, projectName string) {
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(viper.GetString(params.ProjectsPathKey))
	projectModel, _, err := projectsWrapper.GetByName(projectName)
	if err == nil && projectModel != nil {
		DeleteProject(t, projectModel.ID)
	}
}

func CreateProject(t *testing.T, tags map[string]string, groups []string) (string, string) {
	projectName := getProjectNameForTest() + "_for_project"
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
