package configfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gofrs/flock"
	"github.com/stretchr/testify/assert"
)

func TestLoadMissingFileReturnsEmptyMap(t *testing.T) {
	config, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	assert.NoError(t, err)
	assert.Empty(t, config)
}

func TestLoadZeroByteFileReturnsEmptyMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.yaml")
	assert.NoError(t, os.WriteFile(path, nil, 0o600))

	config, err := Load(path)
	assert.NoError(t, err)
	assert.Empty(t, config)
}

func TestLoadCommentOnlyFileReturnsEmptyMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comments.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("# nothing but a comment\n"), 0o600))

	config, err := Load(path)
	assert.NoError(t, err)
	assert.Empty(t, config)
}

func TestLoadCorruptYamlReturnsNilMapAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("foo: [bar"), 0o600))

	config, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error decoding YAML")
	assert.Nil(t, config)
}

func TestSetKeyRoundTripAndOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	assert.NoError(t, SetKey(path, "cx_apikey", "first"))
	config, err := Load(path)
	assert.NoError(t, err)
	assert.Equal(t, "first", config["cx_apikey"])

	assert.NoError(t, SetKey(path, "cx_apikey", "second"))
	assert.NoError(t, SetKey(path, "cx_tenant", "qa"))
	config, err = Load(path)
	assert.NoError(t, err)
	assert.Equal(t, "second", config["cx_apikey"])
	assert.Equal(t, "qa", config["cx_tenant"])
}

func TestRemoveKeyRemovesOnlyTargetKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	assert.NoError(t, SetKey(path, "cx_apikey", "gone"))
	assert.NoError(t, SetKey(path, "cx_base_uri", "https://keep"))

	assert.NoError(t, RemoveKey(path, "cx_apikey"))
	config, err := Load(path)
	assert.NoError(t, err)
	assert.NotContains(t, config, "cx_apikey")
	assert.Equal(t, "https://keep", config["cx_base_uri"])
}

func TestRemoveKeyAbsentIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	assert.NoError(t, SetKey(path, "cx_branch", "main"))

	assert.NoError(t, RemoveKey(path, "cx_apikey"))
	config, err := Load(path)
	assert.NoError(t, err)
	assert.Equal(t, "main", config["cx_branch"])
}

// Save must not widen permissions on an existing file and must leave no
// temporary file behind.
func TestSavePreservesPermissionsAndCleansTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("cx_apikey: original\n"), 0o600))

	assert.NoError(t, Save(path, map[string]interface{}{"cx_apikey": "updated"}))

	info, err := os.Stat(path)
	assert.NoError(t, err)
	if runtime.GOOS != "windows" {
		// Windows emulates the POSIX read/write bits (any writable file
		// reports 0666), so owner-only persistence is only assertable on Unix.
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "save must keep owner-only permissions")
	}

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp")
	}
}

func TestLoadDirectoryPathReturnsError(t *testing.T) {
	_, err := Load(t.TempDir())
	assert.Error(t, err)
}

func TestSaveInvalidTargetPathReturnsErrorAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "no-such-dir", "config.yaml")
	assert.Error(t, Save(bad, map[string]interface{}{"k": "v"}))
	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSetKeyMissingParentDirFails(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "no-such-dir", "config.yaml")
	assert.Error(t, SetKey(bad, "k", "v"))
}

func TestRemoveKeyMissingParentDirFails(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "no-such-dir", "config.yaml")
	assert.Error(t, RemoveKey(bad, "k"))
}

func TestSetKeyLockHeldByOtherProcessFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	lock := flock.New(path + ".lock")
	locked, lockErr := lock.TryLock()
	assert.NoError(t, lockErr)
	assert.True(t, locked)
	t.Cleanup(func() { _ = lock.Unlock() })

	assert.ErrorContains(t, SetKey(path, "k", "v"), "held by another process")
}

func TestRemoveKeyLockHeldByOtherProcessFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	lock := flock.New(path + ".lock")
	locked, lockErr := lock.TryLock()
	assert.NoError(t, lockErr)
	assert.True(t, locked)
	t.Cleanup(func() { _ = lock.Unlock() })

	assert.ErrorContains(t, RemoveKey(path, "k"), "held by another process")
}
