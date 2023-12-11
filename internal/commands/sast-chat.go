package commands

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/commands/sastchat"
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

const ScanResultsFileErrorFormat = "Error reading and parsing scan results %s: %v"
const CreatePromptErrorFormat = "Error creating prompt for result ID %s: %v"

func NewSastChatCommand(sastChatWrapper wrappers.SastChatWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "sastchat",
		Short: "OpenAI-based SAST results remediation",
		Long:  "Use OpenAI models to remediate SAST results",
		RunE:  runSastChat(sastChatWrapper),
	}

	chatCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatCmd.Flags().String(params.ScanResultsFile, "", "Results file in JSON format containing SAST scan results")
	chatCmd.Flags().String(params.SourceDir, "", "Source code root directory relevant for the results file")
	chatCmd.Flags().String(params.SastLanguage, "", "Language of the result to remediate")
	chatCmd.Flags().String(params.SastQuery, "", "Query of the result to remediate")
	chatCmd.Flags().String(params.SastResultId, "", "ID of the result to remediate")

	_ = chatCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatCmd.MarkFlagRequired(params.ScanResultsFile)
	_ = chatCmd.MarkFlagRequired(params.SourceDir)
	_ = chatCmd.MarkFlagRequired(params.SastResultId)

	return chatCmd
}

func runSastChat(sastChatWrapper wrappers.SastChatWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		chatAPIKey, _ := cmd.Flags().GetString(params.ChatAPIKey)
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		scanResultsFile, _ := cmd.Flags().GetString(params.ScanResultsFile)
		sourceDir, _ := cmd.Flags().GetString(params.SourceDir)
		//sastLanguage, _ := cmd.Flags().GetString(params.SastLanguage) // TODO: add support for language
		//sastQuery, _ := cmd.Flags().GetString(params.SastQuery) // TODO: add support for query
		sastResultId, _ := cmd.Flags().GetString(params.SastResultId)

		statefulWrapper := wrapper.NewStatefulWrapper(connector.NewFileSystemConnector(""), chatAPIKey, chatModel, dropLen, 0)

		if chatConversationID == "" {
			chatConversationID = statefulWrapper.GenerateId().String()
		} else {
			userInput, _ := cmd.Flags().GetString(params.ChatUserInput)
			if userInput == "" {
				msg := fmt.Sprintf("%s is required when %s is provided", params.ChatUserInput, params.ChatConversationID)
				logger.PrintIfVerbose(msg)
				return outputError(cmd, uuid.Nil, errors.Errorf(msg))
			}
		}

		id, err := uuid.Parse(chatConversationID)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(ConversationIDErrorFormat, chatConversationID, err))
		}

		scanResults, err := sastchat.ReadResultsSAST(scanResultsFile)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(ScanResultsFileErrorFormat, scanResultsFile, err))
		}

		if sastResultId == "" {
			msg := fmt.Sprintf("currently only %s is supported", params.SastResultId)
			logger.PrintIfVerbose(msg)
			return outputError(cmd, uuid.Nil, errors.Errorf(msg))
		}

		//languages := GetLanguages(scanResults, sastLanguage)
		//queriesByLanguage := GetQueries(scanResults, languages, sastQuery)
		sastResult, err := sastchat.GetResultById(scanResults, sastResultId)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		sources, err := sastchat.GetSourcesForResult(sastResult, sourceDir)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		promptTemplate, err := sastchat.ReadPromptTemplate("internal/commands/sastchat/prompts/CEF_MethodsAndNodes.txt")
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		prompt, err := sastchat.CreatePromptWithSource(sastResult, sources, promptTemplate)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(CreatePromptErrorFormat, sastResultId, err))
		}

		var newMessages []message.Message
		newMessages = append(newMessages, message.Message{
			Role:    role.User,
			Content: prompt,
		})

		response, err := sastChatWrapper.Call(statefulWrapper, id, newMessages)
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
