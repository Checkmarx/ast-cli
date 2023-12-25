package commands

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestChatSastHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "chat", "sast")
}

func TestChatSastInvalidId(t *testing.T) {
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

func TestChatSastInvalidScanResultsFile(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
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
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "dir",
		"--sast-result-id", "invalidResultId")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, "result ID invalidResultId not found"), s)
}

func TestChatSastInvalidSourceDir(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--conversation-id", uuid.New().String(),
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

func TestChatSastCorrectResponse(t *testing.T) {
	buffer, err := executeRedirectedTestCommand("chat", "sast",
		"--conversation-id", uuid.New().String(),
		"--chat-apikey", "apiKey",
		"--user-input", "userInput",
		"--scan-results-file", "./data/cx_result.json",
		"--source-dir", "./data",
		"--sast-result-id", "13588362")
	assert.NilError(t, err)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := strings.ToLower(string(output))
	assert.Assert(t, strings.Contains(s, "mock"), s)
}
