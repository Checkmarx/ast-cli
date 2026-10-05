//go:build !integration

package commands

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestRunScanContainersRealtimeCommand_EmptyFilePath_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(t, "scan", "containers-realtime", "-s", "")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "file path is required") ||
		strings.Contains(err.Error(), "realtime engine error"),
		"unexpected error: %v", err)
}

func TestRunScanContainersRealtimeCommand_MissingSourcesFlag_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(t, "scan", "containers-realtime")
	assert.NotNil(t, err)
}

func TestRunScanContainersRealtimeCommand_Dockerfile_Success(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(t, "scan", "containers-realtime", "-s", "data/Dockerfile")
}

func TestRunScanContainersRealtimeCommand_ContainersTestdata_Success(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(t, "scan", "containers-realtime", "-s", "data/containers/testdata/Dockerfile")
}

func TestRunScanContainersRealtimeCommand_MissingFile_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(t, "scan", "containers-realtime", "-s", "data/does-not-exist-Dockerfile")
	assert.NotNil(t, err)
}

func TestRunScanContainersRealtimeCommand_WithIgnoredFilePathFlag(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	// Empty/missing ignore file should still succeed (service fail-opens on ignore load).
	execCmdNilAssertion(t,
		"scan", "containers-realtime",
		"-s", "data/Dockerfile",
		"--ignored-file-path", "data/does-not-exist-ignore.json",
	)
}
