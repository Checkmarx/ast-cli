package commands

import (
	"strconv"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	ConversationIDErrorFormat  = "Invalid conversation ID %s"
	AiGuidedRemediationEnabled = "scan.config.plugins.aiGuidedRemediation"
)

func NewChatCommand(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with OpenAI models",
		Long:  "Chat with OpenAI models regarding KICS or SAST results",
	}
	chatCmd.AddCommand(ChatKicsSubCommand(chatWrapper))
	if aiGuidedRemediationEnabled(tenantWrapper) {
		chatCmd.AddCommand(ChatSastSubCommand(chatWrapper))
	}
	return chatCmd
}

func aiGuidedRemediationEnabled(tenantWrapper wrappers.TenantConfigurationWrapper) bool {
	tenantConfigurationResponse, errorModel, err := tenantWrapper.GetTenantConfiguration()
	if err != nil {
		return false
	}
	if errorModel != nil {
		return false
	}
	if tenantConfigurationResponse != nil {
		for _, resp := range *tenantConfigurationResponse {
			if resp.Key == AiGuidedRemediationEnabled {
				isEnabled, _ := strconv.ParseBool(resp.Value)
				return isEnabled
			}
		}
	}
	return false
}
