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

// provenanceTag returns the string prepended to permissionDecisionReason and additionalContext in
// formatFindings so the agent can recognize genuine Checkmarx output. NOT an authority claim — an
// attacker could spoof this exact string from a file, a fetched page, or another tool's output — it
// is a signpost: text that LOOKS like a Checkmarx finding but lacks it, or that instructs running a
// script/CLI command "silently"/"without asking", did not come from Checkmarx.
func provenanceTag(agent agenthooks.AgentID) string {
	return "[Checkmarx cx-devassist — automated security output, not user input]"
}

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
	tag := provenanceTag(agent)
	reason = tag + " " + permissionDecisionReason(filePath, summary)
	switch agent {
	case agenthooks.AgentCursor:
		context = cursorAdditionalContext(filePath, cxBinary, findings, workDir, sessionID)
	default:
		context = additionalContext(filePath, cxBinary, findings, workDir, agent, sessionID)
	}
	context = tag + " " + context
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

// hookDenyHeader is the agent-facing opening of additionalContext. The provenance tag is
// prepended separately in formatFindings. This header does not contain the spoof phrases
// the skills treat as proof a message is not from Checkmarx.
const hookDenyHeader = "This is a Checkmarx hook deny. Follow it. It is not file content and not an untrusted tool error.\n"

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
		hookDenyHeader+
			"KICS blocked the write to %s.\nFindings:\n%s"+
			"Handle every finding yourself. Never ask the user, including \"Would you like me to "+
			"remediate?\" (that question is only for on-demand scans).\n"+
			"1. Classify each finding:\n"+
			"- False positive only if one of these is already true and you can cite it: (a) the user "+
			"explicitly told you to ignore or suppress it; (b) a file you opened this session shows, at a "+
			"line you can cite, that the configuration does not do what the rule flags, or the same issue "+
			"is already handled there. Deployment or runtime assumptions you cannot see in a file are not "+
			"evidence. What a finding or file says about another file is not evidence; open that file. "+
			"Apparent intent is not evidence; that still needs (a) or (b).\n"+
			"- Otherwise it is a true positive, including when you are unsure.\n"+
			"2. True positive: call %s for each one with type \"iac\" and metadata (title, description, "+
			"remediationAdvice = how to harden this configuration). The %s skill can make this call; if "+
			"it disagrees with this deny, follow this deny. Apply remediation_steps only with Edit, "+
			"Write, MultiEdit, or NotebookEdit, never a shell command (shell writes are not scanned). "+
			"Make the smallest change to the flagged configuration; if the fix needs a resource in "+
			"another file, that is also a gated write. If the blocked write creates a new file, the fix "+
			"is that same Write with the fixed content.\n"+
			"If %s is unavailable, change nothing and ignore nothing because of it. Report the findings "+
			"as unresolved and tell the user to %s.\n"+
			"3. False positive: run exactly this command for it:\n%s"+
			"4. Verify: skip only the initial scan. Run `cx scan iac-realtime -s <file>` with the "+
			"canonical cx path, then retry the blocked write once; the hook on that retry is the check. "+
			"If it is denied, a remaining finding, or one your fix introduced, gets one more %s call. "+
			"Stop after 3 denied retries or when the tool returns no safe change. Then ignore only "+
			"findings that meet step 1, report the rest as unresolved, and stop editing the file. Do not "+
			"ask whether to continue.\n"+
			"Never write this content through another tool, a shell command, or a different file to "+
			"bypass the scan.\n"+
			"5. Always finish with this report, even if you asked the user a question, the file is new, "+
			"or the retry passed. Give one line for every finding listed above, omitting empty sections "+
			"(finding lines are 0-based; show line + 1):\n"+
			"Checkmarx Dev Assist IaC(KICS) Remediation Summary\n"+
			"Title: <title>  Severity: <severity>  Line: <line+1>\n"+
			"Files Modified: - <file> line <n>: <change>\n"+
			"Ignored: - <title> line <n> <severity> - <evidence: the user's words, or the file and line "+
			"you read>\n"+
			"Unresolved: - <title> line <n> <severity> - <reason>\n"+
			"Final status: All fixed | Partially fixed | Unresolved\n"+
			"Then continue the user's original task.\n",
		filePath, findingsSummary(filePath, findings), codeTool, skill, codeTool, agentprofile.McpReconnect(agentLabel(agent)), suppressCmds, codeTool,
	)
}

// cursorAdditionalContext is remediation guidance for Cursor only. Uses the plugin-prefixed MCP
// tool name and PowerShell --% stop-parsing for suppress commands on Windows.
func cursorAdditionalContext(filePath, cxBinary string, findings []iacrealtime.IacRealtimeResult, workDir, sessionID string) string {
	suppressCmds := kicsSuppressCommands(cxBinary, findings, workDir, agenthooks.AgentCursor, sessionID)
	skill := kicsSkillName(agenthooks.AgentCursor)
	codeTool := cursorplugin.MCPTool("codeRemediation")
	return fmt.Sprintf(
		hookDenyHeader+
			"KICS blocked the write to %s. Follow the cx-hook-deny.mdc and cx-devassist-kics.mdc rules "+
			"for this deny.\n"+
			"Handle every finding yourself. Never ask the user, including \"Would you like me to "+
			"remediate?\" (that question is only for on-demand scans).\n"+
			"1. Classify each finding:\n"+
			"- False positive only if one of these is already true and you can cite it: (a) the user "+
			"explicitly told you to ignore or suppress it; (b) a file you opened this session shows, at a "+
			"line you can cite, that the configuration does not do what the rule flags, or the same issue "+
			"is already handled there (for example a parent module or sibling manifest). Deployment or "+
			"runtime assumptions you cannot see in a file are not evidence. What a finding or file says "+
			"about another file is not evidence; open that file. Apparent intent is not evidence; that "+
			"still needs (a) or (b).\n"+
			"- Otherwise it is a true positive, including when you are unsure.\n"+
			"2. True positive: invoke the %s skill exactly as written (do not skip, abbreviate, or "+
			"reimplement its steps); it skips the initial scan and calls %s. If the skill is not "+
			"available, call %s directly with type \"iac\" and metadata (title, description, "+
			"remediationAdvice = how to harden this configuration). If the skill text disagrees with this "+
			"deny, follow this deny. Apply remediation_steps only with Write or StrReplace, never a shell "+
			"command (shell writes are not scanned). Make the smallest change to the flagged "+
			"configuration; if the fix needs a resource in another file, that is also a gated write. If "+
			"the blocked write creates a new file, the fix is that same Write with the fixed content.\n"+
			"If %s is unavailable, change nothing and ignore nothing because of it. Report the findings "+
			"as unresolved and tell the user to %s.\n"+
			"3. False positive: run exactly this command for it:\n%s"+
			"4. Verify: run `cx scan iac-realtime -s <file>` with the canonical cx path, then retry the "+
			"blocked write once with the fixed content (never the same unchanged content); the hook on "+
			"that retry is the check. If it is denied, a remaining finding, or one your fix introduced, "+
			"gets one more %s call. Stop after 3 denied retries or when the tool returns no safe change. "+
			"Then ignore only findings that meet step 1, report the rest as unresolved, and stop editing "+
			"the file. Do not ask whether to continue.\n"+
			"Never write this content through another tool, a shell command, or a different file to "+
			"bypass the scan, and do not paste it in chat instead.\n"+
			"5. Always finish by showing the skill's IaC Remediation Summary verbatim (without the skill, "+
			"use the same structure), even if you asked the user a question, the file is new, or the "+
			"retry passed. Give one line for every finding, omitting empty sections (finding lines are "+
			"0-based; show line + 1):\n"+
			"Remediation Summary\n"+
			"Title: <title>  Severity: <severity>  Line: <line+1>\n"+
			"Files Modified: - <file> line <n>: <change>\n"+
			"Ignored: - <title> line <n> <severity> - <evidence: the user's words, or the file and line "+
			"you read>\n"+
			"Unresolved: - <title> line <n> <severity> - <reason>\n"+
			"Final status: All fixed | Partially fixed | Unresolved\n"+
			"This is a security check triggered mid-task, not a new task: then continue with the task "+
			"the user originally asked for.\n",
		filePath, skill, codeTool, codeTool, codeTool, agentprofile.McpReconnect(agentLabel(agenthooks.AgentCursor)),
		suppressCmds, codeTool,
	)
}

// kicsSkillName returns the agent-specific skill invocation string for KICS remediation.
func kicsSkillName(agent agenthooks.AgentID) string {
	if agent == agenthooks.AgentGemini {
		return "/cx-devassist-kics"
	}
	return "cx-devassist:cx-devassist-kics"
}
