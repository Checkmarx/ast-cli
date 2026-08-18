//go:build integration

package integration

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/scarealtime/scaconfig"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

// Separate from scaconfig.Params.WorkingDirName so it never touches the sca-realtime tests' cache dir.
const scaResolverWorkingDirName = "SCAResolverIntegrationTest"

var (
	scaResolverOnce   sync.Once
	scaResolverConfig osinstaller.InstallationConfiguration
	scaResolverPath   string
	scaResolverErr    error
)

// getScaResolverExecutable downloads the real ScaResolver executable once and shares it
// across every test in this file, so the ~114MB download only happens a single time.
func getScaResolverExecutable(t *testing.T) string {
	scaResolverOnce.Do(func() {
		scaResolverConfig = scaconfig.Params
		scaResolverConfig.WorkingDirName = scaResolverWorkingDirName

		_, scaResolverErr = osinstaller.InstallOrUpgrade(&scaResolverConfig, nil)
		if scaResolverErr == nil {
			scaResolverPath = scaResolverConfig.ExecutableFilePath()
		}
	})

	if scaResolverErr != nil {
		t.Fatalf("Failed to download ScaResolver executable: %v", scaResolverErr)
	}

	return scaResolverPath
}

// TestIntegrationScaResolverExecutable_Success runs a real
// `cx scan create --sca-resolver <path> ...` using the actual downloaded ScaResolver
// executable, rather than a path pre-staged outside the test.
func TestIntegrationScaResolverExecutable_Success(t *testing.T) {
	resolverPath := getScaResolverExecutable(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.ScaResolverFlag), resolverPath,
		flag(params.ScaResolverParamsFlag), "-q",
		flag(params.ScanTypes), "sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
	assert.Assert(
		t,
		strings.Contains(buf.String(), "Resolved packages information was saved"),
		"Expected ScaResolver success message not found in logs",
	)
}

// Test --no-scan without --sbom-first (in --sca-resolver-params) is rejected with the
// bad-use error and no scan is submitted.
func TestIntegrationScaResolverNoScanWithoutSbomFirst(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.BranchFlag), "dummy_branch",
		flag(params.NoScanFlag),
	}

	err, _ := executeCommand(t, args...)
	assertError(
		t,
		err,
		"--no-scan flag was passed without --sbom-first: No SBOM was generated and the CxOne scan was skipped. "+
			"Submit --sbom-first under --sca-resolver-params to generate an SBOM.",
	)
}

// --no-scan + --sbom-first (default location): SBOM saved to <source-dir>/cx-sbom.json, no scan submitted.
func TestIntegrationScaResolverNoScanWithSbomFirst_DefaultLocation(t *testing.T) {
	resolverPath := getScaResolverExecutable(t)

	absDir, absErr := filepath.Abs(Dir)
	assert.NilError(t, absErr)
	expectedSbomPath := filepath.Clean(filepath.Join(absDir, "cx-sbom.json"))
	defer func() { _ = os.Remove(expectedSbomPath) }()

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.ScaResolverFlag), resolverPath,
		flag(params.ScaResolverParamsFlag), "--sbom-first",
		flag(params.ScanTypes), "sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.NoScanFlag),
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "scan create with --no-scan + --sbom-first (default location) should succeed")

	logText := buf.String()
	assert.Assert(
		t,
		strings.Contains(logText, "Resolved packages information was saved"),
		"Expected ScaResolver success message not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "SBOM generated and saved to: "+expectedSbomPath),
		"Expected SBOM generation confirmation at the default location not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "--no-scan set: skipping source compression and upload."),
		"Expected source compression/upload skip message not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "--no-scan set: skipping scan submission."),
		"Expected scan submission skip message not found in logs",
	)

	_, statErr := os.Stat(expectedSbomPath)
	assert.NilError(t, statErr, "SBOM file should actually exist at the default location on disk")
}

// --no-scan + --sbom-first with custom --sbom-output-path/--sbom-output-name: SBOM saved to the custom location, no scan submitted.
func TestIntegrationScaResolverNoScanWithSbomFirst_CustomOutputPathAndName(t *testing.T) {
	resolverPath := getScaResolverExecutable(t)

	// Subdir under the source dir, not t.TempDir(), to avoid OS temp-folder quirks.
	absSourceDir, absErr := filepath.Abs(Dir)
	assert.NilError(t, absErr)
	outputDir := filepath.Join(absSourceDir, "custom-sbom-output")
	assert.NilError(t, os.MkdirAll(outputDir, 0o755))
	defer func() { _ = os.RemoveAll(outputDir) }()

	const customSbomName = "my-project-sbom.json"
	expectedSbomPath := filepath.Clean(filepath.Join(outputDir, customSbomName))

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.ScaResolverFlag), resolverPath,
		flag(params.ScaResolverParamsFlag),
		"--sbom-first --sbom-output-path " + outputDir + " --sbom-output-name " + customSbomName,
		flag(params.ScanTypes), "sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.NoScanFlag),
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "scan create with --no-scan + custom sbom output path/name should succeed")

	logText := buf.String()
	assert.Assert(
		t,
		strings.Contains(logText, "Resolved packages information was saved"),
		"Expected ScaResolver success message not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "SBOM generated and saved to: "+expectedSbomPath),
		"Expected SBOM generation confirmation at the custom path/name not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "--no-scan set: skipping source compression and upload."),
		"Expected source compression/upload skip message not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "--no-scan set: skipping scan submission."),
		"Expected scan submission skip message not found in logs",
	)

	_, statErr := os.Stat(expectedSbomPath)
	assert.NilError(t, statErr, "SBOM file should actually exist at the custom output path/name on disk")
}

// --sbom-first without --no-scan: SBOM is generated and the CxOne scan still runs normally.
func TestIntegrationScaResolverSbomFirstWithoutNoScan(t *testing.T) {
	resolverPath := getScaResolverExecutable(t)

	absDir, absErr := filepath.Abs(Dir)
	assert.NilError(t, absErr)
	expectedSbomPath := filepath.Clean(filepath.Join(absDir, "cx-sbom.json"))
	defer func() { _ = os.Remove(expectedSbomPath) }()

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.ScaResolverFlag), resolverPath,
		flag(params.ScaResolverParamsFlag), "--sbom-first",
		flag(params.ScanTypes), "sca",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "scan create with --sbom-first (no --no-scan) should succeed and submit a scan")

	logText := buf.String()
	assert.Assert(
		t,
		strings.Contains(logText, "Resolved packages information was saved"),
		"Expected ScaResolver success message not found in logs",
	)
	assert.Assert(
		t,
		strings.Contains(logText, "SBOM generated and saved to: "+expectedSbomPath),
		"Expected SBOM generation confirmation not found in logs",
	)
	assert.Assert(
		t,
		!strings.Contains(logText, "--no-scan set"),
		"scan submission/upload should NOT be skipped when --no-scan is not passed",
	)

	_, statErr := os.Stat(expectedSbomPath)
	assert.NilError(t, statErr, "SBOM file should actually exist on disk even though the scan was also submitted")
}

