package commands

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestChatSastHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "chat", "sast")
}

func TestChatSastInvalidConversationId(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--conversation-id", "invalidId",
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--scan-results-file", "file",
		"--source-dir", "dir",
		"--sast-result-id", "resultId")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, fmt.Sprintf(ConversationIDErrorFormat, "invalidId")), s)
}

func TestChatSastNoUserInput(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--scan-results-file", "file",
		"--source-dir", "dir",
		"--sast-result-id", "resultId")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, fmt.Sprintf(UserInputRequiredErrorFormat, "user-input", "conversation-id")), s)
}

func TestChatSastInvalidScanResultsFile(t *testing.T) {
	const ScanResultsFileErrorFormat = "Error reading and parsing SAST results file '%s'"

	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--scan-results-file", "invalidFile",
		"--source-dir", "dir",
		"--sast-result-id", "resultId")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, fmt.Sprintf(ScanResultsFileErrorFormat, "invalidFile")), s)
}

func TestChatSastInvalideResultId(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "dir",
		"--sast-result-id", "invalidResultId")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, "result ID invalidResultId not found"), s)
}

func TestChatSastAiGuidedRemediationDisabled(t *testing.T) {
	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{{
		Key:   "scan.config.plugins.ideScans",
		Value: "true",
	},
		{
			Key:   "scan.config.plugins.aiGuidedRemediation",
			Value: "false",
		},
	}

	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "dir",
		"--sast-result-id", "13588362")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, AiGuidedRemediationDisabledError), s)
	mock.TenantConfiguration = []*wrappers.TenantConfigurationResponse{}
}

func TestChatSastInvalidSourceDir(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "invalidDir",
		"--sast-result-id", "13588362")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, "open invalidDir"), s)
}

func TestChatSastFirstMessageCorrectResponse(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "./data",
		"--sast-result-id", "13588362")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))
	assert.Assert(t, strings.Contains(s, "mock"), s)
}

func TestChatSastSecondMessageCorrectResponse(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--chat-apikey", "apiKey",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "./data",
		"--sast-result-id", "13588362",
		"--conversation-id", uuid.New().String(),
		"--user-input", "userInput")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))
	assert.Assert(t, strings.Contains(s, "mock"), s)
}
