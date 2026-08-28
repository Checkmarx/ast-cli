package kicsengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"gotest.tools/assert"
)

const (
	terraformFixture  = "../../../test/integration/data/positive1.tf"
	dockerfileFixture = "../../../test/integration/data/positive/Dockerfile"
)

// scanFixture copies a fixture into a temp directory under the given name, scans it, and
// returns the parsed report. Passing an empty sourceFile disables narrowing, which is how the
// tests compare a narrowed scan against the full one.
func scanFixture(t *testing.T, fixturePath, name, sourceFile string) wrappers.KicsResultsCollection {
	t.Helper()

	body, err := os.ReadFile(fixturePath)
	assert.NilError(t, err)

	scanDir, outputDir := t.TempDir(), t.TempDir()
	scanFile := filepath.Join(scanDir, name)
	assert.NilError(t, os.WriteFile(scanFile, body, 0o600))

	if sourceFile != "" {
		sourceFile = scanFile
	}
	assert.NilError(t, Scan(context.Background(), Options{
		ScanPath:   scanDir,
		OutputDir:  outputDir,
		SourceFile: sourceFile,
	}))

	raw, err := os.ReadFile(filepath.Join(outputDir, ResultsFileName))
	assert.NilError(t, err)

	var results wrappers.KicsResultsCollection
	assert.NilError(t, json.Unmarshal(raw, &results))
	return results
}

// TestScanProducesResults is the core guarantee of the embedded engine: it runs KICS with no
// container runtime and no downloads, and leaves behind the same results.json the container
// engine wrote, parseable by the existing CLI result types.
func TestScanProducesResults(t *testing.T) {
	results := scanFixture(t, terraformFixture, "positive1.tf", "positive1.tf")

	assert.Assert(t, len(results.Results) > 0, "expected findings in a known-vulnerable terraform file")
	assert.Assert(t, results.Count > 0, "expected a non-zero total counter")

	for _, query := range results.Results {
		assert.Assert(t, query.QueryID != "", "every finding must carry a query id")
		assert.Assert(t, len(query.Locations) > 0, "every finding must carry at least one location")
		for _, location := range query.Locations {
			assert.Assert(t, location.SimilarityID != "", "every location must carry a similarity id")
		}
	}
}

// TestAssetsRootIsReusable checks the extraction is memoised and self-describing, so repeated
// scans in one process do not re-unpack the archive.
func TestAssetsRootIsReusable(t *testing.T) {
	first, err := AssetsRoot()
	assert.NilError(t, err)

	second, err := AssetsRoot()
	assert.NilError(t, err)
	assert.Equal(t, first, second)

	for _, dir := range []string{queriesDirName, librariesDirName, transitionDirName} {
		info, statErr := os.Stat(filepath.Join(first, assetsDirName, dir))
		assert.NilError(t, statErr)
		assert.Assert(t, info.IsDir(), "expected %s to be extracted", dir)
	}
}
