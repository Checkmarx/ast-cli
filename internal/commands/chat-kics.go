package commands

import (
	"fmt"
	"os"

	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/role"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const systemInput = `You are the Checkmarx AI Guided Remediation bot who can answer technical questions related to the results of Infrastructure as Code Security.
You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.
You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.
If a question irrelevant to the mentioned Infrastructure as Code Security source or result is asked,
answer 'I am the AI Guided Remediation assistant and can answer only on questions related to the selected result'.`

const assistantInputFormat = `Checkmarx Infrastructure as Code Security has scanned this source code and reported the result.
This is the source code:
` + "```" + `
%s
` + "```" + `
and this is the result (vulnerability or security issue) found by Infrastructure as Code Security:
'%s' is detected in line %s with severity '%s'.`

const userInputFormat = `The user question is:
'<|IAC_QUESTION_START|>'
"%s"
'<|IAC_QUESTION_END|>'`

// dropLen number of messages to drop when limit is reached, 4 due to 2 from prompt, 1 from user question, 1 from reply
const dropLen = 4

const FileErrorFormat = "It seems that %s is not available for AI Guided Remediation. Please ensure that you have opened the correct workspace or the relevant file."

// chatModel model to use when calling the CheckmarxAI
const checkmarxAiChatModel = "GPT4"
const aiProxyRoute = "/api/ai-proxy"
const tenantIDClaimKey = "tenant_id"
const guidedRemediationFeatureNameKics = "cli-guided-remediation-kics"
const guidedRemediationFeatureNameSast = "cli-guided-remediation-sast"

type OutputModel struct {
	ConversationID string   `json:"conversationId"`
	Response       []string `json:"response"`
}

func ChatKicsSubCommand(chatWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	chatKicsCmd := &cobra.Command{
		Use:    "kics",
		Short:  "Chat about KICS result with OpenAI models",
		Long:   "Chat about KICS result with OpenAI models",
		Hidden: true,
		RunE:   runChatKics(chatWrapper, tenantWrapper),
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
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultFile)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultLine)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultSeverity)
	_ = chatKicsCmd.MarkFlagRequired(params.ChatKicsResultVulnerability)

	return chatKicsCmd
}

func runChatKics(
	chatKicsWrapper wrappers.ChatWrapper, tenantWrapper wrappers.TenantConfigurationWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		chatConversationID, _ := cmd.Flags().GetString(params.ChatConversationID)
		chatResultFile, _ := cmd.Flags().GetString(params.ChatKicsResultFile)
		chatResultLine, _ := cmd.Flags().GetString(params.ChatKicsResultLine)
		chatResultSeverity, _ := cmd.Flags().GetString(params.ChatKicsResultSeverity)
		chatResultVulnerability, _ := cmd.Flags().GetString(params.ChatKicsResultVulnerability)
		userInput, _ := cmd.Flags().GetString(params.ChatUserInput)

		var chatGptEnabled, azureAiEnabled, checkmarxAiEnabled bool
		var tenantConfigurationResponses *[]*wrappers.TenantConfigurationResponse

		if !isCxOneApiKeyAvailable() {
			chatGptEnabled = true
			azureAiEnabled = false
			checkmarxAiEnabled = false
			logger.Printf("CxOne API key is not available, ChatGPT model will be used for guided remediation.")
		} else {
			var err error
			tenantConfigurationResponses, err = GetTenantConfigurationResponses(tenantWrapper)
			if err != nil {
				return outputError(cmd, uuid.Nil, err)
			}

			azureAiEnabled = isAzureAiGuidedRemediationEnabled(tenantConfigurationResponses)
			checkmarxAiEnabled = isCheckmarxAiGuidedRemediationEnabled(tenantConfigurationResponses)
			chatGptEnabled = isChatGPTAiGuidedRemediationEnabled(tenantConfigurationResponses)
		}

		statefulWrapper, customerToken := CreateStatefulWrapper(cmd, azureAiEnabled, checkmarxAiEnabled, tenantConfigurationResponses)

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
		tenantID, _ := wrappers.ExtractFromTokenClaims(customerToken, tenantIDClaimKey)
		requestID := statefulWrapper.GenerateId().String()

		var response []message.Message
		if azureAiEnabled {
			azureAiEndPoint, _ := GetAzureAiEndPoint(tenantConfigurationResponses)
			metadata := message.MetaData{
				TenantID:  tenantID,
				RequestID: requestID,
				UserAgent: params.DefaultAgent,
				Feature:   guidedRemediationFeatureNameKics,
				ExternalModel: &message.ExternalAzure{
					Endpoint: azureAiEndPoint,
				},
			}
			logger.PrintIfVerbose("Sending message to Azure AI model for KICS guided remediation. RequestID: " + requestID)

			response, err = chatKicsWrapper.SecureCall(statefulWrapper, id, newMessages, &metadata, customerToken)
			if err != nil {
				return outputError(cmd, id, err)
			}
		} else if checkmarxAiEnabled {
			metadata := message.MetaData{
				TenantID:      tenantID,
				RequestID:     requestID,
				UserAgent:     params.DefaultAgent,
				Feature:       guidedRemediationFeatureNameKics,
				ExternalModel: nil,
			}
			logger.PrintIfVerbose("Sending message to Checkmarx AI model for KICS guided remediation. RequestID: " + requestID)
			response, err = chatKicsWrapper.SecureCall(statefulWrapper, id, newMessages, &metadata, customerToken)
			if err != nil {
				return outputError(cmd, id, err)
			}
		} else if chatGptEnabled {
			logger.PrintIfVerbose("Sending message to ChatGPT model for KICS guided remediation. RequestID: " + requestID)
			response, err = chatKicsWrapper.Call(statefulWrapper, id, newMessages)
			if err != nil {
				return outputError(cmd, id, err)
			}
		} else {
			return outputError(cmd, id, errors.New(AllOptionsDisabledError))
		}

		responseContent := getMessageContents(response)

		return printer.Print(cmd.OutOrStdout(), &OutputModel{
			ConversationID: id.String(),
			Response:       responseContent,
		}, printer.FormatJSON)
	}
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
