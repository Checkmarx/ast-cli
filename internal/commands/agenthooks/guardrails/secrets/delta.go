package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
)

// findingKey matches the ignore-file key used by secrets-realtime: Title + SecretValue.
type findingKey struct {
	title string
	value string
}

func keyOf(r secretsrealtime.SecretsRealtimeResult) findingKey {
	return findingKey{title: r.Title, value: r.SecretValue}
}

// NewFindings returns results present in newScan that have no matching key in originalScan.
// A new file (originalScan == nil) returns newScan unchanged.
func NewFindings(originalScan, newScan []secretsrealtime.SecretsRealtimeResult) []secretsrealtime.SecretsRealtimeResult {
	if originalScan == nil {
		return newScan
	}
	baseline := make(map[findingKey]struct{}, len(originalScan))
	for _, r := range originalScan {
		baseline[keyOf(r)] = struct{}{}
	}
	var out []secretsrealtime.SecretsRealtimeResult
	for _, r := range newScan {
		if _, exists := baseline[keyOf(r)]; !exists {
			out = append(out, r)
		}
	}
	return out
}

// formatFindings builds the deny reason and the remediation context delivered to the agent.
// The secret value is written only to a mode-0600 finding file referenced by the suppress
// command, so it does not appear in the verdict text.
func formatFindings(filePath string, findings []secretsrealtime.SecretsRealtimeResult, agent agenthooks.AgentID, workDir, sessionID string) (reason, context string) {
	summary := findingsSummary(filePath, findings)
	reason = permissionDecisionReason(filePath, summary)
	if agent == agenthooks.AgentCursor {
		context = cursorAdditionalContext(filePath, cxExecutable(), findings, workDir, sessionID)
	} else {
		context = additionalContext(filePath, cxExecutable(), findings, workDir, agent, sessionID)
	}
	return reason, context
}

func findingsSummary(filePath string, findings []secretsrealtime.SecretsRealtimeResult) string {
	var sb strings.Builder
	for _, f := range findings {
		line := 0
		if len(f.Locations) > 0 {
			line = f.Locations[0].Line
		}
		fmt.Fprintf(&sb, "  - %s line %d [%s] %s — %s\n",
			filePath, line, f.Severity, f.Title, safeDescription(f))
	}
	return sb.String()
}

func safeDescription(f secretsrealtime.SecretsRealtimeResult) string {
	if f.Description == "" || (f.SecretValue != "" && strings.Contains(f.Description, f.SecretValue)) {
		return "hardcoded secret"
	}
	return f.Description
}

func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"Checkmarx secret detection found hardcoded secrets in %s.\nFindings:\n%s",
		filePath, summary,
	)
}

func cxExecutable() string {
	cxExe, err := os.Executable()
	if err != nil {
		return "cx"
	}
	return cxExe
}

func ignoredFilePathFlag(workDir string) string {
	if workDir == "" {
		return ""
	}
	return fmt.Sprintf(" --ignored-file-path '%s'", ignore.PathFor(workDir))
}

func cursorIgnoredFilePathFlag(workDir string) string {
	if workDir == "" {
		return ""
	}
	p := filepath.ToSlash(ignore.PathFor(workDir))
	return fmt.Sprintf(" --ignored-file-path %q", p)
}

func optionalFlagsFragment(agent agenthooks.AgentID, sessionID string) string {
	label := agentLabel(agent)
	if label == "" {
		return ""
	}
	pairs := "aiProvider=" + label + ";agent=" + label + "-cli"
	if sessionID != "" {
		pairs += ";aiAgentSessionId=" + sessionID
	}
	return fmt.Sprintf(" --optional-flags %q", pairs)
}

func agentLabel(agent agenthooks.AgentID) string {
	switch agent {
	case agenthooks.AgentClaude:
		return "Claude"
	case agenthooks.AgentCopilot, agenthooks.AgentCopilotCLI:
		return "Copilot"
	case agenthooks.AgentCursor:
		return "Cursor"
	case agenthooks.AgentGemini:
		return "Gemini"
	case agenthooks.AgentCodex:
		return "Codex"
	default:
		return ""
	}
}

// secretsSuppressCommands returns one ignore-vulnerability command per finding.
// The finding JSON (which must include SecretValue) is stored in a temp file and passed as
// --data @<file> so the secret value is not embedded in the agent-visible command.
func secretsSuppressCommands(cxBinary string, findings []secretsrealtime.SecretsRealtimeResult, workDir string, agent agenthooks.AgentID, sessionID string) string {
	provenance := optionalFlagsFragment(agent, sessionID)
	var suppressCmds strings.Builder
	for i := range findings {
		f := &findings[i]
		path, err := writeSuppressPayload(*f)
		if err != nil {
			fmt.Fprintf(&suppressCmds, "  # suppress unavailable for %s\n", f.Title)
			continue
		}
		dataArg := "@" + path
		if agent == agenthooks.AgentCursor {
			ignoreFlag := cursorIgnoredFilePathFlag(workDir)
			suppressCmds.WriteString(cursorplugin.IgnoreVulnerabilityCommand(cxBinary, "secrets", []byte(dataArg), ignoreFlag, provenance))
			suppressCmds.WriteString("\n")
			continue
		}
		ignoreFlag := ignoredFilePathFlag(workDir)
		if agent == agenthooks.AgentGemini {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type secrets --data %s%s%s\n", cxBinary, ignore.QuoteDataFlag([]byte(dataArg)), ignoreFlag, provenance)
		} else {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type secrets --data '%s'%s%s\n", cxBinary, dataArg, ignoreFlag, provenance)
		}
	}
	return suppressCmds.String()
}

func writeSuppressPayload(f secretsrealtime.SecretsRealtimeResult) (string, error) {
	if f.Title == "" || f.SecretValue == "" {
		return "", fmt.Errorf("finding missing ignore key")
	}
	payload, err := json.Marshal(secretsrealtime.IgnoredSecret{Title: f.Title, SecretValue: f.SecretValue})
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "cx-secret-finding-*.json")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if _, err = file.Write(payload); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
