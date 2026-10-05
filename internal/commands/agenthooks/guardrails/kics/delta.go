package kics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/agentprofile"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
)

// findingKey is the deduplication tuple used for delta detection.
// Mirrors the ignore-file key used by RunIacRealtimeScan: Title + "_" + SimilarityID.
type findingKey struct {
	title        string
	similarityID string
}

func keyOf(r iacrealtime.IacRealtimeResult) findingKey {
	return findingKey{
		title:        r.Title,
		similarityID: r.SimilarityID,
	}
}

// NewFindings returns results present in newScan that have no matching key in originalScan.
// A new file (originalScan == nil) returns newScan unchanged — any finding is "new".
func NewFindings(originalScan, newScan []iacrealtime.IacRealtimeResult) []iacrealtime.IacRealtimeResult {
	if originalScan == nil {
		return newScan
	}
	baseline := make(map[findingKey]struct{}, len(originalScan))
	for _, r := range originalScan {
		baseline[keyOf(r)] = struct{}{}
	}
	var out []iacrealtime.IacRealtimeResult
	for _, r := range newScan {
		if _, exists := baseline[keyOf(r)]; !exists {
			out = append(out, r)
		}
	}
	return out
}

// findingsSummary returns the bullet list of findings for human display.
func findingsSummary(filePath string, findings []iacrealtime.IacRealtimeResult) string {
	var sb strings.Builder
	for _, f := range findings {
		line := 0
		if len(f.Locations) > 0 {
			line = f.Locations[0].Line
		}
		description := f.Description
		if description == "" {
			description = "No description provided"
		}
		fmt.Fprintf(&sb, "  - %s line %d [%s] %s (similarity_id %s) — %s\n",
			filePath, line, f.Severity, f.Title, f.SimilarityID, description)
	}
	return sb.String()
}

// formatFindings builds the two verdict fields delivered to the agent.
// Cursor receives cursorAdditionalContext (folded into agent_message); other agents
// (including Gemini) receive additionalContext, with MCP tool names adjusted per agent.
func formatFindings(filePath string, findings []iacrealtime.IacRealtimeResult, agent agenthooks.AgentID, workDir, sessionID string) (reason, context string) {
	summary := findingsSummary(filePath, findings)
	cxBinary := cxExecutable()
	reason = permissionDecisionReason(filePath, summary)
	switch agent {
	case agenthooks.AgentCursor:
		context = cursorAdditionalContext(filePath, cxBinary, findings, workDir, sessionID)
	default:
		context = additionalContext(filePath, cxBinary, findings, workDir, agent, sessionID)
	}
	return reason, context
}

func cxExecutable() string {
	cxExe, err := os.Executable()
	if err != nil {
		return "cx"
	}
	return cxExe
}

// ignoredFilePathFlag returns the " --ignored-file-path '<path>'" fragment that pins the
// suppression command to the workspace ignore file anchored at workDir.
func ignoredFilePathFlag(workDir string) string {
	if workDir == "" {
		return ""
	}
	return fmt.Sprintf(" --ignored-file-path '%s'", ignore.PathFor(workDir))
}

// cursorIgnoredFilePathFlag is the Cursor-specific variant of ignoredFilePathFlag.
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

func kicsSuppressCommands(cxBinary string, findings []iacrealtime.IacRealtimeResult, workDir string, agent agenthooks.AgentID, sessionID string) string {
	provenance := optionalFlagsFragment(agent, sessionID)
	var suppressCmds strings.Builder
	for i := range findings {
		f := &findings[i]
		data, _ := json.Marshal(iacrealtime.IgnoredIacFinding{
			Title:        f.Title,
			SimilarityID: f.SimilarityID,
		})
		if agent == agenthooks.AgentCursor {
			ignoreFlag := cursorIgnoredFilePathFlag(workDir)
			suppressCmds.WriteString(cursorplugin.IgnoreVulnerabilityCommand(cxBinary, "iac", data, ignoreFlag, provenance))
			suppressCmds.WriteString("\n")
			continue
		}
		ignoreFlag := ignoredFilePathFlag(workDir)
		if agent == agenthooks.AgentGemini {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type iac --data %s%s%s\n", cxBinary, ignore.QuoteDataFlag(data), ignoreFlag, provenance)
		} else {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type iac --data '%s'%s%s\n", cxBinary, string(data), ignoreFlag, provenance)
		}
	}
	return suppressCmds.String()
}

// permissionDecisionReason is the human-readable deny message shown to the user.
func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"KICS security scan detected IaC vulnerabilities in %s.\nFindings:\n%s",
		filePath, summary,
	)
}

// additionalContext is injected into the agent's context window to drive remediation.
// Contains all action instructions — not shown directly to the user on Claude; on Gemini
// BeforeTool it is folded into the hook deny reason by the ast-cx-hooks gemini adapter.
// Used for Claude, Copilot, Gemini, and other non-Cursor agents. Gemini suppress commands
// use ignore.QuoteDataFlag (PowerShell-safe quoting on Windows); other agents keep the
// original single-quoted JSON payload.
func additionalContext(filePath, cxBinary string, findings []iacrealtime.IacRealtimeResult, workDir string, agent agenthooks.AgentID, sessionID string) string {
	codeTool := "mcp__Checkmarx__codeRemediation"
	if agent == agenthooks.AgentGemini {
		codeTool = "mcp_Checkmarx_codeRemediation"
	}
	skill := kicsSkillName(agent)
	suppressCmds := kicsSuppressCommands(cxBinary, findings, workDir, agent, sessionID)
	return fmt.Sprintf(
		"KICS detected IaC misconfigurations in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command.\n"+
			"Action required:\n"+
			"  1. For each finding, invoke the %s skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, %s\n"+
			"     This is the only supported remediation path — do not attempt manual configuration changes.\n"+
			"  3. If %s is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. If the user accepts the risk for a finding, suppress it by running the corresponding command\n"+
			"     below, then retry the write:\n%s",
		filePath, skill, remediationInstructions(codeTool),
		codeTool, agentprofile.McpReconnect(agentLabel(agent)), suppressCmds,
	)
}

// remediationInstructions returns MCP tool-call guidance for KICS IaC findings.
// All findings from this guardrail come from RunIacRealtimeScan (KICS), so they
// always use codeRemediation with type "iac" — including Dockerfile findings.
func remediationInstructions(codeTool string) string {
	return fmt.Sprintf("For each finding, call the %s tool with:\n"+
		"  {\n"+
		"    \"type\": \"iac\",\n"+
		"    \"metadata\": {\n"+
		"      \"title\": \"[Title from finding]\",\n"+
		"      \"description\": \"[Description from finding]\",\n"+
		"      \"remediationAdvice\": \"[how to harden this configuration]\"\n"+
		"    }\n"+
		"  }\n"+
		"Apply the remediation guidance the tool returns, then retry the write. If a fix "+
		"genuinely requires resources outside this file (for example a separate KMS key or "+
		"a centrally-managed policy), add them as part of your change rather than skipping "+
		"the finding.", codeTool)
}

// cursorAdditionalContext is remediation guidance for Cursor only. Uses the plugin-prefixed MCP
// tool name and PowerShell --% stop-parsing for suppress commands on Windows.
func cursorAdditionalContext(filePath, cxBinary string, findings []iacrealtime.IacRealtimeResult, workDir, sessionID string) string {
	suppressCmds := kicsSuppressCommands(cxBinary, findings, workDir, agenthooks.AgentCursor, sessionID)
	skill := kicsSkillName(agenthooks.AgentCursor)
	return fmt.Sprintf(
		"KICS detected IaC misconfigurations in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command. "+
			"ANALYZE each finding to determine if it is a real misconfiguration or a false positive "+
			"(for example an acceptable deviation for this environment or platform). "+
			"Follow the cx-hook-deny.mdc rule for this deny. "+
			"ASK THE USER FIRST, for every real finding, before taking any action: \"An IaC misconfiguration "+
			"was detected. Would you like to remediate it (harden the configuration via MCP) "+
			"or suppress it (mark as a confirmed false positive and unblock the write)?\" and wait for "+
			"their answer. Do not decide this yourself — an intentionally-inserted misconfiguration (e.g. "+
			"in a lab/demo/training file the user asked for on purpose) is NOT the same as a confirmed "+
			"false positive: suppress only on the user's explicit instruction, never because the "+
			"request seems intentional. "+
			"Apply the cx-devassist-kics.mdc rule: for each finding the user asks you to remediate, "+
			"invoke the %s skill exactly as written — do not skip, abbreviate, or reimplement its steps "+
			"inline. The findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation; the skill also handles MCP unavailability and self-recovery. "+
			"Always show its Step 4 IaC Remediation Summary to the user verbatim when done. "+
			"Do not retry the blocked Write/StrReplace, paste code in chat, or bypass the scan with shell workarounds. "+
			"If that skill is not available in this session, %s\n"+
			"If the user chooses to suppress a finding, run the corresponding command below, then retry the write:\n%s",
		filePath, skill, remediationInstructions(cursorplugin.MCPTool("codeRemediation")),
		suppressCmds,
	)
}

// kicsSkillName returns the agent-specific skill invocation string for KICS remediation.
func kicsSkillName(agent agenthooks.AgentID) string {
	if agent == agenthooks.AgentGemini {
		return "/cx-devassist-kics"
	}
	return "cx-devassist:cx-devassist-kics"
}
