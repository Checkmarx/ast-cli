package governance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationWindow_DefaultsToTwoMinutes(t *testing.T) {
	t.Setenv(correlationWindowEnv, "")
	assert.Equal(t, 2*time.Minute, correlationWindow())
}

func TestCorrelationWindow_HonoursOverride(t *testing.T) {
	t.Setenv(correlationWindowEnv, "10s")
	assert.Equal(t, 10*time.Second, correlationWindow())
}

func TestCorrelationWindow_FallsBackOnGarbage(t *testing.T) {
	// A typo in the env var must not collapse the window to zero, which would
	// make every in-flight tool call look blocked.
	t.Setenv(correlationWindowEnv, "not-a-duration")
	assert.Equal(t, 2*time.Minute, correlationWindow())

	t.Setenv(correlationWindowEnv, "-5s")
	assert.Equal(t, 2*time.Minute, correlationWindow())
}

// writeMarker creates a correlation marker with a chosen age.
func writeMarker(t *testing.T, dir, name string, age time.Duration) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))
	stamp := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(path, stamp, stamp))
	return path
}

func TestReapStrayPost_RemovesOrphanedPost(t *testing.T) {
	dir := t.TempDir()
	writeMarker(t, dir, "abc.post", time.Hour)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	reapStrayPost(dir, entries[0], time.Minute)

	_, err = os.Stat(filepath.Join(dir, "abc.post"))
	assert.True(t, os.IsNotExist(err), "a .post with no .pre partner should be reaped")
}

func TestReapStrayPost_KeepsPostWithLivePartner(t *testing.T) {
	dir := t.TempDir()
	writeMarker(t, dir, "abc.post", time.Hour)
	writeMarker(t, dir, "abc.pre.json", time.Hour)

	for _, e := range mustReadDir(t, dir) {
		if e.Name() == "abc.post" {
			reapStrayPost(dir, e, time.Minute)
		}
	}

	_, err := os.Stat(filepath.Join(dir, "abc.post"))
	assert.NoError(t, err, "a .post whose .pre is still present must survive for normal pairing")
}

func TestReapStrayPost_KeepsRecentPost(t *testing.T) {
	// A .post younger than the window may simply be racing its own sweep.
	dir := t.TempDir()
	writeMarker(t, dir, "abc.post", time.Second)

	entries := mustReadDir(t, dir)
	reapStrayPost(dir, entries[0], time.Minute)

	_, err := os.Stat(filepath.Join(dir, "abc.post"))
	assert.NoError(t, err, "a fresh stray .post must not be reaped yet")
}

func mustReadDir(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	return entries
}
