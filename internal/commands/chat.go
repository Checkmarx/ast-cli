package commands

import (
	"fmt"
	"os"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
	"github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const systemInput = `You are the Checkmarx AI Guided Remediation bot who can answer technical questions related to the results of KICS.

You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.

You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.

 

If a question irrelevant to the mentioned KICS source or result is asked, answer 'I am the AI Guided Remediation bot and can answer only on questions related to the selected result'.`

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

// dropLen number of messages to drop when limit is reached, 4 due to 2 from prompt, 1 from user question, 1 from reply
const dropLen = 4

const ConversationIDErrorFormat = "Invalid conversation ID %s."
const FileErrorFormat = "It seems that %s is not available for AI Guided Remediation. Please ensure that you have opened the correct workspace or the relevant file."

type OutputModel struct {
	ConversationID string   `json:"conversationId"`
	Response       []string `json:"response"`
}

func NewChatCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Interact with OpenAI models",
		Long:  "Interact with OpenAI models",
		RunE:  runChat(chatWrapper),
	}

	chatCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatCmd.Flags().String(params.ChatResultFile, "", "IaC result code file")
	chatCmd.Flags().String(params.ChatResultLine, "", "IaC result line")
	chatCmd.Flags().String(params.ChatResultSeverity, "", "IaC result severity")
	chatCmd.Flags().String(params.ChatResultVulnerability, "", "IaC result vulnerability name")

	_ = chatCmd.MarkFlagRequired(params.ChatUserInput)
	_ = chatCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatCmd.MarkFlagRequired(params.ChatResultFile)
	_ = chatCmd.MarkFlagRequired(params.ChatResultLine)
	_ = chatCmd.MarkFlagRequired(params.ChatResultSeverity)
	_ = chatCmd.MarkFlagRequired(params.ChatResultVulnerability)

	return chatCmd
}

func runChat(chatWrapper wrappers.ChatWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		chatAPIKey, _ := cmd.Flags().GetString(params.ChatAPIKey)
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		chatResultFile, _ := cmd.Flags().GetString(params.ChatResultFile)
		chatResultLine, _ := cmd.Flags().GetString(params.ChatResultLine)
		chatResultSeverity, _ := cmd.Flags().GetString(params.ChatResultSeverity)
		chatResultVulnerability, _ := cmd.Flags().GetString(params.ChatResultVulnerability)
		userInput, _ := cmd.Flags().GetString(params.ChatUserInput)

		statefulWrapper := wrapper.NewStatefulWrapper(connector.NewFileSystemConnector(""), chatAPIKey, chatModel, dropLen)

		if chatConversationID == "" {
			chatConversationID = statefulWrapper.GenerateId().String()
		}

		id, err := uuid.Parse(chatConversationID)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(ConversationIDErrorFormat, chatConversationID))
		}

		chatResultCode, err := os.ReadFile(chatResultFile)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(FileErrorFormat, chatResultFile))
		}

		newMessages := buildMessages(chatResultCode, chatResultVulnerability, chatResultLine, chatResultSeverity, userInput)
		response, err := chatWrapper.Call(statefulWrapper, id, newMessages)
		if err != nil {
			return outputError(cmd, id, err)
		}

		responseContent := getMessageContents(response)

		return printer.Print(cmd.OutOrStdout(), &OutputModel{
			ConversationID: id.String(),
			Response:       responseContent,
		}, printer.FormatJSON)
	}
}

func getMessageContents(response []message.Message) []string {
	var responseContent []string
	for _, r := range response {
		responseContent = append(responseContent, r.Content)
	}
	return responseContent
}

func buildMessages(chatResultCode []byte,
	chatResultVulnerability, chatResultLine, chatResultSeverity, userInput string) []message.Message {
	var newMessages []message.Message
	newMessages = append(newMessages, message.Message{
		Role:    role.System,
		Content: systemInput,
	}, message.Message{
		Role:    role.Assistant,
		Content: fmt.Sprintf(assistantInputFormat, string(chatResultCode), chatResultVulnerability, chatResultLine, chatResultSeverity),
	}, message.Message{
		Role:    role.User,
		Content: fmt.Sprintf(userInputFormat, userInput),
	})
	return newMessages
}

func outputError(cmd *cobra.Command, id uuid.UUID, err error) error {
	return printer.Print(cmd.OutOrStdout(), &OutputModel{
		ConversationID: id.String(),
		Response:       []string{err.Error()},
	}, printer.FormatJSON)
}
