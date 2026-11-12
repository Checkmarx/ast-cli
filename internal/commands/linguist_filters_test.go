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

func TestFilterMatchedMatchesEveryBaseIncludeFilter(t *testing.T) {
	for _, f := range commonParams.BaseIncludeFilters {
		name := strings.NewReplacer("*", "x", "?", "y").Replace(f)
		assert.Assert(t, filterMatched(commonParams.BaseIncludeFilters, name), "file %q should match filter %q", name, f)
		assert.Assert(t, filterMatched(commonParams.BaseIncludeFilters, strings.ToUpper(name)),
			"file %q should match filter %q case-insensitively", strings.ToUpper(name), f)
	}
}

func TestFilterMatchedLinguistFiles(t *testing.T) {
	included := []string{"deploy.sh", "build.PS1", "run.bat", "notebook.ipynb", "tmpl.j2", "Jenkinsfile", "Makefile", "GNUmakefile",
		"Containerfile", "Rakefile", ".bash_history", "BUILD.bazel", "Deploy.SH", "JENKINSFILE"}
	for _, name := range included {
		assert.Assert(t, filterMatched(commonParams.BaseIncludeFilters, name), "%q should be included", name)
	}
	excluded := []string{"logo.png", "archive.zip", "makefile.bak", "jenkinsfile.txt", "data.bin", "sh"}
	for _, name := range excluded {
		assert.Assert(t, !filterMatched(commonParams.BaseIncludeFilters, name), "%q should not be included", name)
	}
}

func TestFilterMatchedExcludeWinsOverLinguistInclude(t *testing.T) {
	filters := append([]string{"!*.min.sh"}, commonParams.BaseIncludeFilters...)
	assert.Assert(t, filterMatched(filters, "deploy.sh"))
	assert.Assert(t, !filterMatched(filters, "deploy.min.sh"), "exclude filter must win over Linguist include")
	assert.Assert(t, !filterMatched(commonParams.BaseExcludeFilters, ".vscode"))
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

	err := addDirFiles(zw, "", root+string(os.PathSeparator), commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters,
		filtering.NopMatcher{}, false)
	assert.NilError(t, err)
	assert.NilError(t, zw.Close())

	assert.DeepEqual(t, []string{"Jenkinsfile", "Main.java", "Makefile", "build.ps1", "deploy.sh", "scripts/run.bat"}, zipEntryNames(t, out.Bytes()))
}

func TestIsDirEmptyWithLinguistFiles(t *testing.T) {
	onlyLinguist := writeLinguistTree(t, []string{"logo.png", "ci/deploy.sh"})
	empty, err := isDirEmpty(onlyLinguist, commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters, filtering.NopMatcher{})
	assert.NilError(t, err)
	assert.Assert(t, !empty, "a Linguist-only file must count as scannable content")

	onlyBinary := writeLinguistTree(t, []string{"logo.png", "data.bin", "node_modules/dep.sh"})
	empty, err = isDirEmpty(onlyBinary, commonParams.BaseExcludeFilters, commonParams.BaseIncludeFilters, filtering.NopMatcher{})
	assert.NilError(t, err)
	assert.Assert(t, empty, "files outside the filters and excluded directories must not count")
}
