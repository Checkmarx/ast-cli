//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/google/uuid"
	"gotest.tools/assert"
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
	assert.Assert(t, strings.Contains(outputModel.Response[0], "Incorrect API key provided"), "Expecting incorrect api key error")
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
	assert.Assert(t, strings.Contains(outputModel.Response[0], "Incorrect API key provided"), "Expecting incorrect api key error")
}
