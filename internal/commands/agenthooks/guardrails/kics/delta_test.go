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
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(reason, "KICS") {
		t.Errorf("reason should contain KICS, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsFilePath(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(reason, "/project/Dockerfile") {
		t.Errorf("reason should contain file path, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsSeverityAndTitle(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(reason, "HIGH") {
		t.Errorf("reason should contain severity, got: %q", reason)
	}
	if !strings.Contains(reason, "PrivilegedContainer") {
		t.Errorf("reason should contain finding title, got: %q", reason)
	}
}

func TestFormatFindings_ContextContainsFixInstruction(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(ctx, "fix") && !strings.Contains(ctx, "Fix") {
		t.Errorf("context should contain fix instruction, got: %q", ctx)
	}
}

func TestFormatFindings_ContextContainsDoNotBypass(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(ctx, "bypass") {
		t.Errorf("context should warn against bypass, got: %q", ctx)
	}
}

// ── isDockerImageFinding / remediation tool routing ────────────────────────────

func TestIsDockerImageFinding_ByPlatform(t *testing.T) {
	cases := []struct {
		platform string
		want     bool
	}{
		{"Dockerfile", true},
		{"DockerCompose", true},
		{"Docker Compose", true},
		{"dockerfile", true},
		{"Terraform", false},
		{"Kubernetes", false},
		{"CloudFormation", false},
		{"Ansible", false},
	}
	for _, c := range cases {
		findings := []iacrealtime.IacRealtimeResult{
			iacResultWithPlatform("SomeFinding", c.platform),
		}
		// Filename deliberately contradicts platform to prove platform wins.
		if got := isDockerImageFinding("/project/values.yaml", findings); got != c.want {
			t.Errorf("isDockerImageFinding with platform %q = %v, want %v", c.platform, got, c.want)
		}
	}
}

func TestIsDockerImageFinding_FallsBackToFilenameWhenPlatformEmpty(t *testing.T) {
	cases := map[string]bool{
		"/project/Dockerfile":              true,
		"/project/api.dockerfile":          true,
		"/project/docker-compose.yml":      true,
		"/project/docker-compose.yaml":     true,
		"/project/docker-compose.prod.yml": true,
		"/project/compose.yaml":            true,
		"/project/main.tf":                 false,
		"/project/deployment.yaml":         false,
		"/project/values.yaml":             false,
	}
	for path, want := range cases {
		findings := []iacrealtime.IacRealtimeResult{iacResult("SomeFinding", "sim1", "HIGH", 1)}
		if got := isDockerImageFinding(path, findings); got != want {
			t.Errorf("isDockerImageFinding(%q) with no platform = %v, want %v", path, got, want)
		}
	}
}

func TestFormatFindings_DockerfilePlatformUsesImageRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("VulnerableBaseImage", "Dockerfile"),
	}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if !strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("Dockerfile context should call imageRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("Dockerfile context should not call codeRemediation, got: %q", ctx)
	}
}

func TestFormatFindings_DockerComposePlatformUsesImageRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("VulnerableBaseImage", "DockerCompose"),
	}
	_, ctx := formatFindings("/project/stack.yml", findings, agenthooks.AgentClaude)
	if !strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("docker-compose context should call imageRemediation, got: %q", ctx)
	}
}

func TestFormatFindings_TerraformUsesCodeRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{
		iacResultWithPlatform("OpenSecurityGroup", "Terraform"),
	}
	_, ctx := formatFindings("/project/main.tf", findings, agenthooks.AgentClaude)
	if !strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("Terraform context should call codeRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("Terraform context should not call imageRemediation, got: %q", ctx)
	}
}

func TestCursorAdditionalContext_UsesImageRemediation(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	ctx := cursorAdditionalContext("/project/Dockerfile", findings)
	if !strings.Contains(ctx, "mcp__plugin-cx-devassist-Checkmarx__imageRemediation") {
		t.Errorf("cursor KICS context should use imageRemediation, got: %q", ctx)
	}
	if strings.Contains(ctx, "codeRemediation") {
		t.Errorf("cursor KICS context should not use codeRemediation, got: %q", ctx)
	}
	if !strings.Contains(ctx, "cx-devassist-kics.mdc") {
		t.Errorf("cursor KICS context should reference cx-devassist-kics.mdc rule, got: %q", ctx)
	}
}

func TestFormatFindings_RoutesCursorContext(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentCursor)
	if !strings.Contains(ctx, "cx-devassist-kics.mdc") {
		t.Fatalf("cursor agent should get context with rule reference, got %q", ctx)
	}
	if strings.Contains(ctx, "MANDATORY NEXT STEPS") {
		t.Fatalf("cursor context should not have verbose MANDATORY NEXT STEPS block, got %q", ctx)
	}
	if !strings.Contains(ctx, "imageRemediation") {
		t.Fatalf("cursor KICS context should reference imageRemediation, got %q", ctx)
	}
	// Use a non-Docker path for the Claude assertion below: Dockerfile findings
	// always route through imageRemediation (see isDockerImageFinding), so
	// asserting codeRemediation here requires a generic IaC file instead.
	_, ctx = formatFindings("/project/main.tf", findings, agenthooks.AgentClaude)
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
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentGemini)
	if !strings.Contains(ctx, "mcp_Checkmarx_imageRemediation") {
		t.Errorf("Gemini context should use underscore MCP name, got: %q", ctx)
	}
	if strings.Contains(ctx, "mcp__Checkmarx__imageRemediation") {
		t.Errorf("Gemini context should not use double-underscore MCP name, got: %q", ctx)
	}
}

func TestAdditionalContext_ClaudeDoesNotOfferSuppress(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings, agenthooks.AgentClaude)
	if strings.Contains(ctx, "ignore-vulnerability") {
		t.Errorf("Claude context should not include suppress commands, got %q", ctx)
	}
}

func TestCursorAdditionalContext_DoesNotOfferSuppress(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	ctx := cursorAdditionalContext("/project/Dockerfile", findings)
	if strings.Contains(ctx, "ignore-vulnerability") {
		t.Errorf("cursor context should not include suppress commands, got %q", ctx)
	}
}
