package commands

import (
	"encoding/json"
	"fmt"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"strings"
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
				findGroup := findGroupID(groupIds, group)
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

func findGroupID(groups []wrappers.Group, name string) wrappers.Group {
	for i := 0; i < len(groups); i++ {
		if groups[i].Name == name {
			return groups[i]
		}
	}
	return wrappers.Group{}
}

func updateGroupValues(input *[]byte, cmd *cobra.Command, groupsWrapper wrappers.GroupsWrapper) error {
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
		return err
	}

	info["groups"] = getGroupIds(groups)
	*input, _ = json.Marshal(info)

	return nil
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
	log.Println("Creating groups assignment...")
	groupsAssignedToTheProject, err := accessManagement.GetGroups(projectID)
	if err != nil {
		return err
	}
	groupsToAssign := getGroupsToAssign(groups, groupsAssignedToTheProject)

	err = accessManagement.CreateGroupsAssignment(projectID, projectName, groupsToAssign)
	if err != nil {
		return err
	}
	return nil
}

func getGroupsToAssign(receivedGroups, existingGroups []*wrappers.Group) []*wrappers.Group {
	var groupsToAssign []*wrappers.Group
	for _, receivedGroup := range receivedGroups {
		var find bool
		for _, existingGroup := range existingGroups {
			if receivedGroup.ID == existingGroup.ID {
				find = true
				log.Println(fmt.Sprintf("Group [%s | %s] already assigned", receivedGroup.ID, receivedGroup.Name))
				break
			}
		}
		if !find {
			groupsToAssign = append(groupsToAssign, receivedGroup)
		}
	}
	return groupsToAssign

}
