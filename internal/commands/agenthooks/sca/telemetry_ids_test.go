//go:build !integration

package sca

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/hooktelemetry"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestDenyFrom_RecordsRemediationFindings(t *testing.T) {
	old := BelowThreshold
	t.Cleanup(func() { BelowThreshold = old })
	BelowThreshold = func(severity string) bool { return severity == "Medium" }
	hooktelemetry.TakeRemediationFindings() // start clean

	vuln := []ossrealtime.OssPackage{{PackageName: "lodash", PackageVersion: "4.17.20",
		Vulnerabilities: []ossrealtime.Vulnerability{{CVE: "CVE-2021-23337", Severity: "High"}, {CVE: "CVE-2020-28500", Severity: "Medium"}}}}
	if f, _, _ := denyFrom(nil, vuln, "", "Claude", "s"); f == "" {
		t.Fatal("vulnerable package must deny")
	}
	if ids, count := hooktelemetry.TakeRemediationFindings(); ids != "CVE-2021-23337" || count != 1 {
		t.Fatalf("vulnerable deny must record CVEs above the threshold, got %q with count %d", ids, count)
	}

	mal := []ossrealtime.OssPackage{
		{PackageName: "evil-pkg", PackageVersion: "1.0.0", Status: statusMalicious},
		{PackageName: "no-version", Status: statusMalicious},
	}
	if f, _, _ := denyFrom(mal, nil, "", "Claude", "s"); f == "" {
		t.Fatal("malicious package must deny")
	}
	if ids, count := hooktelemetry.TakeRemediationFindings(); ids != "evil-pkg@1.0.0,no-version" || count != 2 {
		t.Fatalf("malicious deny must record name@version, got %q with count %d", ids, count)
	}
}

func TestLogScan_SendsDetectedCounts(t *testing.T) {
	var sent []*wrappers.DataForAITelemetry
	s := &Scanner{Telemetry: mock.TelemetryMockWrapper{CustomSendAIDataToLog: func(d *wrappers.DataForAITelemetry) error {
		sent = append(sent, d)
		return nil
	}}}

	s.logScan("Claude", "sess", t.TempDir(), nil, nil)
	if len(sent) != 0 {
		t.Fatal("a clean scan sends nothing")
	}

	mal := []ossrealtime.OssPackage{{PackageName: "evil", Status: statusMalicious}}
	vuln := []ossrealtime.OssPackage{{PackageName: "lodash", Vulnerabilities: []ossrealtime.Vulnerability{
		{CVE: "CVE-1", Severity: "High"}, {CVE: "CVE-2", Severity: "Low"}}}}
	s.logScan("Claude", "sess", t.TempDir(), mal, vuln)
	if len(sent) != 1 {
		t.Fatalf("expected one scan event, got %d", len(sent))
	}
	d := sent[0]
	if d.Type != "hooks-detect" || d.Engine != "oss" || d.ScanType != "oss" || d.Agent != "Claude-cli" || d.TotalCount != 3 {
		t.Fatalf("unexpected event: %+v", d)
	}
	if d.ProblemSeverity != "Critical" {
		t.Fatalf("unexpected problemSeverity %q", d.ProblemSeverity)
	}
	if d.StatusResult != "Critical=1,High=1,Medium=0,Low=1,Info=0" {
		t.Fatalf("unexpected statusResult %q", d.StatusResult)
	}

	(&Scanner{}).logScan("Claude", "sess", "", mal, vuln) // no telemetry wrapper: must not panic
}
