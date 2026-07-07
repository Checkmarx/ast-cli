//go:build !integration

package kics

import (
	"strings"
	"testing"

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
	reason, _ := formatFindings("/project/Dockerfile", findings)
	if !strings.Contains(reason, "KICS") {
		t.Errorf("reason should contain KICS, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsFilePath(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings)
	if !strings.Contains(reason, "/project/Dockerfile") {
		t.Errorf("reason should contain file path, got: %q", reason)
	}
}

func TestFormatFindings_ReasonContainsSeverityAndTitle(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	reason, _ := formatFindings("/project/Dockerfile", findings)
	if !strings.Contains(reason, "HIGH") {
		t.Errorf("reason should contain severity, got: %q", reason)
	}
	if !strings.Contains(reason, "PrivilegedContainer") {
		t.Errorf("reason should contain finding title, got: %q", reason)
	}
}

func TestFormatFindings_ContextContainsFixInstruction(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings)
	if !strings.Contains(ctx, "fix") && !strings.Contains(ctx, "Fix") {
		t.Errorf("context should contain fix instruction, got: %q", ctx)
	}
}

func TestFormatFindings_ContextContainsDoNotBypass(t *testing.T) {
	findings := []iacrealtime.IacRealtimeResult{iacResult("PrivilegedContainer", "sim1", "HIGH", 5)}
	_, ctx := formatFindings("/project/Dockerfile", findings)
	if !strings.Contains(ctx, "bypass") {
		t.Errorf("context should warn against bypass, got: %q", ctx)
	}
}
