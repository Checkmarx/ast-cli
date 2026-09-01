package asca

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
)

// agentCursor identifies Cursor for the shell-quoting branch below. Cursor's CLI
// reformats single-quoted commands into double-quoted ones (notably on Windows
// PowerShell), so its suppression commands need double-quoted JSON with the
// embedded quotes escaped for the shell actually in play (see cursorEscapeJSON) —
// otherwise the reformatted command corrupts the JSON payload or drops
// --ignored-file-path, silently sending the suppression to the wrong file.
const agentCursor = "Cursor"

// agentGemini identifies Gemini CLI. Its suppress commands run through PowerShell on
// Windows, which strips embedded double quotes from native-exe arguments, so Gemini
// uses ignore.QuoteDataFlag. Other non-Cursor agents keep the original single-quoted JSON.
const agentGemini = "Gemini"

// goosWindows is runtime.GOOS's value on Windows, factored out because the shell-quoting
// checks below (and their tests) compare against it repeatedly.
const goosWindows = "windows"

// findingKey is the deduplication tuple used for delta detection.
// Mirrors the cx-devassist plugin's matching logic.
type findingKey struct {
	ruleID          uint32
	problematicLine string // TrimSpace applied
}

func keyOf(d grpcs.ScanDetail) findingKey {
	return findingKey{
		ruleID:          d.RuleID,
		problematicLine: strings.TrimSpace(d.ProblematicLine),
	}
}

// NewFindings returns scan details present in newScan that have no matching key in originalScan.
// A new file (originalScan == nil) returns newScan unchanged — any vuln is "new".
func NewFindings(originalScan, newScan []grpcs.ScanDetail) []grpcs.ScanDetail {
	if originalScan == nil {
		return newScan
	}
	baseline := make(map[findingKey]struct{}, len(originalScan))
	for _, d := range originalScan {
		baseline[keyOf(d)] = struct{}{}
	}
	var out []grpcs.ScanDetail
	for _, d := range newScan {
		if _, exists := baseline[keyOf(d)]; !exists {
			out = append(out, d)
		}
	}
	return out
}

// findingsSummary returns the bullet list shared by both message fields. Each line
// carries the file_name (basename), line, rule_id, severity and remediation so the
// agent has everything needed to suppress a confirmed false positive via
// `cx ignore-vulnerability` without having to re-scan to recover the rule_id.
func findingsSummary(findings []grpcs.ScanDetail) string {
	var sb strings.Builder
	for _, f := range findings {
		remediation := f.Remediation
		if remediation == "" {
			remediation = "No remediation provided"
		}
		fmt.Fprintf(&sb, "  - %s line %d [%s] %s (rule_id %d) — %s\n",
			f.FileName, f.Line, f.Severity, f.RuleName, f.RuleID, remediation)
	}
	return sb.String()
}

// formatFindings builds the two verdict fields delivered to the agent: the
// human-readable deny reason (rendered as permissionDecisionReason) and the
// remediation guidance injected into the agent's context (additionalContext).
// ast-cx-hooks v1.0.3 carries these as distinct fields via RejectEditWithContext.
func formatFindings(filePath string, findings []grpcs.ScanDetail, workDir, agent, sessionID string) (reason, context string) {
	summary := findingsSummary(findings)
	cxExe, err := os.Executable()
	cxBinary := "cx"
	if err == nil {
		cxBinary = cxExe
	}
	reason = permissionDecisionReason(filePath, summary)
	if agent == agentCursor {
		context = cursorAdditionalContext(filePath, cxBinary, findings, workDir, sessionID)
	} else {
		context = additionalContext(filePath, cxBinary, findings, workDir, agent, sessionID)
	}
	return reason, context
}

// ignoredFilePathFlag returns the " --ignored-file-path '<path>'" fragment that pins
// the suppression command to the workspace ignore file, anchored at the hook event's
// workDir. This keeps the write (cx ignore-vulnerability) and the later read (the hook)
// on the same absolute file regardless of either process's CWD — without it, a host CLI
// that runs the agent's shell from a different directory than the hook (e.g. Copilot CLI)
// would write and read different files. Returns "" when workDir is unknown so the command
// falls back to its CWD-relative default.
func ignoredFilePathFlag(workDir string) string {
	if workDir == "" {
		return ""
	}
	return fmt.Sprintf(" --ignored-file-path '%s'", ignore.PathFor(workDir))
}

// cursorIgnoredFilePathFlag is the Cursor-specific variant of ignoredFilePathFlag. It uses
// double quotes and converts backslashes to forward slashes so the flag survives Windows
// PowerShell and cmd.exe without the agent needing to re-quote it. (Cursor agents on Windows
// tend to reformat single-quoted shell commands into double-quoted form and drop flags that
// have complex quoting, causing the ignore entry to land in the wrong directory.)
func cursorIgnoredFilePathFlag(workDir string) string {
	if workDir == "" {
		return ""
	}
	p := filepath.ToSlash(ignore.PathFor(workDir))
	return fmt.Sprintf(" --ignored-file-path %q", p)
}

// cursorEscapeJSON escapes the embedded `"` in a JSON payload so it survives being placed
// inside a double-quoted argument on the shell that actually runs the Cursor agent's command:
// PowerShell on Windows, bash/zsh elsewhere. This runs on the developer's own machine (inside
// the cx process), so runtime.GOOS reflects that shell choice directly. The two shells disagree
// on how to escape an embedded double quote — bash accepts a backslash-escaped `\"`, but
// PowerShell's double-quoted strings do NOT treat `\` as an escape character at all: `\"` ends
// the string early (backslash is literal, then the quote closes it), corrupting everything
// after the first embedded quote. PowerShell requires the quote to be doubled (`""`) instead.
func cursorEscapeJSON(data string) string {
	if runtime.GOOS == goosWindows {
		return strings.ReplaceAll(data, `"`, `""`)
	}
	return strings.ReplaceAll(data, `"`, `\"`)
}

// optionalFlagsFragment carries the suppression's provenance (AI provider, agent, session id) to the
// child `cx ignore-vulnerability` process via --optional-flags, which reads them through
// utils.GetOptionalParam and logs them — matching logRemediationTelemetry's aiProvider/agent/session.
// Empty agent → no fragment (nothing to attribute).
func optionalFlagsFragment(agent, sessionID string) string {
	if agent == "" {
		return ""
	}
	pairs := "aiProvider=" + agent + ";agent=" + agent + "-cli"
	if sessionID != "" {
		pairs += ";aiAgentSessionId=" + sessionID
	}
	return fmt.Sprintf(" --optional-flags %q", pairs)
}

// permissionDecisionReason is the human-readable deny message shown to the user.
// Contains only the findings — no agent instructions.
func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"ASCA security scan detected vulnerabilities in %s.\nFindings:\n%s",
		filePath, summary,
	)
}

// additionalContext is injected into the agent's context window to drive remediation.
// Contains all action instructions — not shown directly to the user on Claude; on Gemini
// BeforeTool it is folded into the hook deny reason by the ast-cx-hooks gemini adapter.
// Used for Claude, Copilot, Gemini, and other non-Cursor agents. Gemini suppress commands
// use ignore.QuoteDataFlag (PowerShell-safe quoting on Windows); other agents keep the
// original single-quoted JSON payload.
func additionalContext(filePath, cxBinary string, findings []grpcs.ScanDetail, workDir, agent, sessionID string) string {
	provenance := optionalFlagsFragment(agent, sessionID)
	var suppressCmds strings.Builder
	for _, f := range findings {
		data, _ := json.Marshal(grpcs.AscaIgnoreFinding{
			FileName: f.FileName,
			Line:     f.Line,
			RuleID:   f.RuleID,
		})
		ignoreFlag := ignoredFilePathFlag(workDir)
		if agent == agentGemini {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type asca --data %s%s%s\n", cxBinary, ignore.QuoteDataFlag(data), ignoreFlag, provenance)
		} else {
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type asca --data '%s'%s%s\n", cxBinary, string(data), ignoreFlag, provenance)
		}
	}
	skill, mcpTool := remediationTargets(agent)
	return fmt.Sprintf(
		"ASCA detected vulnerabilities in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command. "+
			"ANALYZE each finding to determine if it is a real vulnerability or a false positive "+
			"caused by ASCA's single-file scope (it cannot see imported modules or helper files). "+
			"For each real finding, invoke the %s skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation; the skill also handles MCP unavailability and self-recovery. "+
			"If that skill is not available in this session, call %s directly:\n"+
			"  {\n"+
			"    \"language\": \"[auto-detected programming language]\",\n"+
			"    \"metadata\": {\n"+
			"      \"ruleId\": \"[rule_name from scan]\",\n"+
			"      \"description\": \"[description from scan]\",\n"+
			"      \"remediationAdvice\": \"[remediationAdvise from scan]\"\n"+
			"    },\n"+
			"    \"type\": \"sast\"\n"+
			"  }\n"+
			"Use the remediation guidance returned by the tool to fix the vulnerability, then retry the write. "+
			"If a finding is a confirmed false positive, suppress it by running the corresponding command below, then retry the write:\n%s",
		filePath, skill, mcpTool, suppressCmds.String(),
	)
}

// cursorAdditionalContext is remediation guidance for Cursor only. Uses the plugin-prefixed MCP
// tool name and PowerShell --% stop-parsing for suppress commands on Windows.
func cursorAdditionalContext(filePath, cxBinary string, findings []grpcs.ScanDetail, workDir, sessionID string) string {
	provenance := optionalFlagsFragment(agentCursor, sessionID)
	var suppressCmds strings.Builder
	for i := range findings {
		f := &findings[i]
		data, _ := json.Marshal(grpcs.AscaIgnoreFinding{
			FileName: f.FileName,
			Line:     f.Line,
			RuleID:   f.RuleID,
		})
		ignoreFlag := cursorIgnoredFilePathFlag(workDir)
		suppressCmds.WriteString(cursorplugin.IgnoreVulnerabilityCommand(cxBinary, "asca", data, ignoreFlag, provenance))
		suppressCmds.WriteString("\n")
	}
	tool := cursorplugin.MCPTool("codeRemediation")
	return fmt.Sprintf(
		"ASCA detected vulnerabilities in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command. "+
			"ANALYZE each finding to determine if it is a real vulnerability or a false positive "+
			"caused by ASCA's single-file scope (it cannot see imported modules or helper files). "+
			"Follow the cx-hook-deny.mdc rule for this deny. "+
			"ASK THE USER FIRST, for every real finding, before taking any action: \"A security "+
			"vulnerability was detected. Would you like to remediate it (apply an MCP-driven code fix) "+
			"or suppress it (mark as a confirmed false positive and unblock the write)?\" and wait for "+
			"their answer. Do not decide this yourself — an intentionally-inserted vulnerability (e.g. "+
			"in a lab/demo/training file the user asked for on purpose) is NOT the same as a confirmed "+
			"false positive: suppress only on the user's explicit instruction, never because the "+
			"request seems intentional. "+
			"Apply the cx-devassist-asca.mdc rule: for each finding the user asks you to remediate, "+
			"invoke the cx-devassist:cx-devassist-asca skill exactly as written — do not skip, "+
			"abbreviate, or reimplement its steps inline. The findings are already in context so it "+
			"will skip the scan and go directly to MCP-driven remediation; the skill also handles MCP "+
			"unavailability and self-recovery. Always show its Step 5 Remediation Summary to the user "+
			"verbatim when done. "+
			"If that skill is not available in this session, call %s directly:\n"+
			"  {\n"+
			"    \"language\": \"[auto-detected programming language]\",\n"+
			"    \"metadata\": {\n"+
			"      \"ruleId\": \"[rule_name from scan]\",\n"+
			"      \"description\": \"[description from scan]\",\n"+
			"      \"remediationAdvice\": \"[remediationAdvise from scan]\"\n"+
			"    },\n"+
			"    \"type\": \"sast\"\n"+
			"  }\n"+
			"Use the remediation guidance returned by the tool to fix the vulnerability, then retry the write. "+
			"If the user chooses to suppress a finding, run the corresponding command below, then retry the write:\n%s",
		filePath, tool, suppressCmds.String(),
	)
}

// remediationTargets returns the skill invocation and MCP tool name for the agent.
// Gemini CLI's skills are invoked as a bare "/name" slash command and its MCP tool
// names use single underscores (no "__"), unlike Claude Code's "plugin:skill" and
// "mcp__Server__tool" conventions.
func remediationTargets(agent string) (skill, mcpTool string) {
	if agent == agentGemini {
		return "/cx-security-asca", "mcp_Checkmarx_codeRemediation"
	}
	return "cx-devassist:cx-devassist-asca", "mcp__Checkmarx__codeRemediation"
}
