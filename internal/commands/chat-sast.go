package commands

import (
	"fmt"
	"strconv"
	"strings"

	sastchat "github.com/Checkmarx/gen-ai-prompts/prompts/sast_result_remediation"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/connector"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/role"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	_ = chatSastCmd.MarkFlagRequired(params.ChatSastScanResultsFile)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastSourceDir)
	_ = chatSastCmd.MarkFlagRequired(params.ChatSastResultID)

	return chatSastCmd
}

func runChatSast(
	chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		tenantConfigurationResponses, err := GetTenantConfigurationResponses(tenantWrapper)
		if err != nil {
			return outputError(cmd, uuid.Nil, err)
		}
		if !isAiGuidedRemediationEnabled(tenantConfigurationResponses) {
			return outputError(cmd, uuid.Nil, errors.Errorf(AiGuidedRemediationDisabledError))
		}
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		scanResultsFile, _ := cmd.Flags().GetString(params.ChatSastScanResultsFile)
		sourceDir, _ := cmd.Flags().GetString(params.ChatSastSourceDir)
		sastResultID, _ := cmd.Flags().GetString(params.ChatSastResultID)
		azureAiEnabled := isAzureAiGuidedRemediationEnabled(tenantConfigurationResponses)
		checkmarxAiEnabled := isCheckmarxAiGuidedRemediationEnabled(tenantConfigurationResponses)

		statefulWrapper, customerToken := CreateStatefulWrapper(cmd, azureAiEnabled, checkmarxAiEnabled, tenantConfigurationResponses)

		tenantID := getTenantID(customerToken)

		newConversation, userInput, id, err := getSastConversationDetails(cmd, chatConversationID, statefulWrapper)
		if err != nil {
			return err
		}

		newMessages, err := buildSastMessages(cmd, newConversation, scanResultsFile, sastResultID, sourceDir, id, userInput)
		if err != nil {
			return err
		}

		responseContent, err := sendRequest(statefulWrapper, azureAiEnabled, checkmarxAiEnabled, tenantID, chatWrapper, id, newMessages, customerToken, guidedRemediationFeatureNameSast)
		if err != nil {
			return outputError(cmd, id, err)
		}

		responseContent = sastchat.AddDescriptionForIdentifier(responseContent)

		return printer.Print(cmd.OutOrStdout(), &OutputModel{
			ConversationID: id.String(),
			Response:       responseContent,
		}, printer.FormatJSON)
	}
}

func getSastConversationDetails(cmd *cobra.Command, chatConversationID string, statefulWrapper wrapper.StatefulWrapper) (
	isNewConversation bool, userInput string, conversationID uuid.UUID, err error) {
	newConversation := false
	if chatConversationID == "" {
		newConversation = true
		chatConversationID = statefulWrapper.GenerateId().String()
	} else {
		userInput, _ = cmd.Flags().GetString(params.ChatUserInput)
		if userInput == "" {
			msg := fmt.Sprintf(UserInputRequiredErrorFormat, params.ChatUserInput, params.ChatConversationID)
			logger.PrintIfVerbose(msg)
			return false, "", uuid.UUID{}, outputError(cmd, uuid.Nil, errors.Errorf(msg))
		}
	}

	id, err := uuid.Parse(chatConversationID)
	if err != nil {
		logger.PrintIfVerbose(err.Error())
		return false, "", uuid.UUID{}, outputError(cmd, id, errors.Errorf(ConversationIDErrorFormat, chatConversationID))
	}
	return newConversation, userInput, id, nil
}

func buildSastMessages(cmd *cobra.Command, newConversation bool, scanResultsFile, sastResultID, sourceDir string, id uuid.UUID, userInput string) ([]message.Message, error) {
	var newMessages []message.Message
	if newConversation {
		systemPrompt, userPrompt, e := sastchat.BuildPrompt(scanResultsFile, sastResultID, sourceDir)
		if e != nil {
			logger.PrintIfVerbose(e.Error())
			return nil, outputError(cmd, id, e)
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
	return newMessages, nil
}

func CreateStatefulWrapper(cmd *cobra.Command, azureAiEnabled, checkmarxAiEnabled bool, tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) (
	statefulWrapper wrapper.StatefulWrapper, customerToken string) {
	conn := connector.NewFileSystemConnector("")

	customerToken, _ = wrappers.GetAccessToken()

	if azureAiEnabled {
		aiProxyAzureAIRoute := viper.GetString(params.AiProxyAzureAiRouteKey)
		aiProxyEndPoint, _ := wrappers.GetURL(aiProxyAzureAIRoute, customerToken)
		model, _ := GetAzureAiModel(tenantConfigurationResponses)
		statefulWrapper, _ = wrapper.NewStatefulWrapperNew(conn, aiProxyEndPoint, customerToken, model, dropLen, 0)
	} else if checkmarxAiEnabled {
		aiProxyCheckmarxAIRoute := viper.GetString(params.AiProxyCheckmarxAiRouteKey)
		aiProxyEndPoint, _ := wrappers.GetURL(aiProxyCheckmarxAIRoute, customerToken)
		model := checkmarxAiChatModel
		statefulWrapper, _ = wrapper.NewStatefulWrapperNew(conn, aiProxyEndPoint, customerToken, model, dropLen, 0)
	} else {
		chatModel, _ := cmd.Flags().GetString(params.ChatModel)
		chatAPIKey, _ := cmd.Flags().GetString(params.ChatAPIKey)
		statefulWrapper = wrapper.NewStatefulWrapper(conn, chatAPIKey, chatModel, dropLen, 0)
	}
	return statefulWrapper, customerToken
}

func GetTenantConfigurationResponses(tenantWrapper wrappers.TenantConfigurationWrapper) (*[]*wrappers.TenantConfigurationResponse, error) {
	tenantConfigurationResponse, errorModel, err := tenantWrapper.GetTenantConfiguration()
	if err != nil {
		return nil, err
	}
	if errorModel != nil {
		return nil, errors.New(errorModel.Message)
	}
	return tenantConfigurationResponse, nil
}

func GetTenantConfiguration(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse, configKey string) (string, error) {
	if tenantConfigurationResponses != nil {
		for _, resp := range *tenantConfigurationResponses {
			if resp.Key == configKey {
				return resp.Value, nil
			}
		}
	}
	return "", errors.New(configKey + " not found")
}

func GetTenantConfigurationBool(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse, configKey string) (bool, error) {
	value, err := GetTenantConfiguration(tenantConfigurationResponses, configKey)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(value)
}

func isAiGuidedRemediationEnabled(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) bool {
	isEnabled, err := GetTenantConfigurationBool(tenantConfigurationResponses, AiGuidedRemediationEnabled)
	if err != nil {
		return false
	}
	return isEnabled
}

func isCxOneAPIKeyAvailable() bool {
	apiKey := viper.GetString(params.AstAPIKey)
	return apiKey != ""
}

func isAzureAiGuidedRemediationEnabled(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) bool {
	engine, err := GetTenantConfiguration(tenantConfigurationResponses, AiGuidedRemediationEngine)
	if err != nil {
		return false
	}
	isEnabled := strings.EqualFold(engine, AiGuidedRemediationAzureAiValue)
	return isEnabled
}

func isCheckmarxAiGuidedRemediationEnabled(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) bool {
	engine, err := GetTenantConfiguration(tenantConfigurationResponses, AiGuidedRemediationEngine)
	if err != nil {
		return false
	}
	isEnabled := strings.EqualFold(engine, AiGuidedRemediationCheckmarxAiValue)
	return isEnabled
}

func GetAzureAiModel(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) (string, error) {
	return GetTenantConfiguration(tenantConfigurationResponses, AzureAiModel)
}

func getMessageContents(response []message.Message) []string {
	var responseContent []string
	for _, r := range response {
		responseContent = append(responseContent, r.Content)
	}
	return responseContent
}
