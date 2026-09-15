//go:build !integration

package commands

// Regression tests for AST-165915.
//
// The ticket reports two separate defects for the same customer scenario - scanning a locally
// built, single-architecture linux/arm64 image with
// `--container-images docker:<image> --containers-local-resolution`:
//
//  1. container image analysis was pinned to linux/amd64, so a single-arch arm64 image never
//     resolved. Fixed in containers-syft-packages-extractor v1.0.27 (no platform is forced) and
//     containers-resolver v1.0.37 (Resolve calls AnalyzeImages again).
//  2. the resolution failure was swallowed: the CLI uploaded an empty resolution, the scan was
//     marked Completed with 0 findings and the pipeline exited 0, indistinguishable from a
//     genuinely clean scan. Fixed here, in runContainerResolver.
//
// Defect 1 lives entirely in the libraries, so what is asserted here is the CLI half: a resolved
// image must let the scan proceed, and an unresolved one must stop it. The payloads below were
// captured from real containers-resolver runs against a real Docker daemon, not hand-written.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

// resolvedArm64Payload is what containers-resolver writes for a locally built single-architecture
// linux/arm64 image once the platform fix is in place. Captured verbatim (packages truncated) from
// a live run on an arm64 Docker host - before the fix this same image produced a Failed entry.
const resolvedArm64Payload = `[{
  "ContainerImage": {
   "ImageName": "docker:ast165915-local",
   "ImageTag": "arm64",
   "Distribution": "alpine:3.20.10",
   "ImageLocations": [{"Origin": "UserInput", "Path": "Custom Images", "FinalStage": false}],
   "status": "Resolved"
  },
  "ContainerPackages": [{"Name": "musl", "Version": "1.2.5-r1"}]
 }]`

// platformMismatchPayload is the customer's own failure, with the message the extractor now maps
// "mismatched platform" errors to. Before the platform fix this is what every arm64 image produced.
const platformMismatchPayload = `[{
  "ContainerImage": {
   "ImageName": "docker:vrif/migration",
   "ImageTag": "0.0.1-32fa28e8",
   "ImageLocations": [{"Origin": "UserInput", "Path": "Custom Images", "FinalStage": false}],
   "status": "Failed",
   "ScanError": "The image architecture does not match the requested platform. Registry: index.docker.io"
  },
  "ContainerPackages": []
 }]`

// badTagPayload is the generic reproduction from case 00286204 on the ticket: a public image with a
// non-existent tag, no arm64 and no private registry involved. Captured verbatim from a live run.
const badTagPayload = `[{
  "ContainerImage": {
   "ImageName": "debian",
   "ImageTag": "non-existent-tag-999",
   "Distribution": "NONE",
   "ImageId": "debian:non-existent-tag-999",
   "ImageLocations": [{"Origin": "UserInput", "Path": "Custom Images", "FinalStage": false}],
   "status": "Failed",
   "ScanError": "The requested image is not found or is unavailable. Registry: index.docker.io"
  },
  "ContainerPackages": []
 }]`

// fakeContainerResolver stands in for containers-resolver and reproduces the behaviour at the heart
// of defect 2: it writes a resolution file that may contain Failed entries and still returns nil.
type fakeContainerResolver struct {
	payload    string
	gotImages  []string
	gotInvoked bool
}

func (f *fakeContainerResolver) Resolve(scanPath, resolutionFolderPath string, images []string, isDebug bool) error {
	f.gotInvoked = true
	f.gotImages = images

	dir := filepath.Join(resolutionFolderPath, ".checkmarx", "containers")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, containerResolutionFileName), []byte(f.payload), 0o600); err != nil {
		return err
	}
	// The real resolver reports per-image failures only inside the file, never through this
	// return value. That is precisely why the CLI has to inspect the file.
	return nil
}

// runTicketScenario drives the real runContainerResolver entry point with a resolver that produces
// the given payload, mirroring `cx scan create --container-images <image> --containers-local-resolution`.
func runTicketScenario(t *testing.T, payload, containerImages string) (*fakeContainerResolver, error) {
	t.Helper()

	fake := &fakeContainerResolver{payload: payload}
	original := containerResolver
	containerResolver = fake
	t.Cleanup(func() { containerResolver = original })

	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")

	return fake, runContainerResolver(cmd, t.TempDir(), containerImages, true)
}

// Defect 1, CLI half: the arm64 image the customer could not scan now resolves, so the scan must be
// allowed to continue without any error.
func TestAST165915_Defect1_LocallyBuiltArm64ImageLetsTheScanProceed(t *testing.T) {
	fake, err := runTicketScenario(t, resolvedArm64Payload, "docker:ast165915-local:arm64")

	assert.NilError(t, err, "a resolved arm64 image must not block the scan")
	assert.Assert(t, fake.gotInvoked, "the resolver must actually be invoked")
	assert.Equal(t, len(fake.gotImages), 1)
	assert.Equal(t, fake.gotImages[0], "docker:ast165915-local:arm64", "the image must reach the resolver unmangled, prefix included")
}

// Defect 2, the ticket's headline scenario: the customer's arm64 image failing to resolve must stop
// the scan instead of completing with 0 findings and exit 0.
func TestAST165915_Defect2_PlatformMismatchStopsTheScan(t *testing.T) {
	_, err := runTicketScenario(t, platformMismatchPayload, "docker:vrif/migration:0.0.1-32fa28e8")

	assert.Assert(t, err != nil, "an unresolved image must fail the scan, not complete silently")
	assert.ErrorContains(t, err, "docker:vrif/migration:0.0.1-32fa28e8")
	assert.ErrorContains(t, err, "NOT scanned")
	assert.ErrorContains(t, err, "The image architecture does not match the requested platform")
}

// Defect 2 is not arm64-specific. This is case 00286204's reproduction: a public image with a bad
// tag has to fail the same way, which is what the ticket asks for ("cover the general case").
func TestAST165915_Defect2_AnyResolutionFailureStopsTheScan(t *testing.T) {
	_, err := runTicketScenario(t, badTagPayload, "debian:non-existent-tag-999")

	assert.Assert(t, err != nil, "a generic resolution failure must fail the scan too")
	assert.ErrorContains(t, err, "debian:non-existent-tag-999")
	assert.ErrorContains(t, err, "The requested image is not found or is unavailable")
}

// The exact regression: a Failed entry in the resolution file must never coexist with a nil error,
// because that combination is what made the scan indistinguishable from a clean one.
func TestAST165915_Defect2_FailedEntryNeverReportsSuccess(t *testing.T) {
	for name, payload := range map[string]string{
		"platform mismatch": platformMismatchPayload,
		"image not found":   badTagPayload,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := runTicketScenario(t, payload, "some-image:tag")
			assert.Assert(t, err != nil, "a Failed resolution entry must never be reported as success")
		})
	}
}

// Guards the one case that must keep the old lenient behaviour: an image discovered in the scanned
// sources rather than requested by name only warns, as AST-146648 deliberately chose.
func TestAST165915_DiscoveredImageStillOnlyWarns(t *testing.T) {
	discovered := `[{"ContainerImage":{"ImageName":"internal/app","ImageTag":"1.0",
		"ImageLocations":[{"Origin":"Dockerfile","Path":"/src/Dockerfile"}],
		"status":"Failed","ScanError":"The requested image is not found or is unavailable."},
		"ContainerPackages":[]}]`

	_, err := runTicketScenario(t, discovered, "")
	assert.NilError(t, err, "an image only discovered in the sources must not fail the scan (AST-146648)")
}

// Compile-time proof the fake honours the interface the CLI actually injects.
var _ wrappers.ContainerResolverWrapper = &fakeContainerResolver{}
