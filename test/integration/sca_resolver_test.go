//go:build integration

package integration

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/scarealtime/scaconfig"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"gotest.tools/assert"
)

// scaResolverWorkingDirName is deliberately separate from scaconfig.Params.WorkingDirName
// so downloading/cleaning up the executable for these tests never touches the cache used
// by the sca-realtime tests (which share the process-wide scaconfig.Params working dir).
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

		_, scaResolverErr = osinstaller.InstallOrUpgrade(&scaResolverConfig)
		if scaResolverErr == nil {
			scaResolverPath = scaResolverConfig.ExecutableFilePath()
		}
	})

	if scaResolverErr != nil {
		t.Fatalf("Failed to download ScaResolver executable: %v", scaResolverErr)
	}

	return scaResolverPath
}

// TestScaResolverExecutable_Success runs a real
// `cx scan create --sca-resolver <path> ...` using the actual downloaded ScaResolver
// executable, rather than a path pre-staged outside the test.
func TestScaResolverExecutable_Success(t *testing.T) {
	resolverPath := getScaResolverExecutable(t)

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), Dir,
		flag(params.ScaResolverFlag), resolverPath,
		flag(params.ScaResolverParamsFlag), "-q",
		flag(params.ScanTypes), "iac-security,sca",
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
func TestCreateScanNoScanWithoutSbomFirst(t *testing.T) {
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

// TestZZZCleanupScaResolverExecutable removes the downloaded executable after every other
// test in this file has run. It must stay the last test declared in this file, since Go
// runs tests within a file in source declaration order. It is a no-op unless
// getScaResolverExecutable actually triggered a download during this run.
func TestZZZCleanupScaResolverExecutable(t *testing.T) {
	if scaResolverPath == "" {
		return
	}
	assert.NilError(t, os.RemoveAll(scaResolverConfig.WorkingDir()))
}
