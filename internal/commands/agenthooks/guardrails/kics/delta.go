package kics

import (
	"fmt"
	"path/filepath"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
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
// Cursor receives cursorAdditionalContext (folded into agent_message); other agents
// (including Gemini) receive additionalContext, with MCP tool names adjusted per agent.
func formatFindings(filePath string, findings []iacrealtime.IacRealtimeResult, agent agenthooks.AgentID) (reason, context string) {
	summary := findingsSummary(filePath, findings)
	reason = permissionDecisionReason(filePath, summary)
	switch agent {
	case agenthooks.AgentCursor:
		context = cursorAdditionalContext(filePath, findings)
	default:
		context = additionalContext(filePath, findings, agent)
	}
	return reason, context
}

// permissionDecisionReason is the human-readable deny message shown to the user.
func permissionDecisionReason(filePath, summary string) string {
	return fmt.Sprintf(
		"KICS security scan detected IaC vulnerabilities in %s.\nFindings:\n%s",
		filePath, summary,
	)
}

// dockerImagePlatforms are the KICS "platform" values (result.Platform, sourced from
// KICS query metadata) whose findings concern container images rather than generic
// IaC misconfigurations. These line up with the fileType enum accepted by the
// imageRemediation MCP tool (Dockerfile, DockerCompose).
var dockerImagePlatforms = map[string]bool{
	"dockerfile":     true,
	"dockercompose":  true,
	"docker compose": true,
}

// isDockerImageFinding reports whether a finding's KICS platform identifies it as a
// container image issue (Dockerfile/docker-compose) rather than generic IaC. Falls
// back to filename heuristics only when platform is unavailable (e.g. older cached
// results), since platform is scanner-reported ground truth and filenames can vary.
func isDockerImageFinding(filePath string, findings []iacrealtime.IacRealtimeResult) bool {
	for i := range findings {
		if findings[i].Platform != "" {
			return dockerImagePlatforms[strings.ToLower(findings[i].Platform)]
		}
	}
	return isDockerImageFileByName(filePath)
}

// isDockerImageFileByName is a filename-based fallback for when KICS platform metadata
// isn't available. Mirrors the basename conventions in params.KicsBaseFilters plus the
// docker-compose/compose naming convention (not in KicsBaseFilters since compose files
// match on the generic .yml/.yaml extensions).
func isDockerImageFileByName(filePath string) bool {
	base := strings.ToLower(filepath.Base(filePath))
	if base == "dockerfile" || strings.HasSuffix(base, ".dockerfile") {
		return true
	}
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
	return name == "docker-compose" || strings.HasPrefix(name, "docker-compose.") ||
		name == "compose" || strings.HasPrefix(name, "compose.")
}

// additionalContext is injected into the agent's context window to drive remediation.
// KICS is a deterministic IaC rule engine: unlike ASCA, its findings are not caused by
// missing cross-file context, so the agent is NOT given discretion to treat findings as
// false positives. Every new finding must be fixed.
// Used for Claude, Gemini, Copilot, and other non-Cursor agents.
func additionalContext(filePath string, findings []iacrealtime.IacRealtimeResult, agent agenthooks.AgentID) string {
	var findingList strings.Builder
	for _, f := range findings {
		line := 0
		if len(f.Locations) > 0 {
			line = f.Locations[0].Line
		}
		fmt.Fprintf(&findingList, "  - line %d [%s] %s: %s\n",
			line, f.Severity, f.Title, f.Description)
	}
	imageTool, codeTool := "mcp__Checkmarx__imageRemediation", "mcp__Checkmarx__codeRemediation"
	if agent == agenthooks.AgentGemini {
		imageTool, codeTool = "mcp_Checkmarx_imageRemediation", "mcp_Checkmarx_codeRemediation"
	}
	return fmt.Sprintf(
		"KICS detected IaC misconfigurations in %s. These are deterministic rule matches "+
			"against the configuration itself — they are NOT false positives caused by code "+
			"the scanner cannot see. Do not skip, suppress, or dismiss any finding as a false "+
			"positive, and do not bypass the scan by writing the same content through another "+
			"tool or shell command.\n"+
			"Fix every finding below, then retry the write:\n"+
			"%s"+
			"%s",
		filePath, findingList.String(), remediationInstructions(filePath, findings, imageTool, codeTool),
	)
}

// remediationInstructions returns the tool-call guidance for the finding's file type.
// Dockerfile/docker-compose findings are about container images, so they must go
// through imageRemediation (base image CVEs, safer tags, hardening). All other
// KICS-supported files (Terraform, Kubernetes manifests, CloudFormation, etc.) are
// generic IaC misconfigurations and go through codeRemediation.
func remediationInstructions(filePath string, findings []iacrealtime.IacRealtimeResult, imageTool, codeTool string) string {
	if isDockerImageFinding(filePath, findings) {
		return fmt.Sprintf("For each finding, call the %s tool with:\n"+
			"  {\n"+
			"    \"imageName\": \"[image name from the finding/file, without the tag]\",\n"+
			"    \"imageTag\": \"[image tag from the finding/file, e.g. latest]\",\n"+
			"    \"fileType\": \"[Dockerfile or DockerCompose, matching this file]\"\n"+
			"  }\n"+
			"Apply the remediation guidance the tool returns (safer base image, pinned digest, "+
			"hardening steps), then retry the write.", imageTool)
	}
	return fmt.Sprintf("For each finding, call the %s tool with:\n"+
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
		"the finding.", codeTool)
}

func cursorRemediationInstructions(filePath string, findings []iacrealtime.IacRealtimeResult) string {
	if isDockerImageFinding(filePath, findings) {
		return fmt.Sprintf("For each finding, call the %s tool with:\n"+
			"  {\n"+
			"    \"imageName\": \"[image name from the finding/file, without the tag]\",\n"+
			"    \"imageTag\": \"[image tag from the finding/file, e.g. latest]\",\n"+
			"    \"fileType\": \"[Dockerfile or DockerCompose, matching this file]\"\n"+
			"  }\n"+
			"Apply the remediation guidance the tool returns (safer base image, pinned digest, "+
			"hardening steps), then retry the write.", cursorplugin.MCPTool("imageRemediation"))
	}
	return fmt.Sprintf("For each finding, call the %s tool with:\n"+
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
		"the finding.", cursorplugin.MCPTool("codeRemediation"))
}

// cursorAdditionalContext is remediation guidance for Cursor only. Cursor has no
// additionalContext field on preToolUse — ast-cx-hooks folds this into agent_message.
func cursorAdditionalContext(filePath string, findings []iacrealtime.IacRealtimeResult) string {
	var findingList strings.Builder
	for i := range findings {
		f := &findings[i]
		line := 0
		if len(f.Locations) > 0 {
			line = f.Locations[0].Line
		}
		fmt.Fprintf(&findingList, "  - line %d [%s] %s: %s\n",
			line, f.Severity, f.Title, f.Description)
	}
	return fmt.Sprintf(
		"KICS IaC findings in %s — apply the cx-hook-deny.mdc rule for this deny, and the "+
			"cx-devassist-kics.mdc rule exactly as written: do not "+
			"skip, abbreviate, or reorder its steps, and always show its Step 5 IaC Remediation Summary "+
			"to the user verbatim when done. "+
			"Do not retry the blocked Write/StrReplace, paste code in chat, or bypass the scan with shell workarounds.\n\n"+
			"Fix every finding below (deterministic IaC rule matches — not false positives). "+
			"%s\n"+
			"%s",
		filePath, cursorRemediationInstructions(filePath, findings), findingList.String(),
	)
}
