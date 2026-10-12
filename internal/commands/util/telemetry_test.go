package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendRealtimeScanTelemetry_NilWrapper(t *testing.T) {
	assert.NotPanics(t, func() {
		SendRealtimeScanTelemetry(nil, "Oss", "oss", "package.json", 1)
	})
}

func TestSendRealtimeScanTelemetry_IncludesGitHubURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	gitIn("init")
	gitIn("remote", "add", "origin", "git@github.com:acme/widgets.git")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o600))

	var got *wrappers.DataForAITelemetry
	tel := mock.TelemetryMockWrapper{
		CustomSendAIDataToLog: func(data *wrappers.DataForAITelemetry) error {
			got = data
			return nil
		},
	}
	SendRealtimeScanTelemetry(tel, "Oss", "oss", filepath.Join(dir, "package.json"), 4)
	require.NotNil(t, got)
	assert.Equal(t, "https://github.com/acme/widgets", got.GithubURL)
	assert.Equal(t, "Oss", got.Engine)
	assert.Equal(t, "oss", got.ScanType)
	assert.Equal(t, "scan", got.Type)
	assert.Equal(t, 4, got.TotalCount)
}
