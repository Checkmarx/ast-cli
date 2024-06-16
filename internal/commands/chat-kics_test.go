package commands

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestChatKicsHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "chat", "kics")
}

func TestChatKicsInvalidId(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "kics",
		"--conversation-id", "invalidId",
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--result-file", "file",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, fmt.Sprintf(ConversationIDErrorFormat, "invalidId")), s)
}

func TestChatKicsInvalidFile(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--result-file", "invalidfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, fmt.Sprintf(FileErrorFormat, "invalidfile")), s)
}

func TestChatKicsCorrectResponse(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--user-input", "userInput",
		"--result-file", "./data/Dockerfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))
	assert.Assert(t, strings.Contains(s, "mock"), s)
}

func TestChatKicsAzureAICorrectResponse(t *testing.T) {
	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{
		{
			Key:   "scan.config.plugins.ideScans",
			Value: "true",
		},
		{
			Key:   "scan.config.plugins.azureAiGuidedRemediation",
			Value: "true",
		},
	}
	origAPIKey := viper.GetString(params.AstAPIKey)
	viper.Set(params.AstAPIKey, "SomeKey")

	buffer, err := executeRedirectedTestCommand("chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--user-input", "userInput",
		"--result-file", "./data/Dockerfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))

	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{}
	viper.Set(params.AstAPIKey, origAPIKey)

	assert.Assert(t, strings.Contains(s, "mock message from securecall with externalmodel: externalmodel is not nil"), s)
}

func TestChatKicsCheckmarxAICorrectResponse(t *testing.T) {
	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{
		{
			Key:   "scan.config.plugins.ideScans",
			Value: "true",
		},
		{
			Key:   "scan.config.plugins.checkmarxAiGuidedRemediation",
			Value: "true",
		},
	}
	origAPIKey := viper.GetString(params.AstAPIKey)
	viper.Set(params.AstAPIKey, "SomeKey")

	buffer, err := executeRedirectedTestCommand("chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--result-file", "./data/Dockerfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))

	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{}
	viper.Set(params.AstAPIKey, origAPIKey)

	assert.Assert(t, strings.Contains(s, "mock message from securecall with externalmodel: externalmodel is nil"), s)
}
