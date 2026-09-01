package governance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTool_FilesystemRead(t *testing.T) {
	input := json.RawMessage(`{"file_path": "/home/user/main.go"}`)
	ti := normalizeTool("Read", input)
	assert.Equal(t, "filesystem", ti.Category)
	assert.Equal(t, "Read", ti.Name)
	assert.Equal(t, []string{"/home/user/main.go"}, ti.Paths)
	assert.Empty(t, ti.Command)
	assert.Nil(t, ti.MCP)
}

func TestNormalizeTool_FilesystemWrite(t *testing.T) {
	input := json.RawMessage(`{"file_path": "src/main.go"}`)
	ti := normalizeTool("Write", input)
	assert.Equal(t, "filesystem", ti.Category)
	assert.Equal(t, []string{"src/main.go"}, ti.Paths)
}

func TestNormalizeTool_FilesystemGlob(t *testing.T) {
	input := json.RawMessage(`{"pattern": "**/*.go"}`)
	ti := normalizeTool("Glob", input)
	assert.Equal(t, "filesystem", ti.Category)
	assert.Equal(t, []string{"**/*.go"}, ti.Paths)
}

func TestNormalizeTool_BashShell(t *testing.T) {
	input := json.RawMessage(`{"command": "rm -rf ./tmp", "restart": false}`)
	ti := normalizeTool("Bash", input)
	assert.Equal(t, "shell", ti.Category)
	assert.Equal(t, "Bash", ti.Name)
	assert.Equal(t, "rm -rf ./tmp", ti.Command)
	assert.Empty(t, ti.Paths)
}

func TestNormalizeTool_MCPTool(t *testing.T) {
	ti := normalizeTool("mcp__github__create_issue", json.RawMessage(`{}`))
	assert.Equal(t, "mcp", ti.Category)
	assert.Equal(t, "create_issue", ti.Name)
	assert.Equal(t, "mcp__github__create_issue", ti.RawName)
	require.NotNil(t, ti.MCP)
	assert.Equal(t, "github", ti.MCP.Server)
	assert.Equal(t, "create_issue", ti.MCP.Name)
}

func TestNormalizeTool_MCPToolNoDoubleUnderscore(t *testing.T) {
	ti := normalizeTool("mcp__stripe", json.RawMessage(`{}`))
	assert.Equal(t, "mcp", ti.Category)
	// Only one segment after prefix — server and tool are both the segment
	assert.Equal(t, "stripe", ti.MCP.Server)
}

func TestNormalizeTool_WebSearch(t *testing.T) {
	input := json.RawMessage(`{"query": "golang goroutine leak"}`)
	ti := normalizeTool("WebSearch", input)
	assert.Equal(t, "web_search", ti.Category)
	assert.Equal(t, "WebSearch", ti.Name)
	assert.Equal(t, "golang goroutine leak", ti.Query)
}

func TestNormalizeTool_WebFetch(t *testing.T) {
	input := json.RawMessage(`{"url": "https://example.com"}`)
	ti := normalizeTool("WebFetch", input)
	assert.Equal(t, "web_search", ti.Category)
	assert.Equal(t, "https://example.com", ti.Query)
}

func TestNormalizeTool_UnknownTool(t *testing.T) {
	ti := normalizeTool("SomeFutureTool", json.RawMessage(`{}`))
	assert.Equal(t, "other", ti.Category)
	assert.Equal(t, "SomeFutureTool", ti.Name)
	assert.Equal(t, "SomeFutureTool", ti.RawName)
}

func TestNormalizeTool_MultiEditFilesystem(t *testing.T) {
	ti := normalizeTool("MultiEdit", json.RawMessage(`{"file_path": "a.go"}`))
	assert.Equal(t, "filesystem", ti.Category)
}

func TestParseMCPName(t *testing.T) {
	tests := []struct {
		input      string
		wantServer string
		wantTool   string
	}{
		{"mcp__github__create_issue", "github", "create_issue"},
		{"mcp__stripe__charge_card", "stripe", "charge_card"},
		{"mcp__server__tool__extra", "server", "tool__extra"}, // extra segments stay in tool
		{"mcp__onlyone", "onlyone", "onlyone"},
	}
	for _, tt := range tests {
		server, tool := parseMCPName(tt.input)
		assert.Equal(t, tt.wantServer, server, "server for %s", tt.input)
		assert.Equal(t, tt.wantTool, tool, "tool for %s", tt.input)
	}
}

func TestExtractPaths_DirectoryKey(t *testing.T) {
	input := json.RawMessage(`{"directory": "/home/user"}`)
	paths := extractPaths(input)
	assert.Equal(t, []string{"/home/user"}, paths)
}

func TestExtractPaths_EmptyInput(t *testing.T) {
	paths := extractPaths(json.RawMessage(`{}`))
	assert.Nil(t, paths)
}

func TestRuntimeEvent_WithDecision_PromptIncluded(t *testing.T) {
	ev := RuntimeEvent{
		EventID:   "evt_abc",
		Kind:      "prompt.submit",
		SessionID: "sess_1",
		MachineID: "machine-1",
		UserID:    "alice",
		Agent:     AgentInfo{Type: "claude-code", ID: "agent-1"},
		Tool:      ToolInfo{Category: "prompt"},
		Prompt:    PromptInfo{Length: 99},
	}
	d := Decision{Action: "ALERT", PolicyID: "pol_1", PolicyName: "Alert prompts"}
	act := ev.WithDecision(d)

	assert.Equal(t, "evt_abc", act.EventID)
	assert.Equal(t, "prompt.submit", act.Kind)
	require.NotNil(t, act.Prompt)
	assert.Equal(t, 99, act.Prompt.Length)
	assert.Equal(t, "ALERT", act.Decision.Action)
}

func TestRuntimeEvent_WithDecision_ToolEventNoPrompt(t *testing.T) {
	ev := RuntimeEvent{
		Kind:  "tool.pre",
		Tool:  ToolInfo{Category: "filesystem"},
	}
	act := ev.WithDecision(Decision{Action: "ALLOW"})
	assert.Nil(t, act.Prompt) // prompt must not be set for tool events
}
