//go:build !integration

package sca

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

func TestAboveThreshold_DropsLowCVEsAndEmptyPackages(t *testing.T) {
	old := BelowThreshold
	t.Cleanup(func() { BelowThreshold = old })
	BelowThreshold = func(sev string) bool { return sev == "Low" || sev == "Medium" }

	pkgs := []ossrealtime.OssPackage{
		{PackageName: "mixed", Vulnerabilities: []ossrealtime.Vulnerability{{CVE: "CVE-1", Severity: "High"}, {CVE: "CVE-2", Severity: "Medium"}}},
		{PackageName: "lowonly", Vulnerabilities: []ossrealtime.Vulnerability{{CVE: "CVE-3", Severity: "Low"}}},
		{PackageName: "unknown", Vulnerabilities: []ossrealtime.Vulnerability{{CVE: "CVE-4", Severity: ""}}},
	}
	got := aboveThreshold(pkgs)
	if len(got) != 2 || got[0].PackageName != "mixed" || got[1].PackageName != "unknown" {
		t.Fatalf("unexpected packages: %+v", got)
	}
	if len(got[0].Vulnerabilities) != 1 || got[0].Vulnerabilities[0].CVE != "CVE-1" {
		t.Fatalf("mixed package must keep only its High CVE: %+v", got[0].Vulnerabilities)
	}
	if len(pkgs[0].Vulnerabilities) != 2 {
		t.Fatal("input must not be mutated")
	}
}

func TestDenyFrom_MaliciousIgnoresThreshold(t *testing.T) {
	old := BelowThreshold
	t.Cleanup(func() { BelowThreshold = old })
	BelowThreshold = func(string) bool { return true }

	mal := []ossrealtime.OssPackage{{PackageName: "evil", Status: statusMalicious}}
	if f, _, sev := denyFrom(mal, nil, "", "Copilot", "s"); f == "" || sev != statusMalicious {
		t.Fatalf("malicious must always deny, got finding=%q severity=%q", f, sev)
	}
}
