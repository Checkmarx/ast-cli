//go:build integration

package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

const (
	INCORRECT_API_ERROR = "Error Code: 401, Access denied due to invalid subscription key or wrong API endpoint"
)

func TestChatKicsInvalidAPIKey(t *testing.T) {
	args := []string{
		"chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "invalidApiKey",
		"--user-input", "userInput",
		"--result-file", "./data/Dockerfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability",
	}
	err, respBuffer := executeCommand(t, args...)
	assert.NilError(t, err)
	outputModel := commands.OutputModel{}
	unmarshall(t, respBuffer, &outputModel, "Reading results should pass")
	assert.Assert(t, strings.Contains(outputModel.Response[0], "Incorrect API key provided"), "Expecting incorrect api key error. Got: "+outputModel.Response[0])
}

func TestChatSastInvalidAPIKey(t *testing.T) {
	args := []string{
		"chat", "sast",
		"--chat-apikey", "invalidApiKey",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "./data",
		"--sast-result-id", "13588362",
	}
	err, respBuffer := executeCommand(t, args...)
	assert.NilError(t, err)
	outputModel := commands.OutputModel{}
	unmarshall(t, respBuffer, &outputModel, "Reading results should pass")
	assert.Assert(t, strings.Contains(outputModel.Response[0], "Incorrect API key provided"), "Expecting incorrect api key error. Got: "+outputModel.Response[0])
}

func TestChatKicsAzureAIInvalidAPIKey(t *testing.T) {
	createASTIntegrationTestCommand(t)
	mockConfig := []*wrappers.TenantConfigurationResponse{
		{
			Key:   "scan.config.plugins.ideScans",
			Value: "true",
		},
		{
			Key:   "scan.config.plugins.azureAiGuidedRemediation",
			Value: "true",
		},
		{
			Key:   "scan.config.plugins.aiGuidedRemediationAiEngine",
			Value: "azureai",
		},
	}

	mockTenant := mock.TenantConfigurationMockWrapper{}
	mockTenant.SetTenantConfiguration(mockConfig)

	args := []string{
		"chat", "kics",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "invalidApiKey",
		"--user-input", "userInput",
		"--result-file", "./data/Dockerfile",
		"--result-line", "0",
		"--result-severity", "LOW",
		"--result-vulnerability", "Vulnerability",
	}

	response := RunKicsChatForTest(t, mockTenant, args...)
	assert.Assert(t, strings.Contains(response.Response[0], INCORRECT_API_ERROR), "Expecting incorrect api key error. Got: "+response.Response[0])

}

func RunKicsChatForTest(t *testing.T, tenantWrapper mock.TenantConfigurationMockWrapper, args ...string) commands.OutputModel {
	outputBuffer := bytes.NewBufferString("")
	cmd := commands.ChatKicsSubCommand(wrappers.NewChatWrapper(), tenantWrapper)
	cmd.SetArgs(args)
	cmd.SetOut(outputBuffer)
	err := cmd.Execute()
	assert.NilError(t, err)
	outputModel := commands.OutputModel{}
	unmarshall(t, outputBuffer, &outputModel, "Reading results should pass")
	return outputModel
}
