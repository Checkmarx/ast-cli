package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	ConversationIDErrorFormat           = "Invalid conversation ID %s"
	AiGuidedRemediationEnabled          = "scan.config.plugins.aiGuidedRemediation"
	ChatGPTGuidedRemediationEnabled     = "scan.config.plugins.chatGPTGuidedRemediation"     // todo: check final value
	AzureAiGuidedRemediationEnabled     = "scan.config.plugins.azureAiGuidedRemediation"     // todo: check final value
	AzureAiEndPoint                     = "scan.config.plugins.azureAiEndPoint"              // todo: check final value
	AzureAiModel                        = "scan.config.plugins.azureAiModel"                 // todo: check final value
	CheckmarxAiGuidedRemediationEnabled = "scan.config.plugins.checkmarxAiGuidedRemediation" // todo: check final value // final value
)

func NewChatCommand(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:    "chat",
		Short:  "Chat with OpenAI models",
		Long:   "Chat with OpenAI models regarding KICS or SAST results",
		Hidden: true,
	}
	chatKicsCmd := ChatKicsSubCommand(chatWrapper, tenantWrapper)
	chatSastCmd := ChatSastSubCommand(chatWrapper, tenantWrapper)

	chatCmd.AddCommand(chatKicsCmd, chatSastCmd)
	return chatCmd
}
