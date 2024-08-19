package services

import (
	"slices"
	"strconv"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	ErrorCodeFormat                     = "%s: CODE: %d, %s\n"
	FailedCreatingProj                  = "Failed creating a project"
	FailedGettingProj                   = "Failed getting a project"
	failedUpdatingProj                  = "Failed updating a project"
	failedFindingGroup                  = "Failed finding groups"
	failedProjectApplicationAssociation = "Failed association project to application"
)

func FindProject(
	applicationID []string,
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) (string, error) {
	params := make(map[string]string)
	params["names"] = projectName
	resp, _, err := projectsWrapper.Get(params)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
			projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
			projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
			return updateProject(
				resp,
				cmd,
				projectsWrapper,
				groupsWrapper,
				accessManagementWrapper,
				applicationWrapper,
				projectName,
				applicationID,
				projectGroups,
				projectTags,
				projectPrivatePackage,
				featureFlagsWrapper)
		}
	}

	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationWrapper,
		applicationID, projectGroups, projectPrivatePackage, featureFlagsWrapper)
	if err != nil {
		logger.PrintIfVerbose("error in creating project!")
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
	applicationsWrapper wrappers.ApplicationsWrapper,
	applicationID []string,
	projectGroups string,
	projectPrivatePackage string,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) (string, error) {
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	applicationName, _ := cmd.Flags().GetString(commonParams.ApplicationName)
	var projModel = wrappers.Project{}
	projModel.Name = projectName
	projModel.ApplicationIds = applicationID
	_, groupErr := GetGroupMap(groupsWrapper, projectGroups, &projModel, nil, featureFlagsWrapper)
	if groupErr != nil {
		return "", groupErr
	}

	if projectPrivatePackage != "" {
		projModel.PrivatePackage, _ = strconv.ParseBool(projectPrivatePackage)
	}
	projModel.Tags = createTagMap(projectTags)
	logger.PrintIfVerbose("Creating new project")
	resp, errorModel, err := projectsWrapper.Create(&projModel)
	projectID := ""
	if errorModel != nil {
		err = errors.Errorf(ErrorCodeFormat, FailedCreatingProj, errorModel.Code, errorModel.Message)
	}
	if err == nil {
		projectID = resp.ID
		if applicationName != "" || len(applicationID) > 0 {
			err = verifyApplicationAssociationDone(applicationName, projectID, applicationsWrapper)
			if err != nil {
				return projectID, err
			}
		}

		if projectGroups != "" {
			err = UpsertProjectGroupsInCreate(groupsWrapper, &projModel, projectsWrapper, accessManagementWrapper, nil, projectGroups, projectID, projectName, featureFlagsWrapper)
			if err != nil {
				return projectID, err
			}
		}
	}
	return projectID, err
}

func verifyApplicationAssociationDone(applicationName, projectID string, applicationsWrapper wrappers.ApplicationsWrapper) error {
	var applicationRes *wrappers.ApplicationsResponseModel
	var err error
	params := make(map[string]string)
	params["name"] = applicationName

	logger.PrintIfVerbose("polling application until project association done or timeout of 2 min")
	var timeoutDuration = 2 * time.Minute
	timeout := time.Now().Add(timeoutDuration)
	for time.Now().Before(timeout) {
		applicationRes, err = applicationsWrapper.Get(params)
		if err != nil {
			return err
		} else if applicationRes != nil && len(applicationRes.Applications) > 0 &&
			slices.Contains(applicationRes.Applications[0].ProjectIds, projectID) {
			logger.PrintIfVerbose("application association done successfully")
			return nil
		} else if time.Now().After(timeout) {
			return errors.Errorf("%s: %v", failedProjectApplicationAssociation, "timeout of 2 min for association")
		}
		time.Sleep(time.Second)
		logger.PrintIfVerbose("application association polling - waiting for associating to complete")
	}

	return errors.Errorf("%s: %v", failedProjectApplicationAssociation, "timeout of 2 min for association")
}

//nolint:gocyclo
func updateProject(
	resp *wrappers.ProjectsCollectionResponseModel,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	projectName string,
	applicationID []string,
	projectGroups string,
	projectTags string,
	projectPrivatePackage string,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,

) (string, error) {
	var projectID string
	applicationName, _ := cmd.Flags().GetString(commonParams.ApplicationName)
	var projModel = wrappers.Project{}
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

	logger.PrintIfVerbose("Fetching existing Project for updating")
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

	if applicationName != "" || len(applicationID) > 0 {
		err = verifyApplicationAssociationDone(applicationName, projectID, applicationsWrapper)
		if err != nil {
			return projectID, err
		}
	}

	if projectGroups != "" {
		err = UpsertProjectGroupsInUpdate(groupsWrapper, &projModel, projectsWrapper, accessManagementWrapper, projModelResp, projectGroups, projectID, projectName, featureFlagsWrapper)
		if err != nil {
			return projectID, err
		}
	}
	return projectID, nil
}

func UpsertProjectGroupsInCreate(groupsWrapper wrappers.GroupsWrapper, projModel *wrappers.Project, projectsWrapper wrappers.ProjectsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, projModelResp *wrappers.ProjectResponseModel,
	projectGroups string, projectID string, projectName string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	return UpsertProjectGroups(groupsWrapper, projModel, projectsWrapper, accessManagementWrapper, projModelResp, projectGroups, projectID, projectName, featureFlagsWrapper)

}

func UpsertProjectGroupsInUpdate(groupsWrapper wrappers.GroupsWrapper, projModel *wrappers.Project, projectsWrapper wrappers.ProjectsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, projModelResp *wrappers.ProjectResponseModel,
	projectGroups string, projectID string, projectName string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	_, groupErr := GetGroupMap(groupsWrapper, projectGroups, projModel, projModelResp, featureFlagsWrapper)
	if groupErr != nil {
		return groupErr
	}
	return UpsertProjectGroups(groupsWrapper, projModel, projectsWrapper, accessManagementWrapper, projModelResp, projectGroups, projectID, projectName, featureFlagsWrapper)

}

func UpsertProjectGroups(groupsWrapper wrappers.GroupsWrapper, projModel *wrappers.Project, projectsWrapper wrappers.ProjectsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, projModelResp *wrappers.ProjectResponseModel,
	projectGroups string, projectID string, projectName string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	groupsMap, groupErr := GetGroupMap(groupsWrapper, projectGroups, projModel, projModelResp, featureFlagsWrapper)
	if groupErr != nil {
		return groupErr
	}

	err := AssignGroupsToProjectNewAccessManagement(projectID, projectName, groupsMap, accessManagementWrapper, featureFlagsWrapper)
	if err != nil {
		return err
	}

	logger.PrintIfVerbose("Updating project groups")
	err = projectsWrapper.Update(projectID, projModel)
	if err != nil {
		return errors.Errorf("%s: %v", failedUpdatingProj, err)
	}
	return nil
}
