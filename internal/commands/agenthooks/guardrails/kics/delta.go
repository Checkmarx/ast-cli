package kics

import (
	"fmt"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
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
		fmt.Fprintf(&sb, "  - %s line %d [%s] %s — %s\n",
			filePath, line, f.Severity, f.Title, description)
	}
	return sb.String()
}

// formatFindings builds the two verdict fields delivered to the agent.
func formatFindings(filePath string, findings []iacrealtime.IacRealtimeResult) (reason, context string) {
	summary := findingsSummary(filePath, findings)
	return permissionDecisionReason(filePath, summary), additionalContext(filePath, findings)
}

// permissionDecisionReason is the human-readable deny message shown to the user.
func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"KICS security scan detected IaC vulnerabilities in %s.\nFindings:\n%s",
		filePath, summary,
	)
}

// additionalContext is injected into the agent's context window to drive remediation.
// KICS is a deterministic IaC rule engine: unlike ASCA, its findings are not caused by
// missing cross-file context, so the agent is NOT given discretion to treat findings as
// false positives. Every new finding must be fixed.
func additionalContext(filePath string, findings []iacrealtime.IacRealtimeResult) string {
	var findingList strings.Builder
	for _, f := range findings {
		line := 0
		if len(f.Locations) > 0 {
			line = f.Locations[0].Line
		}
		fmt.Fprintf(&findingList, "  - line %d [%s] %s: %s\n",
			line, f.Severity, f.Title, f.Description)
	}
	return fmt.Sprintf(
		"KICS detected IaC misconfigurations in %s. These are deterministic rule matches "+
			"against the configuration itself — they are NOT false positives caused by code "+
			"the scanner cannot see. Do not skip, suppress, or dismiss any finding as a false "+
			"positive, and do not bypass the scan by writing the same content through another "+
			"tool or shell command.\n"+
			"Fix every finding below, then retry the write:\n"+
			"%s"+
			"For each finding, call the mcp__Checkmarx__codeRemediation tool with:\n"+
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
			"the finding.",
		filePath, findingList.String(),
	)
}
