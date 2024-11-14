package commands

import (
	"encoding/json"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func updateGroupValues(input *[]byte, cmd *cobra.Command, groupsWrapper wrappers.GroupsWrapper) ([]*wrappers.Group, error) {
	groupListStr, _ := cmd.Flags().GetString(commonParams.GroupList)
	groups, err := services.CreateGroupsMap(groupListStr, groupsWrapper)
	if err != nil {
		return groups, err
	}

	// we're not checking here status of the feature flag, because of refactoring in AM
	var info map[string]interface{}
	_ = json.Unmarshal(*input, &info)
	info["groups"] = services.GetGroupIds(groups)
	*input, _ = json.Marshal(info)

	return groups, nil
}
