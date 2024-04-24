package services

import (
	"strings"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

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
