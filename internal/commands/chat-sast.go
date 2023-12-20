package commands

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/commands/chatsast"
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

func NewChatSastCommand(sastChatWrapper wrappers.ChatSastWrapper) *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chatsast",
		Short: "OpenAI-based SAST results remediation",
		Long:  "Use OpenAI models to remediate SAST results and chat about them",
		RunE:  runSastChat(sastChatWrapper),
	}

	chatCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatCmd.Flags().String(params.ChatSastScanResultsFile, "", "Results file in JSON format containing SAST scan results")
	chatCmd.Flags().String(params.ChatSastSourceDir, "", "Source code root directory relevant for the results file")
	chatCmd.Flags().String(params.ChatSastLanguage, "", "Language of the result to remediate")
	chatCmd.Flags().String(params.ChatSastQuery, "", "Query of the result to remediate")
	chatCmd.Flags().String(params.ChatSastResultId, "", "ID of the result to remediate")

	_ = chatCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatCmd.MarkFlagRequired(params.ChatSastScanResultsFile)
	_ = chatCmd.MarkFlagRequired(params.ChatSastSourceDir)
	_ = chatCmd.MarkFlagRequired(params.ChatSastResultId)

	return chatCmd
}

func runSastChat(sastChatWrapper wrappers.ChatSastWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		chatAPIKey, _ := cmd.Flags().GetString(params.ChatAPIKey)
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		scanResultsFile, _ := cmd.Flags().GetString(params.ChatSastScanResultsFile)
		sourceDir, _ := cmd.Flags().GetString(params.ChatSastSourceDir)
		//sastLanguage, _ := cmd.Flags().GetString(params.SastLanguage) // TODO: add support for language
		//sastQuery, _ := cmd.Flags().GetString(params.SastQuery) // TODO: add support for query
		sastResultId, _ := cmd.Flags().GetString(params.ChatSastResultId)

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

		scanResults, err := chatsast.ReadResultsSAST(scanResultsFile)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(ScanResultsFileErrorFormat, scanResultsFile, err))
		}

		if sastResultId == "" {
			msg := fmt.Sprintf("currently only %s is supported", params.ChatSastResultId)
			logger.PrintIfVerbose(msg)
			return outputError(cmd, uuid.Nil, errors.Errorf(msg))
		}

		//languages := GetLanguages(scanResults, sastLanguage)
		//queriesByLanguage := GetQueries(scanResults, languages, sastQuery)
		sastResult, err := chatsast.GetResultById(scanResults, sastResultId)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		sources, err := chatsast.GetSourcesForResult(sastResult, sourceDir)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		promptTemplate, err := chatsast.ReadPromptTemplate("internal/commands/sastchat/prompts/CEF_MethodsAndNodes.txt")
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, err)
		}

		prompt, err := chatsast.CreatePromptWithSource(sastResult, sources, promptTemplate)
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
