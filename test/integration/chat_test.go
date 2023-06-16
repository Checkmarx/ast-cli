package integration

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestChatInvalidId(t *testing.T) {
	args := []string{"chat",
		flag(params.ChatConversationId), "invalidId",
		flag(params.ChatApiKey), "apiKey",
		flag(params.ChatUserInput), "userInput",
		flag(params.ChatResultFile), "file",
		flag(params.ChatResultLine), "0",
		flag(params.ChatResultSeverity), "LOW",
		flag(params.ChatResultVulnerability), "Vulnerability",
	}

	buffer := executeCmdNilAssertion(t, "Calling ChatGPT should pass", args...)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, s == fmt.Sprintf(commands.ConversationIdErrorFormat, "invalidId"))
}

func TestChatInvalidFile(t *testing.T) {
	args := []string{"chat",
		flag(params.ChatApiKey), "apiKey",
		flag(params.ChatUserInput), "userInput",
		flag(params.ChatResultFile), "invalidfile",
		flag(params.ChatResultLine), "0",
		flag(params.ChatResultSeverity), "LOW",
		flag(params.ChatResultVulnerability), "Vulnerability",
	}

	buffer := executeCmdNilAssertion(t, "Calling ChatGPT should pass", args...)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, s == fmt.Sprintf(commands.FileErrorFormat, "invalidfile"))
}

func TestChatInvalidApiKey(t *testing.T) {
	args := []string{"chat",
		flag(params.ChatApiKey), "apiKey",
		flag(params.ChatUserInput), "userInput",
		flag(params.ChatResultFile), "./data/Dockerfile",
		flag(params.ChatResultLine), "0",
		flag(params.ChatResultSeverity), "LOW",
		flag(params.ChatResultVulnerability), "Vulnerability",
	}

	buffer := executeCmdNilAssertion(t, "Calling ChatGPT should pass", args...)
	output, err := io.ReadAll(buffer)
	assert.NilError(t, err)
	s := string(output)
	assert.Assert(t, strings.Contains(s, "api_key"), s)
}

func TestChatCorrectResponse(t *testing.T) {
	args := []string{"chat",
		flag(params.ChatApiKey), viper.GetString("CHAT_APIKEY"),
		flag(params.ChatUserInput), "Explain the result.",
		flag(params.ChatResultFile), "./data/Dockerfile",
		flag(params.ChatResultLine), "0",
		flag(params.ChatResultSeverity), "LOW",
		flag(params.ChatResultVulnerability), "Vulnerability",
	}
	_ = executeCmdNilAssertion(t, "Calling ChatGPT with correct API Key should pass", args...)
}
