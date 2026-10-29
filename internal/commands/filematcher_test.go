//go:build !integration

package commands

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// legacyFilterMatched is the linear matcher fileMatcher replaced. It is the reference that
// TestFileMatcherEquivalentToLegacy compares against.
func legacyFilterMatched(filters []string, fileName string) bool {
	firstMatch := true
	matched := true
	fileNameLower := strings.ToLower(fileName)
	for _, filter := range filters {
		filterLower := strings.ToLower(filter)
		if filterLower[0] == '!' {
			excluded, _ := path.Match(filterLower[1:], fileNameLower)
			if excluded {
				return false
			}
		} else {
			if firstMatch {
				firstMatch = false
				matched = false
			}
			if !matched {
				matched, _ = path.Match(filterLower, fileNameLower)
			}
		}
	}
	return matched
}

func TestFileMatcher(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		fileName string
		expected bool
	}{
		{"suffix", []string{"*.go"}, "main.go", true},
		{"suffix is case-insensitive", []string{"*.Go"}, "MAIN.GO", true},
		{"suffix no match", []string{"*.go"}, "main.py", false},
		{"multi-part suffix", []string{"*.auto.tfvars"}, "prod.auto.tfvars", true},
		{"short suffix matches multi-dot name", []string{"*.erb"}, "index.js.erb", true},
		{"suffix needs the dot", []string{"*.go"}, "go", false},
		{"dot file matches its suffix", []string{"*.env"}, ".env", true},
		{"exact name", []string{"Makefile"}, "makefile", true},
		{"exact name is not a substring match", []string{"makefile"}, "makefile.am", false},
		{"glob", []string{"*Dockerfile*"}, "prod.dockerfile.bak", true},
		{"glob with infix", []string{"*.php5*6"}, "x.php5_6", true},
		{"exclusion wins over inclusion", []string{"*.js", "!*.min.js"}, "app.min.js", false},
		{"exclusion before inclusion", []string{"!*.min.js", "*.js"}, "app.min.js", false},
		{"only exclusions include everything else", []string{"!node_modules"}, "main.go", true},
		{"only exclusions exclude matches", []string{"!node_modules"}, "node_modules", false},
		{"no filters include everything", nil, "anything.bin", true},
		{"empty filter entries are ignored", []string{"", "*.go"}, "main.go", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, newFileMatcher(tt.filters).Matched(tt.fileName))
		})
	}
}

// matcherTestNames returns names that hit, nearly hit or miss every pattern in filters.
func matcherTestNames(filters []string) []string {
	names := []string{".", "..", "a", "A.JAVA", "Makefile", "x.Auto.TfVars", "foo.js.erb", ".js", "js",
		"Dockerfile.dev", "my.dockerfile", "a.php5x6", "node_modules", ".vscode", "file.", "a..java", "a.b.c.d.e",
		"requirements-dev.txt", "CMakeLists.txt", "README.md", "image.png", "lib.so.1"}
	for _, f := range filters {
		f = strings.TrimPrefix(f, "!")
		hit := strings.NewReplacer("*", "x", "?", "y").Replace(f)
		names = append(names, hit, strings.ToUpper(hit), "pre"+hit, hit+"post", hit+".bak")
	}
	return names
}

func TestFileMatcherEquivalentToLegacy(t *testing.T) {
	userGlobs := []string{"*.custom", "!*.min.js", "Special?.cfg", "[ab]*.txt"}
	sets := map[string][]string{
		"base include":        commonParams.BaseIncludeFilters,
		"base exclude":        commonParams.BaseExcludeFilters,
		"base include + user": append(append([]string{}, commonParams.BaseIncludeFilters...), userGlobs...),
		"base exclude + user": append(append([]string{}, commonParams.BaseExcludeFilters...), userGlobs...),
		"user only":           userGlobs,
		"empty":               nil,
	}
	names := matcherTestNames(append(append([]string{}, commonParams.BaseIncludeFilters...), userGlobs...))
	for label, filters := range sets {
		m := newFileMatcher(filters)
		mismatches := 0
		for _, name := range names {
			if got, want := m.Matched(name), legacyFilterMatched(filters, name); got != want {
				mismatches++
				t.Errorf("%s: %q: fileMatcher=%v legacy=%v", label, name, got, want)
			}
		}
		assert.Equal(t, 0, mismatches, "%s: %d names checked", label, len(names))
	}
}

func TestCopyFileToZip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big.go")
	// Larger than the copy buffer, so the content spans several reads.
	content := bytes.Repeat([]byte("package main\n"), 10*zipCopyBufSize/13)
	assert.NilError(t, os.WriteFile(src, content, 0o600))

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	buf := make([]byte, zipCopyBufSize)
	assert.NilError(t, copyFileToZip(zw, "src/big.go", src, buf))
	err := copyFileToZip(zw, "src/missing.go", filepath.Join(dir, "missing.go"), buf)
	assert.Assert(t, os.IsNotExist(err), "expected not-exist error, got %v", err)
	assert.NilError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(zr.File), "a file that cannot be opened must not leave an entry")
	rc, err := zr.File[0].Open()
	assert.NilError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	assert.NilError(t, err)
	assert.Equal(t, "src/big.go", zr.File[0].Name)
	assert.Assert(t, bytes.Equal(content, got), "zip content differs from source")
}

var benchmarkFileNames = []string{"Main.java", "index.ts", "README.md", "logo.png", "go.sum", "Makefile",
	"server.zig", "app.min.js", "data.bin", "Dockerfile", "deploy.sh", "style.css", "vendor.lock", "a.b.c.d"}

func BenchmarkFilterMatchedLegacy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		legacyFilterMatched(commonParams.BaseIncludeFilters, benchmarkFileNames[i%len(benchmarkFileNames)])
	}
}

func BenchmarkFileMatcher(b *testing.B) {
	m := newFileMatcher(commonParams.BaseIncludeFilters)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Matched(benchmarkFileNames[i%len(benchmarkFileNames)])
	}
}

func BenchmarkNewFileMatcher(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		newFileMatcher(commonParams.BaseIncludeFilters)
	}
}
