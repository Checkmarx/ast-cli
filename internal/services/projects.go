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
) (string, error) {
	params := make(map[string]string)
	params["names"] = projectName
	resp, _, err := projectsWrapper.Get(params)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(resp.Projects); i++ {
		if resp.Projects[i].Name == projectName {
			return updateProject(resp, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationWrapper, projectName, applicationID)
		}
	}

	projectGroups, _ := cmd.Flags().GetString(commonParams.ProjectGroupList)
	projectPrivatePackage, _ := cmd.Flags().GetString(commonParams.ProjecPrivatePackageFlag)
	projectID, err := createProject(projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper, applicationWrapper, applicationID, projectGroups, projectPrivatePackage)
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
	applicationsWrapper wrappers.ApplicationsWrapper,
	applicationID []string,
	projectGroups string,
	projectPrivatePackage string,
) (string, error) {
	projectTags, _ := cmd.Flags().GetString(commonParams.ProjectTagList)
	var projModel = wrappers.Project{}
	projModel.Name = projectName
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
		if len(applicationID) > 0 {
			err = verifyApplicationAssociationDone(applicationID, projectID, applicationsWrapper)
			if err != nil {
				return projectID, err
			}
		}

		if projectGroups != "" {
			err = UpsertProjectGroups(groupsWrapper, &projModel, projectsWrapper, accessManagementWrapper, nil, projectGroups, projectID, projectName)
			if err != nil {
				return projectID, err
			}
		}
	}
	return projectID, err
}

func verifyApplicationAssociationDone(applicationID []string, projectID string, applicationsWrapper wrappers.ApplicationsWrapper) error {
	var applicationRes *wrappers.ApplicationsResponseModel
	var err error
	params := make(map[string]string)
	params["id"] = applicationID[0]

	logger.PrintIfVerbose("polling application until project association done or timeout of 2 min")
	start := time.Now()
	timeout := 2 * time.Minute
	for applicationRes != nil && len(applicationRes.Applications) > 0 &&
		!slices.Contains(applicationRes.Applications[0].ProjectIds, projectID) {
		applicationRes, err = applicationsWrapper.Get(params)
		if err != nil {
			return err
		} else if time.Since(start) < timeout {
			return errors.Errorf("%s: %v", failedProjectApplicationAssociation, "timeout of 2 min for association")
		}
	}

	logger.PrintIfVerbose("application association done successfully")
	return nil
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

	if len(applicationID) > 0 {
		err = verifyApplicationAssociationDone(applicationID, projectID, applicationsWrapper)
		if err != nil {
			return projectID, err
		}
	}

	if projectGroups != "" {
		err = UpsertProjectGroups(groupsWrapper, &projModel, projectsWrapper, accessManagementWrapper, projModelResp, projectGroups, projectID, projectName)
		if err != nil {
			return projectID, err
		}
	}
	return projectID, nil
}

func UpsertProjectGroups(groupsWrapper wrappers.GroupsWrapper, projModel *wrappers.Project, projectsWrapper wrappers.ProjectsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, projModelResp *wrappers.ProjectResponseModel,
	projectGroups string, projectID string, projectName string) error {
	groupsMap, groupErr := CreateGroupsMap(projectGroups, groupsWrapper)
	if groupErr != nil {
		return errors.Errorf("%s: %v", failedUpdatingProj, groupErr)
	}

	projModel.Groups = getGroupsForRequest(groupsMap)
	if projModelResp != nil {
		groups := append(getGroupsForRequest(groupsMap), projModelResp.Groups...)
		projModel.Groups = groups
	}

	err := AssignGroupsToProjectNewAccessManagement(projectID, projectName, groupsMap, accessManagementWrapper)
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
