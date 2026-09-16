//go:build !integration

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

// realFailedResolution is the resolution file containers-resolver actually writes for
// `--container-images debian:non-existent-tag-999`, captured verbatim from a live run. Using the
// real payload keeps this test honest about the JSON shape (note the lower-case "status" key).
const realFailedResolution = `[
 {
  "ContainerImage": {
   "ImageName": "debian",
   "ImageTag": "non-existent-tag-999",
   "Distribution": "NONE",
   "ImageHash": "",
   "ImageId": "debian:non-existent-tag-999",
   "ImageLocations": [
    {"Origin": "UserInput", "Path": "Custom Images", "FinalStage": false}
   ],
   "Layers": [],
   "History": [],
   "status": "Failed",
   "ScanError": "The requested image is not found or is unavailable. Registry: index.docker.io"
  },
  "ContainerPackages": []
 }
]`

// discoveredOnlyPayload is a Failed entry reached only through Dockerfile discovery, never named
// explicitly by the user - the case AST-146648 deliberately chose to only warn about.
const discoveredOnlyPayload = `[{"ContainerImage":{"ImageName":"internal/app","ImageTag":"1.0",
		"ImageLocations":[{"Origin":"Dockerfile","Path":"/src/Dockerfile"}],
		"status":"Failed","ScanError":"The requested image is not found or is unavailable."},
		"ContainerPackages":[]}]`

func writeResolution(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	resolutionDir := filepath.Join(dir, ".checkmarx", "containers")
	assert.NilError(t, os.MkdirAll(resolutionDir, 0o750))
	assert.NilError(t, os.WriteFile(filepath.Join(resolutionDir, containerResolutionFileName), []byte(content), 0o600))
	return dir
}

// An image the user named with --container-images must fail the scan: the pipeline asked for it by
// name, so completing with 0 findings would be indistinguishable from a clean scan (AST-165915).
func TestReportUnresolvedContainerImages_UserRequestedImageFails(t *testing.T) {
	err := reportUnresolvedContainerImages(writeResolution(t, realFailedResolution))

	assert.Assert(t, err != nil, "a user-requested image that failed to resolve must return an error")
	assert.ErrorContains(t, err, "debian:non-existent-tag-999")
	assert.ErrorContains(t, err, "NOT scanned")
	assert.ErrorContains(t, err, "The requested image is not found or is unavailable")
}

// An image only discovered inside the scanned sources warns but does not fail, preserving the
// warn-rather-than-fail behaviour chosen in AST-146648 for images the CLI cannot reach.
func TestReportUnresolvedContainerImages_DiscoveredImageOnlyWarns(t *testing.T) {
	assert.NilError(t, reportUnresolvedContainerImages(writeResolution(t, discoveredOnlyPayload)))
}

// An image reachable both ways is still the user's explicit request, so it fails.
func TestReportUnresolvedContainerImages_MixedOriginCountsAsRequested(t *testing.T) {
	mixed := `[{"ContainerImage":{"ImageName":"internal/app","ImageTag":"1.0",
		"ImageLocations":[{"Origin":"Dockerfile","Path":"/src/Dockerfile"},{"Origin":"UserInput","Path":"Custom Images"}],
		"status":"Failed","ScanError":"boom"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, mixed))
	assert.Assert(t, err != nil, "an image also named by the user must fail the scan")
	assert.ErrorContains(t, err, "internal/app:1.0")
}

// The arm64 case from the ticket, once the platform fix is in: resolved images must not be reported.
func TestReportUnresolvedContainerImages_ResolvedImageIsSilent(t *testing.T) {
	resolved := `[{"ContainerImage":{"ImageName":"docker:ast165915-local","ImageTag":"arm64",
		"ImageLocations":[{"Origin":"UserInput","Path":"Custom Images"}],"status":"Resolved"},
		"ContainerPackages":[{"Name":"musl"}]}]`

	assert.NilError(t, reportUnresolvedContainerImages(writeResolution(t, resolved)))
}

// A missing or unreadable resolution file must not invent a failure - Resolve already reports any
// failure of the resolution step itself through its return value.
func TestReportUnresolvedContainerImages_MissingFileIsNotAnError(t *testing.T) {
	assert.NilError(t, reportUnresolvedContainerImages(t.TempDir()))
}

func TestReportUnresolvedContainerImages_UnparsableFileIsNotAnError(t *testing.T) {
	assert.NilError(t, reportUnresolvedContainerImages(writeResolution(t, "not json at all")))
}

// Several failures are reported together rather than one at a time.
func TestReportUnresolvedContainerImages_AggregatesMultipleFailures(t *testing.T) {
	multiple := `[
		{"ContainerImage":{"ImageName":"a","ImageTag":"1","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed","ScanError":"first"},"ContainerPackages":[]},
		{"ContainerImage":{"ImageName":"b","ImageTag":"2","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed","ScanError":"second"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, multiple))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "2 container images")
	assert.ErrorContains(t, err, "a:1")
	assert.ErrorContains(t, err, "b:2")
}

// An image with no ScanError still has to be named, not reported as an empty reason.
func TestReportUnresolvedContainerImages_FailureWithoutScanErrorStillReported(t *testing.T) {
	noReason := `[{"ContainerImage":{"ImageName":"c","ImageTag":"3","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, noReason))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "c:3")
	assert.ErrorContains(t, err, "the image could not be resolved")
}
