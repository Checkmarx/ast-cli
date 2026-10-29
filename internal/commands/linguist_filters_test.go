//go:build !integration

package commands

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/filtering"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestFileMatcherMatchesEveryBaseIncludeFilter(t *testing.T) {
	m := newFileMatcher(commonParams.BaseIncludeFilters)
	for _, f := range commonParams.BaseIncludeFilters {
		name := strings.NewReplacer("*", "x", "?", "y").Replace(f)
		assert.Assert(t, m.Matched(name), "file %q should match filter %q", name, f)
		assert.Assert(t, m.Matched(strings.ToUpper(name)), "file %q should match filter %q case-insensitively", strings.ToUpper(name), f)
	}
}

func TestFileMatcherLinguistFiles(t *testing.T) {
	m := newFileMatcher(commonParams.BaseIncludeFilters)
	included := []string{"deploy.sh", "build.PS1", "run.bat", "notebook.ipynb", "tmpl.j2", "Jenkinsfile", "Makefile", "GNUmakefile",
		"Containerfile", "Rakefile", ".bash_history", "BUILD.bazel"}
	for _, name := range included {
		assert.Assert(t, m.Matched(name), "%q should be included", name)
	}
	excluded := []string{"logo.png", "archive.zip", "makefile.bak", "jenkinsfile.txt", "data.bin", "sh"}
	for _, name := range excluded {
		assert.Assert(t, !m.Matched(name), "%q should not be included", name)
	}
}

// writeLinguistTree builds a source tree mixing Linguist files, SAST files and files that must be skipped.
func writeLinguistTree(t *testing.T, files []string) string {
	t.Helper()
	root := t.TempDir()
	for _, rel := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		assert.NilError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NilError(t, os.WriteFile(p, []byte("content of "+rel), 0o600))
	}
	return root
}

func zipEntryNames(t *testing.T, data []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	assert.NilError(t, err)
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func TestAddDirFilesIncludesLinguistFiles(t *testing.T) {
	root := writeLinguistTree(t, []string{
		"deploy.sh", "Jenkinsfile", "Makefile", "build.ps1", "Main.java", "scripts/run.bat",
		"logo.png", "data.bin", "node_modules/dep.sh", ".idea/config.sh",
	})
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	walker := newZipWalker(zw, commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters, filtering.NopMatcher{}, false, false)

	assert.NilError(t, addDirFiles(walker, "", root+string(os.PathSeparator)))
	assert.NilError(t, zw.Close())

	assert.DeepEqual(t, []string{"Jenkinsfile", "Main.java", "Makefile", "build.ps1", "deploy.sh", "scripts/run.bat"}, zipEntryNames(t, out.Bytes()))
}

func TestIsDirEmptyWithLinguistFiles(t *testing.T) {
	newWalker := func() *zipWalker {
		return newZipWalker(zip.NewWriter(&bytes.Buffer{}), commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters, filtering.NopMatcher{}, false, false)
	}

	onlyLinguist := writeLinguistTree(t, []string{"logo.png", "ci/deploy.sh"})
	empty, err := isDirEmpty(onlyLinguist, newWalker())
	assert.NilError(t, err)
	assert.Assert(t, !empty, "a Linguist-only file must count as scannable content")

	onlyBinary := writeLinguistTree(t, []string{"logo.png", "data.bin", "node_modules/dep.sh"})
	empty, err = isDirEmpty(onlyBinary, newWalker())
	assert.NilError(t, err)
	assert.Assert(t, empty, "files outside the filters and excluded directories must not count")
}

func TestZipWalkerFileIncluded(t *testing.T) {
	walker := newZipWalker(zip.NewWriter(&bytes.Buffer{}), []string{"!*.min.sh"}, commonParams.BaseIncludeFilters, filtering.NopMatcher{}, false, false)
	assert.Assert(t, walker.fileIncluded("deploy.sh", "ci/deploy.sh"))
	assert.Assert(t, !walker.fileIncluded("deploy.min.sh", "ci/deploy.min.sh"), "exclude filter must win over Linguist include")
	assert.Assert(t, !walker.fileIncluded("logo.png", "logo.png"))
}

func TestFilterMatchedLinguistFilters(t *testing.T) {
	assert.Assert(t, filterMatched(commonParams.BaseIncludeFilters, "Deploy.SH"))
	assert.Assert(t, filterMatched(commonParams.BaseIncludeFilters, "JENKINSFILE"))
	assert.Assert(t, !filterMatched(commonParams.BaseIncludeFilters, "logo.png"))
	assert.Assert(t, !filterMatched(commonParams.BaseExcludeFilters, ".vscode"))
}
