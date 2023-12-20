package commands

import (
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewChatCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with OpenAI models",
		Long:  "Chat with OpenAI models regarding KICS or SAST results",
	}
	chatKicsCmd := chatKicsSubCommand(chatWrapper)
	chatSastCmd := chatSastSubCommand(chatWrapper)

	chatCmd.AddCommand(chatKicsCmd, chatSastCmd)
	return chatCmd
}

func chatKicsSubCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatKicsCmd := &cobra.Command{
		Use:   "kics",
		Short: "Chat about KICS result with OpenAI models",
		Long:  "Chat about KICS result with OpenAI models",
		RunE:  runChatKics(chatWrapper),
	}

	chatKicsCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatKicsCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatKicsCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatKicsCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatKicsCmd.Flags().String(params.ChatKicsResultFile, "", "IaC result code file")
	chatKicsCmd.Flags().String(params.ChatKicsResultLine, "", "IaC result line")
	chatKicsCmd.Flags().String(params.ChatKicsResultSeverity, "", "IaC result severity")
	chatKicsCmd.Flags().String(params.ChatKicsResultVulnerability, "", "IaC result vulnerability name")

	_ = chatKicsCmd.MarkFlagRequired(params.ChatUserInput)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultFile)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultLine)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultSeverity)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultVulnerability)

	return chatKicsCmd
}

func chatSastSubCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatSastCmd := &cobra.Command{
		Use:   "sast",
		Short: "OpenAI-based SAST results remediation",
		Long:  "Use OpenAI models to remediate SAST results and chat about them",
		RunE:  runChatSast(chatWrapper),
	}

	chatSastCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatSastCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatSastCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatSastCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatSastCmd.Flags().String(params.ChatSastScanResultsFile, "", "Results file in JSON format containing SAST scan results")
	chatSastCmd.Flags().String(params.ChatSastSourceDir, "", "Source code root directory relevant for the results file")
	chatSastCmd.Flags().String(params.ChatSastLanguage, "", "Language of the result to remediate")
	chatSastCmd.Flags().String(params.ChatSastQuery, "", "Query of the result to remediate")
	chatSastCmd.Flags().String(params.ChatSastResultId, "", "ID of the result to remediate")

	_ = chatSastCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastScanResultsFile)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastSourceDir)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastResultId)

	return chatSastCmd
}
