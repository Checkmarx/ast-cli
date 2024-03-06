package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	ConversationIDErrorFormat  = "Invalid conversation ID %s"
	AiGuidedRemediationEnabled = "scan.config.plugins.aiGuidedRemediation"
)

func NewChatCommand(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:    "chat",
		Short:  "Chat with OpenAI models",
		Long:   "Chat with OpenAI models regarding KICS or SAST results",
		Hidden: true,
	}
	chatKicsCmd := ChatKicsSubCommand(chatWrapper)
	chatSastCmd := ChatSastSubCommand(chatWrapper, tenantWrapper)

	chatCmd.AddCommand(chatKicsCmd, chatSastCmd)
	return chatCmd
}
