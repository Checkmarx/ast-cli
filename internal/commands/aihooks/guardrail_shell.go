package aihooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Shell guardrail — blocks commands that match the organization's blocklist.
// The blocklist is read from ~/.checkmarx/policyhooks1.json.
// If the file is missing or malformed the guardrail is fail-open.

const denyMessage = "\n\n" +
	"IMPORTANT: This action was blocked by your organization's security policy. " +
	"Do NOT attempt alternative commands, workarounds, or equivalent approaches to achieve the same result. " +
	"Any alternative method to accomplish this blocked action may be equally harmful to the system and the developer. " +
	"Instead, inform the user that this operation is restricted by organizational policy and suggest they contact their administrator if they need this action performed."

type shellPolicy struct {
	BlocklistTools struct {
		Enabled bool          `json:"enabled"`
		Tools   []blockedTool `json:"tools"`
	} `json:"blocklist_tools"`
}

type blockedTool struct {
	Name     string   `json:"name"`
	OS       []string `json:"os"`
	Category string   `json:"category"`
	Risk     string   `json:"risk"`
}

func shellPolicyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".checkmarx", "policyhooks1.json")
}

func loadBlockedCommands() map[string]blockedTool {
	blocked := map[string]blockedTool{}

	data, err := os.ReadFile(shellPolicyPath())
	if err != nil {
		return blocked
	}
	var policy shellPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return blocked
	}
	if !policy.BlocklistTools.Enabled {
		return blocked
	}

	currentOS := runtime.GOOS
	for _, t := range policy.BlocklistTools.Tools {
		if !matchesOS(t.OS, currentOS) {
			continue
		}
		blocked[strings.ToLower(t.Name)] = t
	}
	return blocked
}

func matchesOS(toolOS []string, currentOS string) bool {
	for _, o := range toolOS {
		mapped := o
		if o == "mac" {
			mapped = "darwin"
		}
		if mapped == currentOS {
			return true
		}
	}
	return false
}

func checkShellBlocklist(command string) string {
	blocked := loadBlockedCommands()
	cmdLower := strings.ToLower(command)

	for name, tool := range blocked {
		if strings.Contains(cmdLower, name) {
			return fmt.Sprintf(
				"Blocked by Checkmarx: command %q is not allowed.\nCategory: %s\nReason: %s%s",
				name, tool.Category, tool.Risk, denyMessage,
			)
		}
	}
	return ""
}

// Shell guardrail hook handler

func handleToolCallHook(route string, licensed bool) {
	var input toolCallHookInput
	if err := readHookInput(&input); err != nil {
		writeToolCallAllow(route)
		return
	}

	shellCmd := extractShellCommand(route, input)

	if shellCmd == "" || !licensed {
		writeToolCallAllow(route)
		return
	}

	if reason := checkShellBlocklist(shellCmd); reason != "" {
		writeToolCallDeny(route, reason)
		return
	}

	writeToolCallAllow(route)
}

func extractShellCommand(route string, input toolCallHookInput) string {
	switch route {
	case "claude-pre-tool-use", "droid-pre-tool-use":
		if input.ToolName == "Bash" {
			var v struct {
				Command string `json:"command"`
			}
			_ = json.Unmarshal(input.ToolInput, &v)
			return v.Command
		}
		return ""
	case "cursor-before-shell":
		return input.Command
	case "windsurf-pre-run-command":
		return input.ToolInfo.CommandLine
	case "gemini-before-tool":
		if input.ToolName == "execute_bash" || input.ToolName == "run_shell_command" {
			var v struct {
				Command string `json:"command"`
			}
			_ = json.Unmarshal(input.ToolInput, &v)
			return v.Command
		}
		return ""
	default:
		return ""
	}
}

func writeToolCallAllow(route string) {
	switch route {
	case "claude-pre-tool-use", "droid-pre-tool-use":
		writeHookOutput(map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":      "PreToolUse",
				"permissionDecision": "allow",
			},
		})
	case "cursor-before-shell", "cursor-before-mcp":
		writeHookOutput(map[string]any{"permission": "allow"})
	default:
		writeHookOutput(struct{}{})
	}
}

func writeToolCallDeny(route string, reason string) {
	switch route {
	case "claude-pre-tool-use":
		writeHookOutput(map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "deny",
				"permissionDecisionReason": reason,
			},
		})
	case "cursor-before-shell", "cursor-before-mcp":
		writeHookOutput(map[string]any{
			"permission":    "deny",
			"user_message":  reason,
			"agent_message": reason,
		})
	case "windsurf-pre-run-command", "windsurf-pre-mcp-tool-use", "droid-pre-tool-use":
		denyViaExit2(reason)
	case "gemini-before-tool":
		writeHookOutput(map[string]any{"decision": "deny", "reason": reason})
	}
}
