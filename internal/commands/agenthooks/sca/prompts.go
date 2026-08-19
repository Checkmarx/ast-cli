package sca

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/agentprofile"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

// agentCursor identifies Cursor for the shell-quoting branch below. Cursor's CLI
// reformats single-quoted commands into double-quoted ones (notably on Windows
// PowerShell), so its suppression commands need double-quoted JSON with the
// embedded quotes escaped for the shell actually in play (see cursorEscapeJSON) —
// otherwise the reformatted command corrupts the JSON payload or drops
// --ignored-file-path, silently sending the suppression to the wrong file.
const agentCursor = "Cursor"

// goosWindows is runtime.GOOS's value on Windows, factored out because the shell-quoting
// checks below (and their tests) compare against it repeatedly.
const goosWindows = "windows"

// defaultPackageRemediationTool is the non-Cursor MCP tool name for package remediation,
// used by both DenyMalicious's remediationNote and DenyVulnerable's vulnerableRemediationNote.
const defaultPackageRemediationTool = "mcp__Checkmarx__packageRemediation"

// DenyMalicious returns the finding and remediation strings for one or more
// packages classified as Malicious.
func DenyMalicious(pkgs []ossrealtime.OssPackage, agent string) (finding, remediation string) {
	var b strings.Builder
	b.WriteString("Checkmarx SCA scan detected MALICIOUS package(s):\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  - %s: Known supply chain attack.\n", pkgLabel(p))
	}
	b.WriteString("\nDo NOT proceed with the installation.")
	return b.String(), remediationNote("malicious package", "safest available version", agent)
}

// DenyVulnerable returns the finding and remediation strings for one or more
// packages with known vulnerabilities. workDir anchors the suppression command's
// --ignored-file-path to the workspace ignore file (see vulnerableRemediationNote).
func DenyVulnerable(pkgs []ossrealtime.OssPackage, workDir, agent, sessionID string) (finding, remediation string) {
	var b strings.Builder
	b.WriteString("Checkmarx SCA scan found vulnerabilities:\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  - %s: %s\n", pkgLabel(p), vulnDetail(p))
	}
	return b.String(), vulnerableRemediationNote(pkgs, workDir, agent, sessionID)
}

// remediationNote returns the action steps to include as additionalContext for malicious packages.
// Remediation goes through the cx-devassist skill (or the Checkmarx MCP tool directly when the skill
// is unavailable); if the MCP tool itself is unavailable the user reconnects it via the client — the
// reconnect phrasing is per-agent, from agentprofile.McpReconnect.
func remediationNote(subject, goal, agent string) string {
	pkgTool := defaultPackageRemediationTool
	skillStep := "  1. For each %s, invoke the cx-devassist:cx-devassist-sca skill — " +
		"the findings are already in context so it will skip the scan and go directly to " +
		"MCP-driven remediation to find the %s; the skill also handles MCP unavailability and self-recovery.\n"
	preamble := ""
	if agent == agentCursor {
		pkgTool = cursorplugin.MCPTool("packageRemediation")
		preamble = "Apply the cx-hook-deny.mdc rule for this deny.\n"
		skillStep = "  1. Apply the cx-devassist-sca.mdc rule: for each %s, invoke the " +
			"cx-devassist:cx-devassist-sca skill exactly as written — do not skip, abbreviate, or " +
			"reimplement its steps inline. The findings are already in context so it will skip the " +
			"scan and go directly to MCP-driven remediation to find the %s; the skill also handles " +
			"MCP unavailability and self-recovery. Always show its Step 5 SCA Remediation Summary to " +
			"the user verbatim when done.\n"
	}
	return fmt.Sprintf(
		preamble+
			"Action required:\n"+
			skillStep+
			"  2. If that skill is not available in this session, use %s for each %s.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If %s is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists, the user can acknowledge the finding via\n"+
			"     the Checkmarx Dev Assist interface.",
		subject, goal, pkgTool, subject, pkgTool, agentprofile.McpReconnect(agent))
}

// vulnerableRemediationNote returns the action steps for vulnerable packages.
// When no safe version is found, the agent runs the per-package ignore command
// and informs the user.
func vulnerableRemediationNote(pkgs []ossrealtime.OssPackage, workDir, agent, sessionID string) string {
	cxBinary := cxExecutable()
	provenance := optionalFlagsFragment(agent, sessionID)
	var suppressCmds strings.Builder
	for _, p := range pkgs {
		data, _ := json.Marshal([]map[string]string{{
			"PackageManager": p.PackageManager,
			"PackageName":    p.PackageName,
			"PackageVersion": p.PackageVersion,
		}})
		if agent == agentCursor {
			ignoreFlag := cursorIgnoredFilePathFlag(workDir)
			suppressCmds.WriteString(cursorplugin.IgnoreVulnerabilityCommand(cxBinary, "sca", data, ignoreFlag, provenance))
			suppressCmds.WriteString("\n")
		} else {
			ignoreFlag := ignoredFilePathFlag(workDir)
			fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data '%s'%s%s\n", cxBinary, string(data), ignoreFlag, provenance)
		}
	}
	if agent == agentCursor {
		pkgTool := cursorplugin.MCPTool("packageRemediation")
		return fmt.Sprintf(
			"Apply the cx-hook-deny.mdc rule for this deny. "+
				"ASK THE USER FIRST, for each affected package, before taking any action: \"A vulnerable "+
				"package was detected. Would you like to remediate it (upgrade/remove via MCP) or "+
				"suppress it (accept the risk)?\" and wait for their answer. Do not decide this "+
				"yourself — a deliberately-pinned or intentionally-included vulnerable package is NOT "+
				"a reason to suppress it without asking; only suppress on the user's explicit "+
				"instruction, never because remediating would undo what the user asked for.\n"+
				"Action required:\n"+
				"  1. If the user asks to remediate: apply the cx-devassist-sca.mdc rule — invoke the "+
				"cx-devassist:cx-devassist-sca skill exactly as written — do not skip, abbreviate, or "+
				"reimplement its steps inline. The findings are already in context so it will skip the "+
				"scan and go directly to MCP-driven remediation to find non-vulnerable versions; the "+
				"skill also handles MCP unavailability and self-recovery. Always show its Step 5 SCA "+
				"Remediation Summary to the user verbatim when done.\n"+
				"  2. If that skill is not available in this session, use %s for each affected package.\n"+
				"     This is the only supported remediation path — do not attempt manual version selection.\n"+
				"  3. If %s is unavailable, tell the user to reconnect the\n"+
				"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
				"  4. If the user asks to suppress instead, or no safe version exists for a package, "+
				"suppress it by running the corresponding command and inform the user of which case "+
				"applied:\n%s",
			pkgTool, pkgTool, agentprofile.McpReconnect(agent),
			suppressCmds.String())
	}
	pkgTool := defaultPackageRemediationTool
	skillStep := "  1. For each affected package, invoke the cx-devassist:cx-devassist-sca skill — " +
		"the findings are already in context so it will skip the scan and go directly to " +
		"MCP-driven remediation to find non-vulnerable versions; the skill also handles MCP unavailability and self-recovery.\n"
	return fmt.Sprintf(
		"Action required:\n"+
			skillStep+
			"  2. If that skill is not available in this session, use %s for each affected package.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If %s is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists for a package, suppress it by running the corresponding command\n"+
			"     and inform the user that no safer version is available:\n%s",
		pkgTool, pkgTool, agentprofile.McpReconnect(agent),
		suppressCmds.String())
}

// ignoredFilePathFlag returns the " --ignored-file-path '<path>'" fragment that
// pins the suppression command to the workspace ignore file, anchored at the hook
// event's workDir. This keeps the write (cx ignore-vulnerability) and the later
// read (the hook) on the same absolute file regardless of either process's CWD —
// without it, a host CLI that runs the agent's shell from a different directory
// than the hook (e.g. Copilot CLI) would write and read different files. Returns
// "" when workDir is unknown so the command falls back to its CWD-relative default.
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

func cxExecutable() string {
	cxExe, err := os.Executable()
	if err != nil {
		return "cx"
	}
	return cxExe
}

func pkgLabel(p ossrealtime.OssPackage) string {
	if p.PackageVersion == "" {
		return p.PackageName
	}
	return p.PackageName + "@" + p.PackageVersion
}

func vulnDetail(p ossrealtime.OssPackage) string {
	if len(p.Vulnerabilities) == 0 {
		return "vulnerability detected"
	}
	v := p.Vulnerabilities[0]
	cve := v.CVE
	if cve == "" {
		cve = "unknown"
	}
	desc := v.Description
	if desc == "" {
		desc = "vulnerability detected"
	}
	return fmt.Sprintf("%s — %s", cve, desc)
}
