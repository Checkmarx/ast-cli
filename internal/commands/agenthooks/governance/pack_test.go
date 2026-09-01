package governance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovernancePack_Evaluate_AllowWhenNoPolicies(t *testing.T) {
	pack := &GovernancePack{PackVersion: 1, Policies: nil}
	ev := RuntimeEvent{Tool: ToolInfo{Category: "filesystem"}}
	d := pack.Evaluate(ev)
	assert.Equal(t, "ALLOW", d.Action)
	assert.Empty(t, d.PolicyID)
}

func TestGovernancePack_Evaluate_BlockOnPathMatch(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{
				PolicyID:    "pol_001",
				PolicyName:  "Block credential files",
				Version:     3,
				Type:        "filesystem",
				Action:      "BLOCK",
				MatchTarget: "path",
				Pattern:     `\.env$|\.pem$|\.key$`,
				AgentMessage: "Credential files are restricted.",
			},
		},
	}

	ev := RuntimeEvent{
		Tool: ToolInfo{
			Category: "filesystem",
			Name:     "Read",
			Paths:    []string{"C:/project/.env"},
		},
	}
	d := pack.Evaluate(ev)

	require.Equal(t, "BLOCK", d.Action)
	assert.Equal(t, "pol_001", d.PolicyID)
	assert.Equal(t, "Block credential files", d.PolicyName)
	assert.Equal(t, "Credential files are restricted.", d.AgentMessage)
	assert.Equal(t, "path", d.MatchedOn)
	assert.Equal(t, "C:/project/.env", d.MatchedValue)
	assert.Equal(t, 3, d.PolicyVersion)
	assert.Equal(t, 1, d.PackVersion)
}

func TestGovernancePack_Evaluate_AlertOnShellCommand(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 2,
		Policies: []PackPolicy{
			{
				PolicyID:    "pol_002",
				PolicyName:  "Alert on destructive shell",
				Version:     1,
				Type:        "shell",
				Action:      "ALERT",
				MatchTarget: "command",
				Pattern:     `rm\s+-rf`,
			},
		},
	}

	ev := RuntimeEvent{
		Tool: ToolInfo{
			Category: "shell",
			Name:     "Bash",
			Command:  "rm -rf ./tmp",
		},
	}
	d := pack.Evaluate(ev)

	assert.Equal(t, "ALERT", d.Action)
	assert.Equal(t, "pol_002", d.PolicyID)
	assert.Equal(t, "rm -rf ./tmp", d.MatchedValue)
}

func TestGovernancePack_Evaluate_AllowWhenCategoryMismatch(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{
				PolicyID: "pol_001", Type: "shell", Action: "BLOCK",
				MatchTarget: "command", Pattern: `rm -rf`,
			},
		},
	}
	// filesystem event — shell policy should not fire
	ev := RuntimeEvent{Tool: ToolInfo{Category: "filesystem", Paths: []string{"rm -rf"}}}
	d := pack.Evaluate(ev)
	assert.Equal(t, "ALLOW", d.Action)
}

func TestGovernancePack_Evaluate_BlockMCPTool(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{
				PolicyID:    "pol_003",
				PolicyName:  "Block GitHub MCP tools",
				Version:     2,
				Type:        "mcp",
				Action:      "BLOCK",
				MatchTarget: "mcp_tool",
				Pattern:     `^github/.*`,
			},
		},
	}
	ev := RuntimeEvent{
		Tool: ToolInfo{
			Category: "mcp",
			Name:     "create_issue",
			MCP:      &MCPInfo{Server: "github", Name: "create_issue"},
		},
	}
	d := pack.Evaluate(ev)
	assert.Equal(t, "BLOCK", d.Action)
	assert.Equal(t, "github/create_issue", d.MatchedValue)
}

func TestGovernancePack_Evaluate_AllowWhenNoPathMatch(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{PolicyID: "pol_001", Type: "filesystem", Action: "BLOCK", MatchTarget: "path", Pattern: `\.env$`},
		},
	}
	ev := RuntimeEvent{Tool: ToolInfo{Category: "filesystem", Paths: []string{"C:/project/main.go"}}}
	d := pack.Evaluate(ev)
	assert.Equal(t, "ALLOW", d.Action)
}

func TestGovernancePack_Evaluate_FirstPolicyWins(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{PolicyID: "pol_001", Type: "shell", Action: "BLOCK", MatchTarget: "command", Pattern: `rm`},
			{PolicyID: "pol_002", Type: "shell", Action: "ALERT", MatchTarget: "command", Pattern: `rm`},
		},
	}
	ev := RuntimeEvent{Tool: ToolInfo{Category: "shell", Command: "rm file.txt"}}
	d := pack.Evaluate(ev)
	assert.Equal(t, "BLOCK", d.Action)
	assert.Equal(t, "pol_001", d.PolicyID)
}

func TestGovernancePack_Evaluate_MalformedRegexSkipped(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{PolicyID: "pol_bad", Type: "shell", Action: "BLOCK", MatchTarget: "command", Pattern: `[invalid`},
			{PolicyID: "pol_ok", Type: "shell", Action: "ALERT", MatchTarget: "command", Pattern: `rm`},
		},
	}
	ev := RuntimeEvent{Tool: ToolInfo{Category: "shell", Command: "rm file.txt"}}
	d := pack.Evaluate(ev)
	// malformed regex is skipped; second policy fires
	assert.Equal(t, "ALERT", d.Action)
	assert.Equal(t, "pol_ok", d.PolicyID)
}

func TestDecision_Message_UsesAgentMessageWhenSet(t *testing.T) {
	d := Decision{PolicyName: "Block creds", AgentMessage: "Use the vault instead."}
	assert.Equal(t, "Use the vault instead.", d.Message())
}

func TestDecision_Message_FallsBackToPolicyName(t *testing.T) {
	d := Decision{PolicyName: "Block creds", AgentMessage: ""}
	assert.Equal(t, "Block creds", d.Message())
}

func TestGovernancePack_Evaluate_MultiplePathsFirstMatch(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{PolicyID: "pol_001", Type: "filesystem", Action: "BLOCK", MatchTarget: "path", Pattern: `\.env$`},
		},
	}
	ev := RuntimeEvent{
		Tool: ToolInfo{
			Category: "filesystem",
			Paths:    []string{"main.go", "config/.env", "README.md"},
		},
	}
	d := pack.Evaluate(ev)
	assert.Equal(t, "BLOCK", d.Action)
	assert.Equal(t, "config/.env", d.MatchedValue) // second path matched
}

func TestGovernancePack_Evaluate_PromptCategory(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{
				PolicyID:    "pol_prompt",
				PolicyName:  "Alert on any prompt",
				Version:     1,
				Type:        "prompt",
				Action:      "ALERT",
				MatchTarget: "prompt",
				Pattern:     `.*`, // any non-empty prompt
			},
		},
	}
	ev := RuntimeEvent{
		Kind:      "prompt.submit",
		Timestamp: time.Now(),
		Tool:      ToolInfo{Category: "prompt"},
		Prompt:    PromptInfo{Length: 42},
	}
	d := pack.Evaluate(ev)
	assert.Equal(t, "ALERT", d.Action)
}

func TestGovernancePack_Evaluate_PromptEmpty(t *testing.T) {
	pack := &GovernancePack{
		PackVersion: 1,
		Policies: []PackPolicy{
			{PolicyID: "pol_prompt", Type: "prompt", Action: "ALERT", MatchTarget: "prompt", Pattern: `.*`},
		},
	}
	ev := RuntimeEvent{
		Kind:   "prompt.submit",
		Tool:   ToolInfo{Category: "prompt"},
		Prompt: PromptInfo{Length: 0}, // empty prompt
	}
	d := pack.Evaluate(ev)
	assert.Equal(t, "ALLOW", d.Action) // no match on empty prompt
}
