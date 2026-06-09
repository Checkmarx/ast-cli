package asca

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
)

// findingKey is the deduplication tuple used for delta detection.
// Mirrors the cx-security plugin's matching logic.
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

// findingsSummary returns the bullet list shared by both message fields.
func findingsSummary(findings []grpcs.ScanDetail) string {
	var sb strings.Builder
	for _, f := range findings {
		remediation := f.Remediation
		if remediation == "" {
			remediation = "No remediation provided"
		}
		fmt.Fprintf(&sb, "  - Line %d: [%s] %s — %s\n", f.Line, f.Severity, f.RuleName, remediation)
	}
	return sb.String()
}

// formatFindings builds the two verdict fields delivered to the agent: the
// human-readable deny reason (rendered as permissionDecisionReason) and the
// remediation guidance injected into the agent's context (additionalContext).
// ast-cx-hooks v1.0.2 carries these as distinct fields via RejectEditWithContext.
func formatFindings(filePath string, findings []grpcs.ScanDetail) (reason, context string) {
	summary := findingsSummary(findings)
	cxExe, err := os.Executable()
	cxBinary := "cx"
	if err == nil {
		cxBinary = filepath.Base(cxExe)
	}
	return permissionDecisionReason(filePath, summary), additionalContext(filePath, cxBinary)
}

// permissionDecisionReason is the human-readable deny message shown to the user.
func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"ASCA security scan detected vulnerabilities in %s."+
			"\n\n⚠️  ASCA scans the changed file in isolation and cannot see imported modules or "+
			"helper files. Findings may be false positives when sanitization or validation is "+
			"performed in code that ASCA cannot reach. Review each finding in context before acting."+
			"\nFindings:\n%s",
		filePath, summary,
	)
}

// additionalContext is injected into the agent's context window to drive remediation.
// Does not repeat the findings — the agent already has them from permissionDecisionReason.
func additionalContext(filePath, cxBinary string) string {
	return fmt.Sprintf(
		"ASCA detected vulnerabilities in %s. "+
			"ANALYZE each finding to determine if it is a real vulnerability or a false positive "+
			"caused by ASCA's single-file scope (it cannot see imported modules or helper files). "+
			"For each real finding, call the mcp__Checkmarx__codeRemediation tool with:\n"+
			"  {\n"+
			"    \"language\": \"[auto-detected programming language]\",\n"+
			"    \"metadata\": {\n"+
			"      \"ruleId\": \"[rule_name from scan]\",\n"+
			"      \"description\": \"[description from scan]\",\n"+
			"      \"remediationAdvice\": \"[remediationAdvise from scan]\"\n"+
			"    },\n"+
			"    \"type\": \"sast\"\n"+
			"  }\n"+
			"Use the remediation guidance returned by the tool to fix the vulnerability. "+
			"If a finding is a confirmed false positive, suppress it by calling:\n"+
			"  %s ignore-vulnerability --scan-type asca --data '{\"FileName\":\"<file_name>\",\"Line\":<line>,\"RuleID\":<rule_id>}'\n"+
			"using the file_name, line, and rule_id from the finding, then proceed.",
		filePath, cxBinary,
	)
}
