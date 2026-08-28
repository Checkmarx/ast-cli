package kicsengine

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

func TestPlatformsForFile(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"main.tf", []string{platformTerraform}},
		{"MAIN.TF", []string{platformTerraform}},
		{"vars.tfvars", []string{platformTerraform}},
		{"/tmp/scan/Dockerfile", []string{platformDockerfile}},
		{"dockerfile", []string{platformDockerfile}},
		{"Dockerfile.prod", []string{platformDockerfile}},
		{"app.dockerfile", []string{platformDockerfile}},
		{"base.ubi8", []string{platformDockerfile}},
		{"base.debian", []string{platformDockerfile}},
		{"service.proto", []string{platformGRPC}},
		{"build.sh", []string{platformBuildah}},
		{"hosts.ini", []string{platformAnsible}},
		{"ansible.cfg", []string{platformAnsible}},
		{"app.conf", []string{platformAnsible}},
		// Ambiguous or unknown extensions must load every platform rather than guess.
		{"deployment.yaml", nil},
		{"template.json", nil},
		{"playbook.yml", nil},
		// .bicep has one parser but its queries live under azureResourceManager, so narrowing
		// on the parser's own platform name would find no queries at all.
		{"main.bicep", nil},
		{"README.md", nil},
		{"noextension", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.DeepEqual(t, PlatformsForFile(tt.path), tt.want)
		})
	}
}

func TestQueryDirs(t *testing.T) {
	root := filepath.Join("assets", "queries")

	// No narrowing must keep the whole tree, so an unrecognised file still scans everything.
	assert.DeepEqual(t, queryDirs(root, nil), []string{root})

	// Narrowing always keeps the common directory alongside the platform directory.
	assert.DeepEqual(t, queryDirs(root, []string{platformTerraform}), []string{
		filepath.Join(root, platformTerraform),
		filepath.Join(root, commonQueriesDir),
	})
}

// TestNarrowingDoesNotChangeFindings is the guard that makes narrowing safe to ship. Narrowing
// drops both queries and parsers, so if a future KICS release moved a query to another platform
// or taught a parser a new extension, narrowing could start hiding findings. This compares the
// narrowed scan against the full scan and fails on any divergence by QueryID or SimilarityID.
func TestNarrowingDoesNotChangeFindings(t *testing.T) {
	for _, fixture := range []struct{ path, name string }{
		{terraformFixture, "positive1.tf"},
		{dockerfileFixture, "Dockerfile"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			narrowed := fingerprint(scanFixture(t, fixture.path, fixture.name, fixture.name))
			all := fingerprint(scanFixture(t, fixture.path, fixture.name, ""))

			assert.Assert(t, len(all) > 0, "fixture should produce findings")
			assert.DeepEqual(t, narrowed, all)
		})
	}
}

// fingerprint reduces a report to a sorted list of QueryID+SimilarityID keys, which is the
// identity downstream consumers and ignore-files depend on.
func fingerprint(results wrappers.KicsResultsCollection) []string {
	keys := make([]string, 0, len(results.Results))
	for _, query := range results.Results {
		for _, location := range query.Locations {
			keys = append(keys, fmt.Sprintf("%s|%s", query.QueryID, location.SimilarityID))
		}
	}
	sort.Strings(keys)
	return keys
}
