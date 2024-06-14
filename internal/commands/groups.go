package commands

import (
	"encoding/json"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func updateGroupValues(input *[]byte, cmd *cobra.Command, groupsWrapper wrappers.GroupsWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) ([]*wrappers.Group, error) {
	groupListStr, _ := cmd.Flags().GetString(commonParams.GroupList)
	groups, err := services.CreateGroupsMap(groupListStr, groupsWrapper)
	if err != nil {
		return groups, err
	}
	flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, featureFlagsConstants.AccessManagementEnabled)
	if !flagResponse.Status {
		var info map[string]interface{}
		_ = json.Unmarshal(*input, &info)
		info["groups"] = services.GetGroupIds(groups)
		*input, _ = json.Marshal(info)
	}
	return groups, nil
}
