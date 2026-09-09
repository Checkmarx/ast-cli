//go:build !integration

package kics

import (
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
)

func iacResult(title, similarityID, severity string, line int) iacrealtime.IacRealtimeResult {
	return iacrealtime.IacRealtimeResult{
		Title:        title,
		SimilarityID: similarityID,
		Severity:     severity,
		Description:  "test description",
		Locations:    []realtimeengine.Location{{Line: line}},
	}
}

func iacResultWithPlatform(title, platform string) iacrealtime.IacRealtimeResult {
	r := iacResult(title, "sim1", "HIGH", 1)
	r.Platform = platform
	return r
}

// ── NewFindings ───────────────────────────────────────────────────────────────

func TestNewFindings_NilOriginalReturnsAll(t *testing.T) {
	newScan := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	got := NewFindings(nil, newScan)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestNewFindings_IdenticalScansReturnsEmpty(t *testing.T) {
	scan := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	got := NewFindings(scan, scan)
	if len(got) != 0 {
		t.Fatalf("expected 0 new findings, got %d", len(got))
	}
}

func TestNewFindings_NewVulnReturned(t *testing.T) {
	orig := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	newScan := []iacrealtime.IacRealtimeResult{
		iacResult("PrivilegedContainer", "sim1", "HIGH", 5),
		iacResult("OpenSecurityGroup", "sim2", "CRITICAL", 10),
	}
	got := NewFindings(orig, newScan)
	if len(got) != 1 || got[0].Title != "OpenSecurityGroup" {
		t.Fatalf("expected finding for OpenSecurityGroup, got %v", got)
	}
}

func TestNewFindings_PreExistingFindingNotReturned(t *testing.T) {
	orig := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	newScan := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	got := NewFindings(orig, newScan)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (pre-existing), got %d", len(got))
	}
}

func TestNewFindings_EmptyNewScanReturnsEmpty(t *testing.T) {
	orig := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	got := NewFindings(orig, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(got))
	}
}

func TestNewFindings_DeltaDedup_SameKeyNotDoubled(t *testing.T) {
	orig := []iacrealtime.IacRealtimeResult{iacResult("RuleA", "simA", "HIGH", 1)}
	newScan := []iacrealtime.IacRealtimeResult{
		iacResult("RuleA", "simA", "HIGH", 1),   // pre-existing
		iacResult("RuleB", "simB", "MEDIUM", 2), // new
	}
	got := NewFindings(orig, newScan)
	if len(got) != 1 || got[0].Title != "RuleB" {
		t.Fatalf("expected only RuleB as new finding, got %v", got)
	}
}

// ── formatFindings ────────────────────────────────────────────────────────────

func TestFormatFindings_ReasonContainsKICS(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(reason, "KICS") {
		t.Errorf("reason should contain KICS, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsFilePath(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(reason, "/project/Dockerfile") {
		t.Errorf("reason should contain file path, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsSeverityAndTitle(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(reason, "HIGH") {
		t.Errorf("reason should contain severity, got: %q", reason)
	}
	if !strings.Contains(reason, "PrivilegedContainer") {
		t.Errorf("reason should contain finding title, got: %q", reason)
	}
}

func TestFormatFindings_ContextContainsFixInstruction(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "fix") && !strings.Contains(ctx, "Fix") && !strings.Contains(ctx, "remediation") {
		t.Errorf("context should contain fix/remediation instruction, got: %q", ctx)
	}
}

func TestFormatFindings_ContextContainsDoNotBypass(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "bypass") {
		t.Errorf("context should warn against bypass, got: %q", ctx)
	}
}

// ── remediation tool routing ─────────────────────────────────────────────────

func TestFormatFindings_DockerfilePlatformUsesCodeRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("VulnerableBaseImage", "Dockerfile"),
	}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("Dockerfile KICS context should call codeRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("Dockerfile KICS context should not call imageRemediation, got: %q", ctx)
	}
}

func TestFormatFindings_DockerComposePlatformUsesCodeRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("VulnerableBaseImage", "DockerCompose"),
	}
	_, ctx := formatFindings("/project/stack.yml", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("docker-compose KICS context should call codeRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("docker-compose KICS context should not call imageRemediation, got: %q", ctx)
	}
}

func TestFormatFindings_TerraformUsesCodeRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("OpenSecurityGroup", "Terraform"),
	}
	_, ctx := formatFindings("/project/main.tf", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("Terraform context should call codeRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("Terraform context should not call imageRemediation, got: %q", ctx)
	}
}

func TestCursorAdditionalContext_UsesCodeRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	ctx := cursorAdditionalContext("/project/Dockerfile", "cx", findings, "/project", "sess1")
	if !strings.Contains(ctx, "mcp__plugin-cx-devassist-Checkmarx__codeRemediation") {
		t.Errorf("cursor KICS context should use codeRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "imageRemediation") {
		t.Errorf("cursor KICS context should not use imageRemediation, got: %q", ctx)
	}
	if !strings.Contains(ctx, "cx-devassist-kics.mdc") {
		t.Errorf("cursor KICS context should reference cx-devassist-kics.mdc rule, got: %q", ctx)
	}
	if !strings.Contains(ctx, "cx-devassist:cx-devassist-kics") {
		t.Errorf("cursor KICS context should reference cx-devassist:cx-devassist-kics skill, got: %q", ctx)
	}
}

func TestFormatFindings_RoutesCursorContext(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentCursor, "/project", "sess1")
	if !strings.Contains(ctx, "cx-devassist-kics.mdc") {
		t.Fatalf("cursor agent should get context with rule reference, got %q", ctx)
	}
	if strings.Contains(ctx, "MANDATORY NEXT STEPS") {
		t.Fatalf("cursor context should not have verbose MANDATORY NEXT STEPS block, got %q", ctx)
	}
	if !strings.Contains(ctx, "codeRemediation") {
		t.Fatalf("cursor KICS context should reference codeRemediation, got %q", ctx)
	}
	_, ctx = formatFindings("/project/main.tf", findings, agenthooks.AgentClaude, "/project", "sess1")
	if strings.Contains(ctx, "cx-devassist-kics.mdc") {
		t.Fatalf("claude agent should not get cursor-specific rule reference, got %q", ctx)
	}
	if !strings.Contains(ctx, "codeRemediation") {
		t.Fatalf("claude KICS context should reference codeRemediation, got %q", ctx)
	}
}

func TestAdditionalContext_GeminiUsesUnderscoreMCPNames(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("VulnerableBaseImage", "Dockerfile"),
	}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentGemini, "/project", "sess1")
	if !strings.Contains(ctx, "mcp_Checkmarx_codeRemediation") {
		t.Errorf("Gemini context should use underscore codeRemediation MCP name, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("Gemini context should not use double-underscore MCP name, got: %q", ctx)
	}
	if strings.Contains(ctx, "imageRemediation") {
		t.Errorf("Gemini KICS context should not use imageRemediation, got: %q", ctx)
	}
}

func TestAdditionalContext_ClaudeOffersSuppress(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude, "/project", "sess1")
	if !strings.Contains(ctx, "ignore-vulnerability") {
		t.Errorf("Claude context should include suppress commands, got %q", ctx)
	}
	if !strings.Contains(ctx, `--scan-type iac`) {
		t.Errorf("Claude context should include iac scan type, got %q", ctx)
	}
}

func TestCursorAdditionalContext_OffersSuppress(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	ctx := cursorAdditionalContext("/project/Dockerfile", "cx", findings, "/project", "sess1")
	if !strings.Contains(ctx, "ignore-vulnerability") {
		t.Errorf("cursor context should include suppress commands, got %q", ctx)
	}
}

func TestCursorAdditionalContext_MatchesAscaAskUserWording(t *testing.T) {
	ctx := cursorAdditionalContext("/project/main.tf", "cx", nil, "/project", "sess1")
	for _, want := range []string{
		"ANALYZE each finding",
		"for every real finding",
		"mark as a confirmed false positive and unblock the write",
		"intentionally-inserted misconfiguration",
		"never because the request seems intentional",
		"If the user chooses to suppress a finding",
	} {
		if !strings.Contains(ctx, want) {
			t.Errorf("cursor KICS context should contain %q, got: %q", want, ctx)
		}
	}
	if strings.Contains(ctx, "accept the risk") {
		t.Errorf("cursor KICS context should not use old suppress wording, got: %q", ctx)
	}
}
