package services

import (
	"slices"
	"strconv"
	"time"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
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
	projectName string,
	cmd *cobra.Command,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	applicationWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) (string, error) {
	resp, err := GetProjectsCollectionByProjectName(projectName, projectsWrapper)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
			projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
			return updateProject(
				resp,
				cmd,
				projectsWrapper,
				accessManagementWrapper,
				projectName,
				projectTags,
				projectPrivatePackage,
				featureFlagsWrapper)
		}
	}

	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)

	applicationName, _ := cmd.Flags().GetString(commonParams.ApplicationName)
	applicationID, appErr := getApplicationID(applicationName, applicationWrapper)
	if appErr != nil {
		return "", appErr
	}

	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationWrapper,
		applicationID, projectGroups, projectPrivatePackage, featureFlagsWrapper)
	if err != nil {
		logger.PrintIfVerbose("error in creating project!")
		return "", err
	}
	return projectID, nil
}

func GetProjectsCollectionByProjectName(projectName string, projectsWrapper wrappers.ProjectsWrapper) (*wrappers.ProjectsCollectionResponseModel, error) {
	params := make(map[string]string)
	params["name"] = projectName
	resp, _, err := projectsWrapper.Get(params)

	if err != nil {
		logger.PrintIfVerbose(err.Error())
		return nil, err
	}

	if resp == nil {
		EmptyProjects := []wrappers.ProjectResponseModel{}
		emptyProjectsCollection := &wrappers.ProjectsCollectionResponseModel{
			TotalCount:         0,
			FilteredTotalCount: 0,
			Projects:           EmptyProjects,
		}
		return emptyProjectsCollection, nil
	}

	return resp, nil
}

func getApplicationID(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper) ([]string, error) {
	var applicationID []string
	if applicationName != "" {
		application, getAppErr := GetApplication(applicationName, applicationsWrapper)
		if getAppErr != nil {
			return nil, getAppErr
		}
		if application == nil {
			return nil, errors.Errorf(errorConstants.ApplicationDoesntExistOrNoPermission)
		}
		applicationID = []string{application.ID}
	}
	return applicationID, nil
}

func GetApplication(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper) (*wrappers.Application, error) {
	if applicationName != "" {
		params := make(map[string]string)
		params["name"] = applicationName
		resp, err := applicationsWrapper.Get(params)
		if err != nil {
			return nil, err
		}
		if resp.Applications != nil && len(resp.Applications) > 0 {
			application := verifyApplicationNameExactMatch(applicationName, resp)

			return application, nil
		}
	}
	return nil, nil
}

func verifyApplicationNameExactMatch(applicationName string, resp *wrappers.ApplicationsResponseModel) *wrappers.Application {
	var application *wrappers.Application
	for i := range resp.Applications {
		if resp.Applications[i].Name == applicationName {
			application = &resp.Applications[i]
			break
		}
	}
	return application
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
	var groupsMap []*wrappers.Group
	if projectGroups != "" {
		var groups []string
		var groupErr error
		groupsMap, groups, groupErr = GetGroupMap(groupsWrapper, projectGroups, nil)
		if groupErr != nil {
			return "", groupErr
		}
		projModel.Groups = groups
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
			err = UpsertProjectGroups(&projModel, projectsWrapper, accessManagementWrapper, projectID, projectName, featureFlagsWrapper, groupsMap)
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
	accessManagementWrapper wrappers.AccessManagementWrapper,
	projectName string,
	projectTags string,
	projectPrivatePackage string,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,

) (string, error) {
	var projectID string
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
	if projectTags == "" && projectPrivatePackage == "" {
		logger.PrintIfVerbose("No tags to update. Skipping project update.")
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

	err = projectsWrapper.Update(projectID, &projModel)
	if err != nil {
		return "", errors.Errorf("%s: %v", failedUpdatingProj, err)
	}

	return projectID, nil
}

func UpsertProjectGroups(projModel *wrappers.Project, projectsWrapper wrappers.ProjectsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, projectID string, projectName string,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper, groupsMap []*wrappers.Group) error {
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
