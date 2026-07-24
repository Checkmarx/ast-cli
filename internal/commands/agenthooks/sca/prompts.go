package sca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/agentprofile"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

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
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. For each %s, invoke the cx-devassist:cx-devassist-sca skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation to find the %s; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, use mcp__Checkmarx__packageRemediation for each %s.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If mcp__Checkmarx__packageRemediation is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists, the user can acknowledge the finding via\n"+
			"     the Checkmarx Dev Assist interface.",
		subject, goal, subject, agentprofile.McpReconnect(agent))
}

// vulnerableRemediationNote returns the action steps for vulnerable packages.
// When no safe version is found, the agent runs the per-package ignore command
// and informs the user.
func vulnerableRemediationNote(pkgs []ossrealtime.OssPackage, workDir, agent, sessionID string) string {
	cxBinary := cxExecutable()
	ignoreFlag := ignoredFilePathFlag(workDir)
	provenance := optionalFlagsFragment(agent, sessionID)
	var suppressCmds strings.Builder
	for _, p := range pkgs {
		data, _ := json.Marshal([]map[string]string{{
			"PackageManager": p.PackageManager,
			"PackageName":    p.PackageName,
			"PackageVersion": p.PackageVersion,
		}})
		fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data '%s'%s%s\n", cxBinary, string(data), ignoreFlag, provenance)
	}
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. For each affected package, invoke the cx-devassist:cx-devassist-sca skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation to find non-vulnerable versions; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, use mcp__Checkmarx__packageRemediation for each affected package.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If mcp__Checkmarx__packageRemediation is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists for a package, suppress it by running the corresponding command\n"+
			"     and inform the user that no safer version is available:\n%s",
		agentprofile.McpReconnect(agent),
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
	return fmt.Sprintf(" --optional-flags \"%s\"", pairs)
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
