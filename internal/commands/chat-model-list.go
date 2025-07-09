package commands

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func ChatModelListSubCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatModelListCmd := &cobra.Command{
		Use:    "openai-models",
		Short:  "Lists available OpenAI models",
		Long:   "Lists available OpenAI models to choose from",
		Hidden: true,
		RunE:   runChatModelList(chatWrapper),
	}

	chatModelListCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	_ = chatModelListCmd.MarkFlagRequired(params.ChatAPIKey)
	return chatModelListCmd
}

func runChatModelList(chatWrapper wrappers.ChatWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		apiKey, err := cmd.Flags().GetString(params.ChatAPIKey)
		if err != nil {
			return fmt.Errorf("error to read api key")
		}
		// fmt.Println("API key ::", apiKey)
		ownedBy := []string{}
		openAIresponse := chatWrapper.GetModelList(apiKey, ownedBy)
		// var outPutModel OutputModel
		return printer.Print(cmd.OutOrStdout(), openAIresponse, printer.FormatJSON)
	}
}
