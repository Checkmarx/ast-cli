package sca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
// packages with known vulnerabilities.
func DenyVulnerable(pkgs []ossrealtime.OssPackage) (finding, remediation string) {
	var b strings.Builder
	b.WriteString("Checkmarx SCA scan found vulnerabilities:\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  - %s: %s\n", pkgLabel(p), vulnDetail(p))
	}
	return b.String(), vulnerableRemediationNote(pkgs)
}

// remediationNote returns the action steps to include as additionalContext for malicious packages.
// Remediation must go through the MCP tool; if it is unavailable the user
// must install or enable the Checkmarx MCP server before retrying.
func remediationNote(subject, goal string) string {
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. Use mcp__Checkmarx__packageRemediation for each %s to find the %s.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  2. If mcp__Checkmarx__packageRemediation is unavailable, ask the user to install or enable\n"+
			"     the Checkmarx MCP server, then retry.\n"+
			"  3. If no safe version exists, the user can acknowledge the finding via\n"+
			"     the Checkmarx Dev Assist interface.",
		subject, goal)
}

// vulnerableRemediationNote returns the action steps for vulnerable packages.
// When no safe version is found, the agent runs the per-package ignore command
// and informs the user.
func vulnerableRemediationNote(pkgs []ossrealtime.OssPackage) string {
	cxBinary := cxExecutable()
	var suppressCmds strings.Builder
	for _, p := range pkgs {
		data, _ := json.Marshal([]map[string]string{{
			"PackageManager": p.PackageManager,
			"PackageName":    p.PackageName,
			"PackageVersion": p.PackageVersion,
		}})
		fmt.Fprintf(&suppressCmds, "  %s ignore-vulnerability --scan-type sca --data '%s'\n", cxBinary, string(data))
	}
	return fmt.Sprintf(
		"Action required:\n"+
			"  1. Use mcp__Checkmarx__packageRemediation for each affected package to find the non-vulnerable version.\n"+
			"     This is the only supported remediation path — do not attempt manual version selection.\n"+
			"  2. If mcp__Checkmarx__packageRemediation is unavailable, ask the user to install or enable\n"+
			"     the Checkmarx MCP server, then retry.\n"+
			"  3. If no safe version exists for a package, suppress it by running the corresponding command\n"+
			"     and inform the user that no safer version is available:\n%s",
		suppressCmds.String())
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
