package commands

import (
	"fmt"
	"strconv"

	sastchat "github.com/Checkmarx/gen-ai-prompts/prompts/sast_result_remediation"
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

const UserInputRequiredErrorFormat = "%s is required when %s is provided"
const AiGuidedRemediationDisabledError = "The AI Guided Remediation is disabled in your tenant account"

func ChatSastSubCommand(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	chatSastCmd := &cobra.Command{
		Use:    "sast",
		Short:  "OpenAI-based SAST results remediation",
		Long:   "Use OpenAI models to remediate SAST results and chat about them",
		Hidden: true,
		RunE:   runChatSast(chatWrapper, tenantWrapper),
	}

	chatSastCmd.Flags().String(params.ChatAPIKey, "", "OpenAI API key")
	chatSastCmd.Flags().String(params.ChatConversationID, "", "ID of existing conversation")
	chatSastCmd.Flags().String(params.ChatUserInput, "", "User question")
	chatSastCmd.Flags().String(params.ChatModel, "", "OpenAI model version")
	chatSastCmd.Flags().String(params.ChatSastScanResultsFile, "", "Results file in JSON format containing SAST scan results")
	chatSastCmd.Flags().String(params.ChatSastSourceDir, "", "Source code root directory relevant for the results file")
	chatSastCmd.Flags().String(params.ChatSastResultID, "", "ID of the result to remediate")

	_ = chatSastCmd.MarkFlagRequired(params.ChatAPIKey)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastScanResultsFile)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastSourceDir)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastResultID)

	return chatSastCmd
}

func runChatSast(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if !isAiGuidedRemediationEnabled(tenantWrapper) {
			return outputError(cmd, uuid.Nil, errors.Errorf(AiGuidedRemediationDisabledError))
		}
		chatAPIKey, _ := cmd.Flags().GetString(params.ChatAPIKey)
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		scanResultsFile, _ := cmd.Flags().GetString(params.ChatSastScanResultsFile)
		sourceDir, _ := cmd.Flags().GetString(params.ChatSastSourceDir)
		sastResultID, _ := cmd.Flags().GetString(params.ChatSastResultID)

		statefulWrapper := wrapper.NewStatefulWrapper(connector.NewFileSystemConnector(""), chatAPIKey, chatModel, dropLen, 0)

		newConversation := false
		var userInput string
		if chatConversationID == "" {
			newConversation = true
			chatConversationID = statefulWrapper.GenerateId().String()
		} else {
			userInput, _ = cmd.Flags().GetString(params.ChatUserInput)
			if userInput == "" {
				msg := fmt.Sprintf(UserInputRequiredErrorFormat, params.ChatUserInput, params.ChatConversationID)
				logger.PrintIfVerbose(msg)
				return outputError(cmd, uuid.Nil, errors.Errorf(msg))
			}
		}

		id, err := uuid.Parse(chatConversationID)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return outputError(cmd, id, errors.Errorf(ConversationIDErrorFormat, chatConversationID))
		}

		var newMessages []message.Message
		if newConversation {
			systemPrompt, userPrompt, e := sastchat.BuildPrompt(scanResultsFile, sastResultID, sourceDir)
			if e != nil {
				logger.PrintIfVerbose(e.Error())
				return outputError(cmd, id, e)
			}
			newMessages = append(newMessages, message.Message{
				Role:    role.System,
				Content: systemPrompt,
			}, message.Message{
				Role:    role.User,
				Content: userPrompt,
			})
		} else {
			newMessages = append(newMessages, message.Message{
				Role:    role.User,
				Content: userInput,
			})
		}

		response, err := chatWrapper.Call(statefulWrapper, id, newMessages)
		if err != nil {
			return outputError(cmd, id, err)
		}

		responseContent := getMessageContents(response)

		responseContent = sastchat.AddDescriptionForIdentifier(responseContent)

		return printer.Print(cmd.OutOrStdout(), &OutputModel{
			ConversationID: id.String(),
			Response:       responseContent,
		}, printer.FormatJSON)
	}
}

func isAiGuidedRemediationEnabled(tenantWrapper wrappers.TenantConfigurationWrapper) bool {
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

func getMessageContents(response []message.Message) []string {
	var responseContent []string
	for _, r := range response {
		responseContent = append(responseContent, r.Content)
	}
	return responseContent
}
