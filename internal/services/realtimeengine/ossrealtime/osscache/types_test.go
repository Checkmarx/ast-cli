package osscache

import (
	"testing"
	"time"
)

func TestCache_GetSetTTL(t *testing.T) {
	c := &Cache{}
	if !c.GetTTL().IsZero() {
		t.Fatalf("zero-value Cache TTL should be zero, got %v", c.GetTTL())
	}

	want := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	c.SetTTL(want)
	if got := c.GetTTL(); !got.Equal(want) {
		t.Errorf("GetTTL() = %v, want %v", got, want)
	}

	later := want.Add(2 * time.Hour)
	c.SetTTL(later)
	if got := c.GetTTL(); !got.Equal(later) {
		t.Errorf("GetTTL after update = %v, want %v", got, later)
	}
}

func TestPackageEntry_FieldsRoundTrip(t *testing.T) {
	entry := PackageEntry{
		PackageID:      "npm:lodash@4.17.21",
		PackageManager: "npm",
		PackageName:    "lodash",
		PackageVersion: "4.17.21",
		Status:         "Vulnerable",
		Vulnerabilities: []Vulnerability{{
			CVE:         "CVE-2021-23337",
			Description: "Command injection",
			Severity:    "High",
		}},
	}
	if entry.PackageName != "lodash" || entry.PackageVersion != "4.17.21" {
		t.Fatalf("unexpected package identity: %+v", entry)
	}
	if len(entry.Vulnerabilities) != 1 || entry.Vulnerabilities[0].CVE == "" {
		t.Fatalf("unexpected vulnerabilities: %+v", entry.Vulnerabilities)
	}
}

func TestCache_WithPackages(t *testing.T) {
	ttl := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	c := Cache{
		TTL: ttl,
		Packages: []PackageEntry{{
			PackageManager: "npm",
			PackageName:    "express",
			PackageVersion: "4.18.0",
			Status:         "OK",
		}},
	}
	if c.GetTTL() != ttl {
		t.Errorf("TTL mismatch")
	}
	if len(c.Packages) != 1 || c.Packages[0].PackageName != "express" {
		t.Errorf("packages = %+v", c.Packages)
	}
}
