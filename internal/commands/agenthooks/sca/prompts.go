package sca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

// DenyMalicious returns the finding and remediation strings for one or more
// packages classified as Malicious.
func DenyMalicious(pkgs []ossrealtime.OssPackage) (finding, remediation string) {
	var b strings.Builder
	b.WriteString("Checkmarx SCA scan detected MALICIOUS package(s):\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  - %s: Known supply chain attack.\n", pkgLabel(p))
	}
	b.WriteString("\nDo NOT proceed with the installation.")
	return b.String(), remediationNote("malicious package", "safest available version")
}

// DenyVulnerable returns the finding and remediation strings for one or more
// packages with known vulnerabilities. workDir anchors the suppression command's
// --ignored-file-path to the workspace ignore file (see vulnerableRemediationNote).
func DenyVulnerable(pkgs []ossrealtime.OssPackage, workDir string) (finding, remediation string) {
	var b strings.Builder
	b.WriteString("Checkmarx SCA scan found vulnerabilities:\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  - %s: %s\n", pkgLabel(p), vulnDetail(p))
	}
	return b.String(), vulnerableRemediationNote(pkgs, workDir)
}

// remediationNote returns the action steps to include as additionalContext for malicious packages.
// Remediation must go through the cx-devassist-sca skill (or MCP fallback); if MCP is unavailable
// Claude must self-recover by running the registration helper rather than asking the user.
func remediationNote(subject, goal string) string {
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. For each %s, invoke the cx-devassist:cx-devassist-sca skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation to find the %s; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, use mcp__Checkmarx__packageRemediation for each %s.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If mcp__Checkmarx__packageRemediation is unavailable, recover it yourself — do NOT ask\n"+
			"     the user. Run: bash \"${CLAUDE_PLUGIN_ROOT}/mcp/cx_mcp_register.sh\"\n"+
			"     Then tell the user: 'I have registered the Checkmarx MCP. Please run /mcp (or restart\n"+
			"     Claude Code) to load it, then retry.' Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists, the user can acknowledge the finding via\n"+
			"     the Checkmarx Dev Assist interface.",
		subject, goal, subject)
}

// vulnerableRemediationNote returns the action steps for vulnerable packages.
// When no safe version is found, the agent runs the per-package ignore command
// and informs the user.
func vulnerableRemediationNote(pkgs []ossrealtime.OssPackage, workDir string) string {
	cxBinary := cxExecutable()
	ignoreFlag := ignoredFilePathFlag(workDir)
	var suppressCmds strings.Builder
	for _, p := range pkgs {
		data, _ := json.Marshal([]map[string]string{{
			"PackageManager": p.PackageManager,
			"PackageName":    p.PackageName,
			"PackageVersion": p.PackageVersion,
		}})
		fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data '%s'%s\n", cxBinary, string(data), ignoreFlag)
	}
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. For each affected package, invoke the cx-devassist:cx-devassist-sca skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation to find non-vulnerable versions; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, use mcp__Checkmarx__packageRemediation for each affected package.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  3. If mcp__Checkmarx__packageRemediation is unavailable, recover it yourself — do NOT ask\n"+
			"     the user. Run: bash \"${CLAUDE_PLUGIN_ROOT}/mcp/cx_mcp_register.sh\"\n"+
			"     Then tell the user: 'I have registered the Checkmarx MCP. Please run /mcp (or restart\n"+
			"     Claude Code) to load it, then retry.' Do not proceed until the MCP is available.\n"+
			"  4. If no safe version exists for a package, suppress it by running the corresponding command\n"+
			"     and inform the user that no safer version is available:\n%s",
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
