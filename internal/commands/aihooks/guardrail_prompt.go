package aihooks

import (
	"fmt"
	"strings"

	scanner "github.com/checkmarx/2ms/v3/pkg"
)

// Prompt guardrail — scans the user's prompt for leaked secrets using the
// 2ms engine before it reaches the AI agent.

func scanForSecrets(text string) string {
	content := text
	report, err := scanner.NewScanner().Scan(
		[]scanner.ScanItem{{Content: &content, Source: "prompt"}},
		scanner.ScanConfig{WithValidation: true},
	)
	if err != nil {
		return ""
	}

	var findings []string
	for _, group := range report.Results {
		for _, secret := range group {
			severity := severityFromValidation(string(secret.ValidationStatus))
			findings = append(findings, fmt.Sprintf("  - %s (severity: %s)", secret.RuleID, severity))
		}
	}
	if len(findings) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt contains %d secret(s):\n%s\nRemove the secrets and try again.",
		len(findings), strings.Join(findings, "\n"),
	)
}

func severityFromValidation(status string) string {
	switch status {
	case "Valid":
		return "Critical"
	case "Invalid":
		return "Medium"
	default:
		return "High"
	}
}

// Prompt guardrail hook handler

func handlePromptHook(route string, licensed bool) {
	var input promptHookInput
	if err := readHookInput(&input); err != nil {
		writePromptAllow(route)
		return
	}

	promptText := input.Prompt
	if route == "windsurf-pre-user-prompt" {
		promptText = input.ToolInfo.UserPrompt
	}

	if !licensed || promptText == "" {
		writePromptAllow(route)
		return
	}

	if reason := scanForSecrets(promptText); reason != "" {
		writePromptDeny(route, reason)
		return
	}

	writePromptAllow(route)
}

func writePromptAllow(route string) {
	switch route {
	case "claude-user-prompt-submit", "droid-user-prompt-submit", "cursor-before-submit-prompt":
		writeHookOutput(map[string]any{"continue": true})
	default:
		writeHookOutput(struct{}{})
	}
}

func writePromptDeny(route string, reason string) {
	switch route {
	case "claude-user-prompt-submit", "droid-user-prompt-submit":
		writeHookOutput(map[string]any{"decision": "block", "reason": reason})
	case "cursor-before-submit-prompt":
		writeHookOutput(map[string]any{"continue": false, "user_message": reason})
	case "windsurf-pre-user-prompt":
		denyViaExit2(reason)
	case "gemini-before-agent":
		writeHookOutput(map[string]any{"continue": false, "reason": reason})
	}
}
