package osinstaller

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newInstallConfig creates an InstallationConfiguration with a short, unique
// WorkingDirName (as production configs do, e.g. "CxVorpal") so that
// InstallationConfiguration.WorkingDir() resolves to a valid path on every OS.
// The resolved directory is created and cleaned up automatically.
func newInstallConfig(t *testing.T) *InstallationConfiguration {
	t.Helper()
	name := fmt.Sprintf("cx-cli-test-%d", time.Now().UnixNano())
	cfg := &InstallationConfiguration{
		ExecutableFile: "tool",
		FileName:       "tool.tar.gz",
		HashFileName:   "tool.hash",
		WorkingDirName: name,
	}
	resolved := cfg.WorkingDir()
	require.NoError(t, os.MkdirAll(resolved, 0755))
	t.Cleanup(func() { _ = os.RemoveAll(resolved) })
	return cfg
}

func TestFileExists_ExistingFile_ReturnsTrue(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0600))

	exists, err := FileExists(filePath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestFileExists_NonExistentFile_ReturnsFalse(t *testing.T) {
	exists, err := FileExists(filepath.Join(t.TempDir(), "missing.txt"))
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetHashValue_ValidFile_ReturnsSha256Hash(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	content := []byte("hash-me")
	require.NoError(t, os.WriteFile(filePath, content, 0600))

	expected := sha256.Sum256(content)

	hash, err := getHashValue(filePath)
	assert.NoError(t, err)
	assert.Equal(t, expected[:], hash)
}

func TestGetHashValue_NonExistentFile_ReturnsError(t *testing.T) {
	hash, err := getHashValue(filepath.Join(t.TempDir(), "missing.txt"))
	assert.Error(t, err)
	assert.Nil(t, hash)
}

func TestCreateWorkingDirectory_CreatesDirectory(t *testing.T) {
	name := fmt.Sprintf("cx-cli-test-not-created-yet-%d", time.Now().UnixNano())
	cfg := &InstallationConfiguration{WorkingDirName: name}
	t.Cleanup(func() { _ = os.RemoveAll(cfg.WorkingDir()) })

	err := createWorkingDirectory(cfg)
	assert.NoError(t, err)

	info, statErr := os.Stat(cfg.WorkingDir())
	require.NoError(t, statErr)
	assert.True(t, info.IsDir())
}

func TestDownloadFile_Success_WritesResponseBodyToFile(t *testing.T) {
	const body = "binary-content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	destPath := filepath.Join(t.TempDir(), "downloaded.bin")
	err := downloadFile(server.URL, destPath)
	assert.NoError(t, err)

	content, readErr := os.ReadFile(destPath)
	require.NoError(t, readErr)
	assert.Equal(t, body, string(content))
}

func TestDownloadFile_UnreachableServer_ReturnsError(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "downloaded.bin")
	err := downloadFile("http://127.0.0.1:0/unreachable", destPath)
	assert.Error(t, err)
}

func TestDownloadHashFile_Success_WritesHashFile(t *testing.T) {
	const hashContent = "deadbeef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hashContent))
	}))
	defer server.Close()

	destPath := filepath.Join(t.TempDir(), "tool.hash")
	err := downloadHashFile(server.URL, destPath)
	assert.NoError(t, err)

	content, readErr := os.ReadFile(destPath)
	require.NoError(t, readErr)
	assert.Equal(t, hashContent, string(content))
}

func TestIsLastVersion_HashUnchanged_ReturnsTrue(t *testing.T) {
	const hashContent = "same-hash-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hashContent))
	}))
	defer server.Close()

	hashFilePath := filepath.Join(t.TempDir(), "tool.hash")
	require.NoError(t, os.WriteFile(hashFilePath, []byte(hashContent), 0600))

	upToDate, err := isLastVersion(hashFilePath, server.URL, hashFilePath)
	assert.NoError(t, err)
	assert.True(t, upToDate)
}

func TestIsLastVersion_HashChanged_ReturnsFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new-hash-value"))
	}))
	defer server.Close()

	hashFilePath := filepath.Join(t.TempDir(), "tool.hash")
	require.NoError(t, os.WriteFile(hashFilePath, []byte("old-hash-value"), 0600))

	upToDate, err := isLastVersion(hashFilePath, server.URL, hashFilePath)
	assert.NoError(t, err)
	assert.False(t, upToDate)
}

func TestIsLastVersion_DownloadFails_ReturnsError(t *testing.T) {
	hashFilePath := filepath.Join(t.TempDir(), "tool.hash")
	require.NoError(t, os.WriteFile(hashFilePath, []byte("old-hash-value"), 0600))

	_, err := isLastVersion(hashFilePath, "http://127.0.0.1:0/unreachable", hashFilePath)
	assert.Error(t, err)
}

func TestDownloadNotNeeded_ExecutableMissing_ReturnsFalse(t *testing.T) {
	cfg := newInstallConfig(t)
	assert.False(t, downloadNotNeeded(cfg))
}

func TestDownloadNotNeeded_ExecutableExistsAndUpToDate_ReturnsTrue(t *testing.T) {
	const hashContent = "matching-hash"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hashContent))
	}))
	defer server.Close()

	cfg := newInstallConfig(t)
	cfg.HashDownloadURL = server.URL
	require.NoError(t, os.WriteFile(cfg.ExecutableFilePath(), []byte("exe"), 0755))
	require.NoError(t, os.WriteFile(cfg.HashFilePath(), []byte(hashContent), 0600))

	assert.True(t, downloadNotNeeded(cfg))
}

func TestInstallOrUpgrade_AlreadyUpToDate_ReturnsFalseNil(t *testing.T) {
	const hashContent = "matching-hash"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hashContent))
	}))
	defer server.Close()

	cfg := newInstallConfig(t)
	cfg.HashDownloadURL = server.URL
	require.NoError(t, os.WriteFile(cfg.ExecutableFilePath(), []byte("exe"), 0755))
	require.NoError(t, os.WriteFile(cfg.HashFilePath(), []byte(hashContent), 0600))

	installed, err := InstallOrUpgrade(cfg)
	assert.NoError(t, err)
	assert.False(t, bool(installed))
}

func TestInstallOrUpgrade_DownloadFileFails_ReturnsError(t *testing.T) {
	cfg := newInstallConfig(t)
	cfg.DownloadURL = "http://127.0.0.1:0/unreachable"

	installed, err := InstallOrUpgrade(cfg)
	assert.Error(t, err)
	assert.False(t, bool(installed))
}
