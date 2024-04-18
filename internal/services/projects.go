package services

import (
	"strconv"
	"strings"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
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

func AssignGroupsToProject(projectID string, projectName string, groups []*wrappers.Group,
	accessManagement wrappers.AccessManagementWrapper) error {
	if !wrappers.FeatureFlags[featureFlagsConstants.AccessManagementEnabled] {
		return nil
	}
	groupsAssignedToTheProject, err := accessManagement.GetGroups(projectID)
	if err != nil {
		return err
	}
	groupsToAssign := getGroupsToAssign(groups, groupsAssignedToTheProject)
	if len(groupsToAssign) == 0 {
		return nil
	}

	err = accessManagement.CreateGroupsAssignment(projectID, projectName, groupsToAssign)
	if err != nil {
		return err
	}
	return nil
}

func CreateGroupsMap(groupsStr string, groupsWrapper wrappers.GroupsWrapper) ([]*wrappers.Group, error) {
	groups := strings.Split(groupsStr, ",")
	var groupsMap []*wrappers.Group
	var groupsNotFound []string
	for _, group := range groups {
		if len(group) > 0 {
			groupsFromEnv, err := groupsWrapper.Get(group)
			if err != nil {
				groupsNotFound = append(groupsNotFound, group)
			} else {
				findGroup := findGroupByName(groupsFromEnv, group)
				if findGroup != nil && findGroup.Name != "" {
					groupsMap = append(groupsMap, findGroup)
				} else {
					groupsNotFound = append(groupsNotFound, group)
				}
			}
		}
	}
	if len(groupsNotFound) > 0 {
		return nil, errors.Errorf("%s: %v", failedFindingGroup, groupsNotFound)
	}
	return groupsMap, nil
}

func getGroupsForRequest(groups []*wrappers.Group) []string {
	if !wrappers.FeatureFlags[featureFlagsConstants.AccessManagementEnabled] {
		return GetGroupIds(groups)
	}
	return nil
}

func createTagMap(tagListStr string) map[string]string {
	tagsList := strings.Split(tagListStr, ",")
	tags := make(map[string]string)
	for _, tag := range tagsList {
		if len(tag) > 0 {
			value := ""
			keyValuePair := strings.Split(tag, ":")
			if len(keyValuePair) > 1 {
				value = keyValuePair[1]
			}
			tags[keyValuePair[0]] = value
		}
	}
	return tags
}

func createApplicationIds(applicationID, existingApplicationIds []string) []string {
	for _, id := range applicationID {
		if !utils.Contains(existingApplicationIds, id) {
			existingApplicationIds = append(existingApplicationIds, id)
		}
	}
	return existingApplicationIds
}

func getGroupsToAssign(receivedGroups, existingGroups []*wrappers.Group) []*wrappers.Group {
	var groupsToAssign []*wrappers.Group
	var groupsMap = make(map[string]bool)
	for _, existingGroup := range existingGroups {
		groupsMap[existingGroup.ID] = true
	}
	for _, receivedGroup := range receivedGroups {
		find := groupsMap[receivedGroup.ID]
		if !find {
			groupsToAssign = append(groupsToAssign, receivedGroup)
		}
	}
	return groupsToAssign
}

func GetGroupIds(groups []*wrappers.Group) []string {
	var groupIds []string
	for _, group := range groups {
		groupIds = append(groupIds, group.ID)
	}
	return groupIds
}

func findGroupByName(groups []wrappers.Group, name string) *wrappers.Group {
	for i := 0; i < len(groups); i++ {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
}
