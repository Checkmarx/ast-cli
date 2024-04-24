package services

import (
	"strconv"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const ErrorCodeFormat = "%s: CODE: %d, %s\n"
const FailedCreatingProj = "Failed creating a project"
const FailedGettingProj = "Failed getting a project"
const failedUpdatingProj = "Failed updating a project"
const failedFindingGroup = "Failed finding groups"

func FindProject(
	applicationID []string,
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
) (string, error) {
	params := make(map[string]string)
	params["names"] = projectName
	resp, _, err := projectsWrapper.Get(params)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			return updateProject(resp, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, projectName, applicationID)
		}
	}
	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationID)
	if err != nil {
		return "", err
	}
	return projectID, nil
}

func createProject(
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationID []string,
) (string, error) {
	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	groupsMap, err := CreateGroupsMap(projectGroups, groupsWrapper)
	if err != nil {
		return "", err
	}
	var projModel = wrappers.Project{}
	projModel.Name = projectName
	projModel.Groups = getGroupsForRequest(groupsMap)
	projModel.ApplicationIds = applicationID

	if projectPrivatePackage != "" {
		projModel.PrivatePackage, _ = strconv.ParseBool(projectPrivatePackage)
	}
	projModel.Tags = createTagMap(projectTags)
	resp, errorModel, err := projectsWrapper.Create(&projModel)
	projectID := ""
	if errorModel != nil {
		err = errors.Errorf(ErrorCodeFormat, FailedCreatingProj, errorModel.Code, errorModel.Message)
	}
	if err == nil {
		projectID = resp.ID
		err = AssignGroupsToProject(projectID, projectName, groupsMap, accessManagementWrapper)
	}
	return projectID, err
}

//nolint:gocyclo
func updateProject(
	resp *wrappers.ProjectsCollectionResponseModel,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	projectName string,
	applicationID []string,

) (string, error) {
	var projectID string
	var projModel = wrappers.Project{}
	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			projectID = resp.Projects[i].ID
		}
		if resp.Projects[i].MainBranch != "" {
			projModel.MainBranch = resp.Projects[i].MainBranch
		}
		if resp.Projects[i].RepoURL != "" {
			projModel.RepoURL = resp.Projects[i].RepoURL
		}
	}
	if projectGroups == "" && projectTags == "" && projectPrivatePackage == "" && len(applicationID) == 0 {
		logger.PrintIfVerbose("No groups, applicationId or tags to update. Skipping project update.")
		return projectID, nil
	}
	if projectPrivatePackage != "" {
		projModel.PrivatePackage, _ = strconv.ParseBool(projectPrivatePackage)
	}
	projModelResp, errModel, err := projectsWrapper.GetByID(projectID)
	if errModel != nil {
		err = errors.Errorf(ErrorCodeFormat, FailedGettingProj, errModel.Code, errModel.Message)
	}
	if err != nil {
		return "", err
	}
	projModel.Name = projModelResp.Name
	projModel.Groups = projModelResp.Groups
	projModel.Tags = projModelResp.Tags
	projModel.ApplicationIds = projModelResp.ApplicationIds
	if projectGroups != "" {
		groupsMap, groupErr := CreateGroupsMap(projectGroups, groupsWrapper)
		if groupErr != nil {
			return "", errors.Errorf("%s: %v", failedUpdatingProj, groupErr)
		}
		logger.PrintIfVerbose("Updating project groups")
		projModel.Groups = getGroupsForRequest(groupsMap)
		err = AssignGroupsToProject(projectID, projectName, groupsMap, accessManagementWrapper)
		if err != nil {
			return "", err
		}
	}
	if projectTags != "" {
		logger.PrintIfVerbose("Updating project tags")
		projModel.Tags = createTagMap(projectTags)
	}
	if len(applicationID) > 0 {
		logger.PrintIfVerbose("Updating project applicationIds")
		projModel.ApplicationIds = createApplicationIds(applicationID, projModelResp.ApplicationIds)
	}
	err = projectsWrapper.Update(projectID, &projModel)
	if err != nil {
		return "", errors.Errorf("%s: %v", failedUpdatingProj, err)
	}
	return projectID, nil
}
