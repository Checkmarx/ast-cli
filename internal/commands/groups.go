package commands

import (
	"encoding/json"
	"log"
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const accessManagementEnabled = "ACCESS_MANAGEMENT_ENABLED" // feature flag

func createGroupsMap(groupsStr string, groupsWrapper wrappers.GroupsWrapper) ([]*wrappers.Group, error) {
	groups := strings.Split(groupsStr, ",")
	var groupMap []*wrappers.Group
	var groupsNotFound []string
	for _, group := range groups {
		if len(group) > 0 {
			groupIds, err := groupsWrapper.Get(group)
			if err != nil {
				groupsNotFound = append(groupsNotFound, group)
			} else {
				findGroup := findGroupByName(groupIds, group)
				if findGroup.Name != "" {
					groupMap = append(groupMap, &findGroup)
				} else {
					groupsNotFound = append(groupsNotFound, group)
				}
			}
		}
	}

	if len(groupsNotFound) > 0 {
		return nil, errors.Errorf("%s: %v", failedFindingGroup, groupsNotFound)
	}

	return groupMap, nil
}

func findGroupByName(groups []wrappers.Group, name string) wrappers.Group {
	for i := 0; i < len(groups); i++ {
		if groups[i].Name == name {
			return groups[i]
		}
	}
	return wrappers.Group{}
}

func updateGroupValues(input *[]byte, cmd *cobra.Command, groupsWrapper wrappers.GroupsWrapper) ([]*wrappers.Group, error) {
	groupListStr, _ := cmd.Flags().GetString(commonParams.GroupList)

	var groupMap []string
	var info map[string]interface{}
	_ = json.Unmarshal(*input, &info)
	if _, ok := info["groups"]; !ok {
		_ = json.Unmarshal([]byte("[]"), &groupMap)
		info["groups"] = groupMap
	}
	groups, err := createGroupsMap(groupListStr, groupsWrapper)
	if err != nil {
		return groups, err
	}
	if !wrappers.FeatureFlags[accessManagementEnabled] {
		info["groups"] = getGroupIds(groups)
		*input, _ = json.Marshal(info)
	}
	return groups, nil
}

func getGroupIds(groups []*wrappers.Group) []string {
	var groupIds []string
	for _, group := range groups {
		groupIds = append(groupIds, group.ID)
	}
	return groupIds
}

func assignGroupsToProject(projectID string, projectName string, groups []*wrappers.Group,
	accessManagement wrappers.AccessManagementWrapper) error {
	if !wrappers.FeatureFlags[accessManagementEnabled] {
		return nil
	}
	groupsAssignedToTheProject, err := accessManagement.GetGroups(projectID)
	if err != nil {
		return err
	}
	groupsToAssign := getGroupsToAssign(groups, groupsAssignedToTheProject)
	if len(groupsToAssign) == 0 {
		log.Println("No new groups to assign")
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
		} else {
			log.Printf("Group [%s | %s] already assigned", receivedGroup.ID, receivedGroup.Name)
		}
	}
	return groupsToAssign
}
