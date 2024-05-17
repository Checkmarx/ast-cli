package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const ConversationIDErrorFormat = "Invalid conversation ID %s"

func NewChatCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with OpenAI models",
		Long:  "Chat with OpenAI models regarding KICS or SAST results",
	}
	chatKicsCmd := ChatKicsSubCommand(chatWrapper)
	chatSastCmd := ChatSastSubCommand(chatWrapper)

	chatCmd.AddCommand(chatKicsCmd, chatSastCmd)
	return chatCmd
}
