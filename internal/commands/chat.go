package commands

import (
	"fmt"
	"os"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
	"github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const systemInput = `You are the Checkmarx KICS bot who can answer technical questions related to the results of KICS.

You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.

You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.

 

If a question irrelevant to the mentioned KICS source or result is asked, answer 'I am the KICS bot and can answer only on questions related to the selected KICS result'.`

const assistantInputFormat = `Checkmarx KICS has scanned this source code and reported the result.
This is the source code:
'<|KICS_SOURCE_START|>'
%s
'<|KICS_SOURCE_END|>'
and this is the result (vulnerability or security issue) found by KICS:
'<|KICS_RESULT_START|>'
'%s' is detected in line %s with severity '%s'.
'<|KICS_RESULT_END|>'`

const userInputFormat = `The user question is:
'<|KICS_QUESTION_START|>'
"%s"
'<|KICS_QUESTION_END|>'`

type OutputModel struct {
	ConversationId string   `json:"conversationId"`
	Response       []string `json:"response"`
}

func NewChatCommand() *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Interact with OpenAI models",
		Long:  "Interact with OpenAI models",
		RunE:  runChat(),
	}

	chatCmd.Flags().String(params.ChatApiKey, "", "OpenAI API key")
	chatCmd.Flags().String(params.ChatConversationId, "", "ID of existing conversation")
	chatCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatCmd.Flags().String(params.ChatResultFile, "", "IaC result code file")
	chatCmd.Flags().String(params.ChatResultLine, "", "IaC result line")
	chatCmd.Flags().String(params.ChatResultSeverity, "", "IaC result severity")
	chatCmd.Flags().String(params.ChatResultVulnerability, "", "IaC result vulnerability name")

	_ = chatCmd.MarkFlagRequired(params.ChatUserInput)
	_ = chatCmd.MarkFlagRequired(params.ChatApiKey)
	_ = chatCmd.MarkFlagRequired(params.ChatResultFile)
	_ = chatCmd.MarkFlagRequired(params.ChatResultLine)
	_ = chatCmd.MarkFlagRequired(params.ChatResultSeverity)
	_ = chatCmd.MarkFlagRequired(params.ChatResultVulnerability)

	return chatCmd
}

func runChat() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		chatApiKey, _ := cmd.Flags().GetString(params.ChatApiKey)
		chatConversationId, _ := cmd.Flags().GetString(params.ChatConversationId)
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		statefulWrapper := wrapper.NewStatefulWrapper(connector.NewFileSystemConnector(""), chatApiKey, chatModel)

		if chatConversationId == "" {
			chatConversationId = statefulWrapper.GenerateId().String()
		}
		id, err := uuid.Parse(chatConversationId)

		var newMessages []message.Message
		newMessages = append(newMessages, message.Message{
			Role:    role.System,
			Content: systemInput,
		})

		chatResultFile, _ := cmd.Flags().GetString(params.ChatResultFile)
		chatResultLine, _ := cmd.Flags().GetString(params.ChatResultLine)
		chatResultSeverity, _ := cmd.Flags().GetString(params.ChatResultSeverity)
		chatResultVulnerability, _ := cmd.Flags().GetString(params.ChatResultVulnerability)
		chatResultCode, err := os.ReadFile(chatResultFile)
		if err != nil {
			return err
		}
		newMessages = append(newMessages, message.Message{
			Role:    role.Assistant,
			Content: fmt.Sprintf(assistantInputFormat, string(chatResultCode), chatResultVulnerability, chatResultLine, chatResultSeverity),
		})

		userInput, _ := cmd.Flags().GetString(params.ChatUserInput)
		newMessages = append(newMessages, message.Message{
			Role:    role.User,
			Content: fmt.Sprintf(userInputFormat, userInput),
		})
		response, err := statefulWrapper.Call(id, newMessages)
		if err != nil {
			return err
		}

		var responseContent []string
		for _, r := range response {
			responseContent = append(responseContent, r.Content)
		}

		output := OutputModel{
			ConversationId: id.String(),
			Response:       responseContent,
		}

		return printer.Print(cmd.OutOrStdout(), &output, printer.FormatJSON)
	}
}
