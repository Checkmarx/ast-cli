package sca

import (
	"fmt"
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
	return b.String(), remediationNote("affected package", "non-vulnerable version")
}

// remediationNote returns the action steps to include as additionalContext.
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
