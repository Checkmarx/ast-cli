//go:build !integration

package commands

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	syftExtractor "github.com/Checkmarx/containers-syft-packages-extractor/pkg/syftPackagesExtractor"
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

// An empty entries array is valid JSON but carries nothing to report.
func TestReportUnresolvedContainerImages_EmptyEntriesListIsNotAnError(t *testing.T) {
	assert.NilError(t, reportUnresolvedContainerImages(writeResolution(t, "[]")))
}

// A Resolved entry sitting alongside a Failed one must never leak into the failure report.
func TestReportUnresolvedContainerImages_ResolvedEntriesAreSkippedAmongFailures(t *testing.T) {
	mixedStatuses := `[
		{"ContainerImage":{"ImageName":"good","ImageTag":"1","ImageLocations":[{"Origin":"UserInput"}],"status":"Resolved"},"ContainerPackages":[{"Name":"x"}]},
		{"ContainerImage":{"ImageName":"bad","ImageTag":"2","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed","ScanError":"boom"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, mixedStatuses))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "bad:2")
	assert.Assert(t, !strings.Contains(err.Error(), "good:1"), "a Resolved entry must never appear in the failure report")
}

// An entry reached through no location at all is not user-requested and must only warn.
func TestReportUnresolvedContainerImages_NoLocationsIsTreatedAsDiscovered(t *testing.T) {
	noLocations := `[{"ContainerImage":{"ImageName":"orphan","ImageTag":"1","ImageLocations":[],"status":"Failed","ScanError":"boom"},"ContainerPackages":[]}]`

	assert.NilError(t, reportUnresolvedContainerImages(writeResolution(t, noLocations)),
		"an entry with no locations at all must not be treated as user-requested")
}

// Status and Origin values must match case-insensitively - nothing in the ticket or the resolver's
// contract guarantees the exact casing the two payloads observed ("Failed"/"UserInput") is the only one.
func TestReportUnresolvedContainerImages_StatusAndOriginAreCaseInsensitive(t *testing.T) {
	upperCase := `[{"ContainerImage":{"ImageName":"case-test","ImageTag":"1","ImageLocations":[{"Origin":"USERINPUT"}],"status":"FAILED","ScanError":"boom"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, upperCase))
	assert.Assert(t, err != nil, "case differences in status/origin must not hide a requested failure")
	assert.ErrorContains(t, err, "case-test:1")
}

// The discovered-only warning message is user-facing output, not just a nil-error contract - assert
// its exact wording so a refactor can't silently drop it while still returning nil.
func TestReportUnresolvedContainerImages_DiscoveredWarningIsLogged(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	err := reportUnresolvedContainerImages(writeResolution(t, discoveredOnlyPayload))

	assert.NilError(t, err)
	loggedMsg := logBuffer.String()
	assert.Assert(t, strings.Contains(loggedMsg, "WARNING"))
	assert.Assert(t, strings.Contains(loggedMsg, "1 container image"))
	assert.Assert(t, strings.Contains(loggedMsg, "was NOT scanned"))
	assert.Assert(t, strings.Contains(loggedMsg, "internal/app:1.0"))
}

// Plural wording must be used once there is more than one discovered-only failure.
func TestReportUnresolvedContainerImages_DiscoveredWarningPluralWording(t *testing.T) {
	twoDiscovered := `[
		{"ContainerImage":{"ImageName":"a","ImageTag":"1","ImageLocations":[{"Origin":"Dockerfile"}],"status":"Failed","ScanError":"x"},"ContainerPackages":[]},
		{"ContainerImage":{"ImageName":"b","ImageTag":"2","ImageLocations":[{"Origin":"Dockerfile"}],"status":"Failed","ScanError":"y"},"ContainerPackages":[]}]`

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	err := reportUnresolvedContainerImages(writeResolution(t, twoDiscovered))

	assert.NilError(t, err, "discovered-only failures must never fail the scan, however many there are")
	loggedMsg := logBuffer.String()
	assert.Assert(t, strings.Contains(loggedMsg, "2 container images"))
	assert.Assert(t, strings.Contains(loggedMsg, "were NOT scanned"))
}

// A resolution file can carry both kinds of failure at once: the discovered one must still only
// warn while the requested one fails the scan, and the two must not bleed into each other's message.
func TestReportUnresolvedContainerImages_RequestedAndDiscoveredTogether(t *testing.T) {
	mixed := `[
		{"ContainerImage":{"ImageName":"requested-img","ImageTag":"1","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed","ScanError":"boom"},"ContainerPackages":[]},
		{"ContainerImage":{"ImageName":"discovered-img","ImageTag":"2","ImageLocations":[{"Origin":"Dockerfile"}],"status":"Failed","ScanError":"boom2"},"ContainerPackages":[]}]`

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	err := reportUnresolvedContainerImages(writeResolution(t, mixed))

	assert.Assert(t, err != nil, "a requested failure must fail the scan even alongside a merely-discovered one")
	assert.ErrorContains(t, err, "requested-img:1")
	assert.Assert(t, !strings.Contains(err.Error(), "discovered-img"), "the discovered image must not appear in the fatal error")
	assert.Assert(t, strings.Contains(logBuffer.String(), "discovered-img:2"), "the discovered image must still be warned about")
}

// An image named without a tag must be displayed by name alone, with no dangling colon.
func TestReportUnresolvedContainerImages_EmptyTagDisplaysNameOnly(t *testing.T) {
	noTag := `[{"ContainerImage":{"ImageName":"registry.example.com/no-tag-image","ImageTag":"","ImageLocations":[{"Origin":"UserInput"}],"status":"Failed","ScanError":"boom"},"ContainerPackages":[]}]`

	err := reportUnresolvedContainerImages(writeResolution(t, noTag))
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "registry.example.com/no-tag-image -")
	assert.Assert(t, !strings.Contains(err.Error(), "registry.example.com/no-tag-image:"),
		"an empty tag must not produce a trailing colon")
}

// isUserRequestedContainerImage exercised directly, independent of file/JSON plumbing.
func TestIsUserRequestedContainerImage(t *testing.T) {
	tests := []struct {
		name  string
		entry syftExtractor.ContainerResolution
		want  bool
	}{
		{
			name:  "no locations at all",
			entry: syftExtractor.ContainerResolution{},
			want:  false,
		},
		{
			name: "single UserInput location",
			entry: syftExtractor.ContainerResolution{ContainerImage: syftExtractor.ContainerImage{
				ImageLocations: []syftExtractor.ImageLocation{{Origin: "UserInput"}},
			}},
			want: true,
		},
		{
			name: "single Dockerfile location",
			entry: syftExtractor.ContainerResolution{ContainerImage: syftExtractor.ContainerImage{
				ImageLocations: []syftExtractor.ImageLocation{{Origin: "Dockerfile"}},
			}},
			want: false,
		},
		{
			name: "origin match is case-insensitive",
			entry: syftExtractor.ContainerResolution{ContainerImage: syftExtractor.ContainerImage{
				ImageLocations: []syftExtractor.ImageLocation{{Origin: "userinput"}},
			}},
			want: true,
		},
		{
			name: "UserInput among several locations",
			entry: syftExtractor.ContainerResolution{ContainerImage: syftExtractor.ContainerImage{
				ImageLocations: []syftExtractor.ImageLocation{{Origin: "Dockerfile"}, {Origin: "UserInput"}},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, isUserRequestedContainerImage(&tt.entry), tt.want)
		})
	}
}

// The small formatting helpers behind the report message, exercised directly for every branch.
func TestContainerImageDisplayName(t *testing.T) {
	assert.Equal(t, containerImageDisplayName("nginx", "alpine"), "nginx:alpine")
	assert.Equal(t, containerImageDisplayName("nginx", ""), "nginx")
}

func TestContainerImageFailureReason(t *testing.T) {
	assert.Equal(t, containerImageFailureReason(""), "the image could not be resolved")
	assert.Equal(t, containerImageFailureReason("custom reason"), "custom reason")
}

func TestContainerImageCount(t *testing.T) {
	assert.Equal(t, containerImageCount(1), "1 container image")
	assert.Equal(t, containerImageCount(2), "2 container images")
	assert.Equal(t, containerImageCount(0), "0 container images")
}

func TestWasOrWere(t *testing.T) {
	assert.Equal(t, wasOrWere(1), "was")
	assert.Equal(t, wasOrWere(2), "were")
	assert.Equal(t, wasOrWere(0), "were")
}
