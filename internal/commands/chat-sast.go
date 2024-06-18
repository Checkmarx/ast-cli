package commands

import (
	"fmt"
	"strconv"

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

const ScanResultsFileErrorFormat = "Error reading and parsing scan results %s"
const CreatePromptErrorFormat = "Error creating prompt for result ID %s"
const UserInputRequiredErrorFormat = "%s is required when %s is provided"
const AiGuidedRemediationDisabledError = "The AI Guided Remediation is disabled in your tenant account"
const AllOptionsDisabledError = "All AI Guided Remediation options are disabled in your tenant account" // check final value

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
		chatGptEnabled := isChatGPTAiGuidedRemediationEnabled(tenantConfigurationResponses)

		statefulWrapper, customerToken := CreateStatefulWrapper(cmd, azureAiEnabled, checkmarxAiEnabled, tenantConfigurationResponses)

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
			systemPrompt, userPrompt, e := buildPrompt(scanResultsFile, sastResultID, sourceDir)
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

		tenantID, _ := wrappers.ExtractFromTokenClaims(customerToken, tenantIDClaimKey)
		requestID := statefulWrapper.GenerateId().String()

		var response []message.Message
		if azureAiEnabled || checkmarxAiEnabled {
			metadata := message.MetaData{
				TenantID:  tenantID,
				RequestID: requestID,
				UserAgent: params.DefaultAgent,
				Feature:   guidedRemediationFeatureNameSast,
			}
			if azureAiEnabled {
				logger.PrintIfVerbose("Sending message to Azure AI model for SAST guided remediation. RequestID: " + requestID)

			} else {
				logger.PrintIfVerbose("Sending message to Checkmarx AI model for SAST guided remediation. RequestID: " + requestID)
			}
			response, err = chatWrapper.SecureCall(statefulWrapper, id, newMessages, &metadata, customerToken)
			if err != nil {
				return outputError(cmd, id, err)
			}
		} else if chatGptEnabled {
			logger.PrintIfVerbose("Sending message to ChatGPT model for SAST guided remediation. RequestID: " + requestID)
			response, err = chatWrapper.Call(statefulWrapper, id, newMessages)
			if err != nil {
				return outputError(cmd, id, err)
			}
		} else {
			return outputError(cmd, uuid.Nil, errors.Errorf(AllOptionsDisabledError))
		}

		responseContent := getMessageContents(response)

		responseContent = addDescriptionForIdentifier(responseContent)

		return printer.Print(cmd.OutOrStdout(), &OutputModel{
			ConversationID: id.String(),
			Response:       responseContent,
		}, printer.FormatJSON)
	}
}

func CreateStatefulWrapper(cmd *cobra.Command, azureAiEnabled, checkmarxAiEnabled bool, tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) (
	statefulWrapper wrapper.StatefulWrapper, customerToken string) {
	conn := connector.NewFileSystemConnector("")

	customerToken, _ = wrappers.GetAccessToken()

	if azureAiEnabled {
		aiProxyEndPoint, _ := wrappers.GetURL(aiProxyAzureAIRoute, customerToken)
		model, _ := GetAzureAiModel(tenantConfigurationResponses)
		statefulWrapper, _ = wrapper.NewStatefulWrapperNew(conn, aiProxyEndPoint, customerToken, model, dropLen, 0) // todo: check final interface
	} else if checkmarxAiEnabled {
		aiProxyEndPoint, _ := wrappers.GetURL(aiProxyCheckmarxAIRoute, customerToken)
		model := checkmarxAiChatModel
		statefulWrapper, _ = wrapper.NewStatefulWrapperNew(conn, aiProxyEndPoint, customerToken, model, dropLen, 0) // todo: check final interface
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
	isEnabled, err := GetTenantConfigurationBool(tenantConfigurationResponses, AzureAiGuidedRemediationEnabled)
	if err != nil {
		return false
	}
	return isEnabled
}

func isCheckmarxAiGuidedRemediationEnabled(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) bool {
	isEnabled, err := GetTenantConfigurationBool(tenantConfigurationResponses, CheckmarxAiGuidedRemediationEnabled)
	if err != nil {
		return false
	}
	return isEnabled
}

func isChatGPTAiGuidedRemediationEnabled(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) bool {
	isEnabled, err := GetTenantConfigurationBool(tenantConfigurationResponses, ChatGPTGuidedRemediationEnabled)
	if err != nil {
		return false
	}
	return isEnabled
}

func GetAzureAiModel(tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse) (string, error) {
	return GetTenantConfiguration(tenantConfigurationResponses, AzureAiModel)
}

func buildPrompt(scanResultsFile, sastResultID, sourceDir string) (systemPrompt, userPrompt string, err error) {
	scanResults, err := ReadResultsSAST(scanResultsFile)
	if err != nil {
		return "", "", fmt.Errorf("error in build-prompt: %s: %w", fmt.Sprintf(ScanResultsFileErrorFormat, scanResultsFile), err)
	}

	if sastResultID == "" {
		return "", "", errors.Errorf(fmt.Sprintf("error in build-prompt: currently only --%s is supported", params.ChatSastResultID))
	}

	sastResult, err := GetResultByID(scanResults, sastResultID)
	if err != nil {
		return "", "", fmt.Errorf("error in build-prompt: %w", err)
	}

	sources, err := GetSourcesForResult(sastResult, sourceDir)
	if err != nil {
		return "", "", fmt.Errorf("error in build-prompt: %w", err)
	}

	prompt, err := CreateUserPrompt(sastResult, sources)
	if err != nil {
		return "", "", fmt.Errorf("error in build-prompt: %s: %w", fmt.Sprintf(CreatePromptErrorFormat, sastResultID), err)
	}

	return GetSystemPrompt(), prompt, nil
}

func getMessageContents(response []message.Message) []string {
	var responseContent []string
	for _, r := range response {
		responseContent = append(responseContent, r.Content)
	}
	return responseContent
}
