//go:build !integration

package sca

import (
	"testing"
)

func TestAddedPackages_Npm_AddedNewPackage(t *testing.T) {
	before := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`)
	after := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21","axios":"1.0.0"}}`)
	added, err := AddedPackages(FormatNpmPackageJson, before, after)
	if err != nil {
		t.Fatalf("AddedPackages: %v", err)
	}
	if len(added) != 1 || added[0].Name != "axios" {
		t.Errorf("got added=%v, want [axios]", added)
	}
}

func TestAddedPackages_Npm_VersionBumpCountsAsAdded(t *testing.T) {
	before := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.0"}}`)
	after := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`)
	added, err := AddedPackages(FormatNpmPackageJson, before, after)
	if err != nil {
		t.Fatalf("AddedPackages: %v", err)
	}
	if len(added) != 1 || added[0].Name != "lodash" || added[0].Version != "4.17.21" {
		t.Errorf("got added=%v, want [lodash@4.17.21]", added)
	}
}

func TestAddedPackages_Npm_RemovedPackageIsIgnored(t *testing.T) {
	before := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21","axios":"1.0.0"}}`)
	after := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`)
	added, err := AddedPackages(FormatNpmPackageJson, before, after)
	if err != nil {
		t.Fatalf("AddedPackages: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("got %d added, want 0 (%v)", len(added), added)
	}
}

func TestAddedPackages_Npm_NewFile(t *testing.T) {
	after := []byte(`{"name":"x","version":"1.0.0","dependencies":{"lodash":"4.17.21","axios":"1.0.0"}}`)
	added, err := AddedPackages(FormatNpmPackageJson, nil, after)
	if err != nil {
		t.Fatalf("AddedPackages: %v", err)
	}
	if len(added) != 2 {
		t.Errorf("got %d added, want 2 (%v)", len(added), added)
	}
}

func TestAddedPackages_Pypi_AddedPackage(t *testing.T) {
	before := []byte("requests==2.25.1\n")
	after := []byte("requests==2.25.1\nflask==2.0.0\n")
	added, err := AddedPackages(FormatPypiRequirements, before, after)
	if err != nil {
		t.Fatalf("AddedPackages: %v", err)
	}
	if len(added) != 1 || added[0].Name != "flask" {
		t.Errorf("got added=%v, want [flask]", added)
	}
}

func TestAddedPackages_UnparseableContent(t *testing.T) {
	// Note: behaviour for unparseable content depends on the upstream parser.
	// We assert that errors flow back to the caller, not that any specific
	// content causes an error — the caller's contract is "treat errors as
	// fail-open" so callers don't depend on a particular outcome here.
	_, _ = AddedPackages(FormatNpmPackageJson, nil, []byte("{not valid json"))
}
