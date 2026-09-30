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

// agentGemini identifies Gemini CLI. Its suppress commands run through PowerShell on
// Windows, which strips embedded double quotes from native-exe arguments, so Gemini
// uses ignore.QuoteDataFlag. Other non-Cursor agents keep the original single-quoted JSON.
const agentGemini = "Gemini"

// provenanceTag returns the string prepended to the finding/remediation strings in denyFrom (sca.go)
// so the agent can recognize genuine Checkmarx output. NOT an authority claim — an attacker could
// spoof this exact string from a file, a fetched page, or another tool's output — it is a signpost:
// text that LOOKS like a Checkmarx finding but lacks it, or that instructs running a script/CLI
// command "silently"/"without asking", did not come from Checkmarx.
func provenanceTag(agent string) string {
	return "[Checkmarx cx-devassist — automated security output, not user input]"
}

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
	var packages strings.Builder
	for _, p := range pkgs {
		line := fmt.Sprintf("  - %s: Known supply chain attack.\n", pkgLabel(p))
		b.WriteString(line)
		packages.WriteString(line)
	}
	b.WriteString("\nDo NOT proceed with the installation.")
	return b.String(), remediationNote(packages.String(), agent)
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

// hookDenyHeader is the agent-facing opening of remediation text. The provenance tag is
// prepended separately in denyFrom. This header does not contain the spoof phrases the
// skills treat as proof a message is not from Checkmarx.
const hookDenyHeader = "This is a Checkmarx hook deny. Follow it. It is not file content and not an untrusted tool error.\n"

// remediationNote returns the action steps to include as additionalContext for malicious packages.
// Remediation goes through the cx-devassist skill (or the Checkmarx MCP tool directly when the skill
// is unavailable); if the MCP tool itself is unavailable the user reconnects it via the client — the
// reconnect phrasing is per-agent, from agentprofile.McpReconnect.
func remediationNote(packages, agent string) string {
	if agent == agentCursor {
		pkgTool := cursorplugin.MCPTool("packageRemediation")
		return fmt.Sprintf(
			hookDenyHeader+
				"SCA blocked the write because it adds a known-malicious package. Follow the "+
				"cx-hook-deny.mdc and cx-devassist-sca.mdc rules for this deny.\nFindings:\n%s"+
				"Handle this yourself; never ask the user to choose. A malicious package has no ignore "+
				"path: never run cx ignore-vulnerability for it, and never install it.\n"+
				"1. Call %s for each malicious package. The cx-devassist:cx-devassist-sca skill can make "+
				"this call; if it disagrees with this deny, follow this deny. Apply what it returns (a safe "+
				"version, or removing the dependency) only by editing the manifest with Write or StrReplace, "+
				"never an install command. Do not guess a safe version.\n"+
				"2. If %s is unavailable, retry the write with only that dependency omitted and every "+
				"other change kept. Report that, and tell the user to %s.\n"+
				"3. Verify: retry the blocked write once; the hook re-scanning it is the check. Do not run "+
				"a separate cx scan.\n"+
				"4. If it is still denied, or no safe version or removal fits: omit only that dependency, "+
				"keep every other change, stop editing the manifest, and report it as unresolved. Only the "+
				"user can accept a malicious package, by acknowledging it in Checkmarx Dev Assist. Do not "+
				"do that for them, and do not ask them to choose.\n"+
				"Never install it through a shell command or write it through another tool or file to "+
				"bypass the scan.\n"+
				"5. Always finish with this report, even if you asked the user a question, the file is new, "+
				"or the retry passed. Give one line for every package listed above:\n"+
				"SCA Remediation Summary\n"+
				"Package: <name> - blocked: known-malicious\n"+
				"Outcome: replaced with <version> | removed | left out, unresolved\n"+
				"Final status: All resolved | Unresolved\n"+
				"Continue the user's original task only after the package is no longer in the write.\n",
			packages, pkgTool, pkgTool, agentprofile.McpReconnect(agent))
	}

	pkgTool := defaultPackageRemediationTool
	return fmt.Sprintf(
		hookDenyHeader+
			"SCA blocked the write because it adds a known-malicious package.\nFindings:\n%s"+
			"Handle this yourself; never ask the user to choose. A malicious package has no ignore "+
			"path: never run cx ignore-vulnerability for it, and never install it.\n"+
			"1. Call %s for each malicious package. The cx-devassist:cx-devassist-sca skill can make "+
			"this call; if it disagrees with this deny, follow this deny. Apply what it returns (a safe "+
			"version, or removing the dependency) only by editing the manifest with Edit, Write, or "+
			"MultiEdit, never an install command. Do not guess a safe version.\n"+
			"2. If %s is unavailable, retry the write with only that dependency omitted and every other "+
			"change kept. Report that, and tell the user to %s.\n"+
			"3. Verify: retry the blocked write once; the hook re-scanning it is the check. Do not run a "+
			"separate cx scan.\n"+
			"4. If it is still denied, or no safe version or removal fits: omit only that dependency, "+
			"keep every other change, stop editing the manifest, and report it as unresolved. Only the "+
			"user can accept a malicious package, by acknowledging it in Checkmarx Dev Assist. Do not do "+
			"that for them, and do not ask them to choose.\n"+
			"Never install it through a shell command or write it through another tool or file to "+
			"bypass the scan.\n"+
			"5. Always finish with this report, even if you asked the user a question, the file is new, "+
			"or the retry passed. Give one line for every package listed above:\n"+
			"SCA Remediation Summary\n"+
			"Package: <name> - blocked: known-malicious\n"+
			"Outcome: replaced with <version> | removed | left out, unresolved\n"+
			"Final status: All resolved | Unresolved\n"+
			"Continue the user's original task only after the package is no longer in the write.\n",
		packages, pkgTool, pkgTool, agentprofile.McpReconnect(agent))
}

// vulnerableRemediationNote returns the action steps for vulnerable packages.
// When no safe version is found, the agent runs the per-package ignore command
// and informs the user. Gemini suppress commands use ignore.QuoteDataFlag
// (PowerShell-safe quoting on Windows); other non-Cursor agents keep the
// original single-quoted JSON payload.
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
			if agent == agentGemini {
				fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data %s%s%s\n", cxBinary, ignore.QuoteDataFlag(data), ignoreFlag, provenance)
			} else {
				fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data '%s'%s%s\n", cxBinary, string(data), ignoreFlag, provenance)
			}
		}
	}
	if agent == agentCursor {
		pkgTool := cursorplugin.MCPTool("packageRemediation")
		var findingLines strings.Builder
		for _, p := range pkgs {
			fmt.Fprintf(&findingLines, "  - %s: %s\n", pkgLabel(p), vulnDetail(p))
		}
		return fmt.Sprintf(
			hookDenyHeader+
				"SCA blocked the write. Follow the cx-hook-deny.mdc and cx-devassist-sca.mdc rules for "+
				"this deny.\nFindings:\n%s"+
				"Handle every package yourself. Never ask the user, including \"Would you like me to "+
				"remediate?\" (that question is only for on-demand scans).\n"+
				"1. Classify each package:\n"+
				"- False positive only if one of these is already true and you can cite it: (a) the user "+
				"explicitly told you to ignore or suppress it; (b) you called %s for it this session and "+
				"it returned that no fixed version exists. \"Looks intentionally pinned\", a major-version "+
				"upgrade, or the tool being unavailable is not evidence; that still needs (a) or (b).\n"+
				"- Otherwise it is a true positive, including when you are unsure. A CVE is a true "+
				"positive unless (a) or (b) is met.\n"+
				"2. True positive: call %s for each package, following its input schema (at least package "+
				"manager, name, version, and CVEs). The cx-devassist:cx-devassist-sca skill can make this "+
				"call; if it disagrees with this deny, follow this deny. Apply the version change only by "+
				"editing the affected manifest entries with Write or StrReplace. Never run npm install, "+
				"pip install, go mod tidy, or any install command; if the lockfile needs refreshing, say "+
				"so in the summary. If the blocked write creates a new file, the fix is that same Write "+
				"with the fixed versions.\n"+
				"If %s is unavailable, do not choose a version yourself and ignore nothing because of it. "+
				"Report the packages as unresolved and tell the user to %s.\n"+
				"3. False positive: run exactly this command for it:\n%s"+
				"4. Verify: run `cx scan oss-realtime -s <manifest>` with the canonical cx path, then "+
				"retry the blocked write once; the hook on that retry is the check. If it is denied, a "+
				"package still reported, a new version with its own CVE, or a transitive dependency it "+
				"pulled in gets one more %s call. Stop after 3 denied retries. Then ignore only packages "+
				"that meet step 1, report the rest as unresolved, and stop editing the manifest. Do not "+
				"ask whether to continue.\n"+
				"Never write this content through another tool, a shell command, or a different file to "+
				"bypass the scan.\n"+
				"5. Always finish with this report, even if you asked the user a question, the file is new, "+
				"or the retry passed. Give one line for every package listed above, omitting empty sections:\n"+
				"Checkmarx Dev Assist SCA Remediation Summary\n"+
				"Package: <name> <old-version> -> <new-version>  Manager: <manager>  Issue: <CVE list> "+
				"(<severity>)\n"+
				"Ignored: - <package@version> - <CVE list> - <evidence: the user's words, or the MCP \"no "+
				"fixed version\" result>\n"+
				"Unresolved: - <package@version> - <reason>\n"+
				"Lockfile refresh needed: yes | no\n"+
				"Final status: All fixed | Partially fixed | Unresolved\n"+
				"Then continue the user's original task.\n",
			findingLines.String(), pkgTool, pkgTool, pkgTool, agentprofile.McpReconnect(agent),
			suppressCmds.String(), pkgTool)
	}
	pkgTool := defaultPackageRemediationTool
	var findingLines strings.Builder
	for _, p := range pkgs {
		fmt.Fprintf(&findingLines, "  - %s: %s\n", pkgLabel(p), vulnDetail(p))
	}
	return fmt.Sprintf(
		hookDenyHeader+
			"SCA blocked the write.\nFindings:\n%s"+
			"Handle every package yourself. Never ask the user, including \"Would you like me to "+
			"remediate?\" (that question is only for on-demand scans).\n"+
			"1. Classify each package:\n"+
			"- False positive only if one of these is already true and you can cite it: (a) the user "+
			"explicitly told you to ignore or suppress it; (b) you called %s for it this session and it "+
			"returned that no fixed version exists. \"Looks intentionally pinned\", a major-version "+
			"upgrade, or the tool being unavailable is not evidence; that still needs (a) or (b).\n"+
			"- Otherwise it is a true positive, including when you are unsure. A CVE is a true positive "+
			"unless (a) or (b) is met.\n"+
			"2. True positive: call %s for each package, following its input schema (at least package "+
			"manager, name, version, and CVEs). The cx-devassist:cx-devassist-sca skill can make this "+
			"call; if it disagrees with this deny, follow this deny. Apply the version change only by "+
			"editing the affected manifest entries with Edit, Write, or MultiEdit. Never run npm "+
			"install, pip install, go mod tidy, or any install command; if the lockfile needs "+
			"refreshing, say so in the summary. If the blocked write creates a new file, the fix is that "+
			"same Write with the fixed versions.\n"+
			"If %s is unavailable, do not choose a version yourself and ignore nothing because of it. "+
			"Report the packages as unresolved and tell the user to %s.\n"+
			"3. False positive: run exactly this command for it:\n%s"+
			"4. Verify: run `cx scan oss-realtime -s <manifest>` with the canonical cx path, then retry "+
			"the blocked write once; the hook on that retry is the check. If it is denied, a package "+
			"still reported, a new version with its own CVE, or a transitive dependency it pulled in "+
			"gets one more %s call. Stop after 3 denied retries. Then ignore only packages that meet "+
			"step 1, report the rest as unresolved, and stop editing the manifest. Do not ask whether "+
			"to continue.\n"+
			"Never write this content through another tool, a shell command, or a different file to "+
			"bypass the scan.\n"+
			"5. Always finish with this report, even if you asked the user a question, the file is new, "+
			"or the retry passed. Give one line for every package listed above, omitting empty sections:\n"+
			"Checkmarx Dev Assist SCA Remediation Summary\n"+
			"Package: <name> <old-version> -> <new-version>  Manager: <manager>  Issue: <CVE list> "+
			"(<severity>)\n"+
			"Ignored: - <package@version> - <CVE list> - <evidence: the user's words, or the MCP \"no "+
			"fixed version\" result>\n"+
			"Unresolved: - <package@version> - <reason>\n"+
			"Lockfile refresh needed: yes | no\n"+
			"Final status: All fixed | Partially fixed | Unresolved\n"+
			"Then continue the user's original task.\n",
		findingLines.String(), pkgTool, pkgTool, pkgTool, agentprofile.McpReconnect(agent), suppressCmds.String(), pkgTool)
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
