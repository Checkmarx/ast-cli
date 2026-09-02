package dependency_resolver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/assert"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	assert.NilError(t, os.WriteFile(path, []byte(content), ownerOnlyFilePerm))
}

func TestComputeManifestHash_MissingFile(t *testing.T) {
	got := computeManifestHash("npm", filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Equal(t, got, "")
}

func TestComputeManifestHash_NonMavenHashesRawBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package-lock.json")

	writeTestFile(t, path, `{"dependencies":{}}`)
	h1 := computeManifestHash("npm", path)

	// For npm/go, any byte change - including whitespace - changes the hash,
	// since lock files are generated and only change when dependencies change.
	writeTestFile(t, path, `{"dependencies": {}}`)
	h2 := computeManifestHash("npm", path)

	assert.Assert(t, h1 != "" && h2 != "")
	assert.Assert(t, h1 != h2)
}

func TestComputeManifestHash_MavenIgnoresCommentsAndWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pom.xml")

	writeTestFile(t, path, "<project><artifactId>app</artifactId></project>")
	base := computeManifestHash("maven", path)

	writeTestFile(t, path, "<project>\n  <!-- a comment -->\n  <artifactId>app</artifactId>\n</project>")
	reformatted := computeManifestHash("maven", path)

	writeTestFile(t, path, "<project><artifactId>changed</artifactId></project>")
	changed := computeManifestHash("maven", path)

	assert.Equal(t, base, reformatted)
	assert.Assert(t, base != changed)
}

func TestTreeCache_WriteThenReadRoundTrip(t *testing.T) {
	cacheFile := treeCacheFilePath()
	defer func() { _ = os.Remove(cacheFile) }()

	key := treeCacheKey("npm", "/tmp/some-project")
	deps := []Dependency{{Name: "lodash", Version: "4.17.21", PackageType: "npm"}}
	assert.NilError(t, writeTreeCache(key, "hash-1", deps))

	got, found := readTreeCache(key, "hash-1")
	assert.Assert(t, found)
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].Name, "lodash")
}

func TestTreeCache_HashMismatchIsMiss(t *testing.T) {
	cacheFile := treeCacheFilePath()
	defer func() { _ = os.Remove(cacheFile) }()

	key := treeCacheKey("npm", "/tmp/some-project")
	assert.NilError(t, writeTreeCache(key, "hash-1", []Dependency{{Name: "lodash"}}))

	_, found := readTreeCache(key, "hash-2")
	assert.Assert(t, !found)
}

func TestTreeCache_EmptyHashIsAlwaysMiss(t *testing.T) {
	cacheFile := treeCacheFilePath()
	defer func() { _ = os.Remove(cacheFile) }()

	key := treeCacheKey("npm", "/tmp/some-project")
	assert.NilError(t, writeTreeCache(key, "hash-1", []Dependency{{Name: "lodash"}}))

	_, found := readTreeCache(key, "")
	assert.Assert(t, !found)
}

func TestTreeCache_MissingFileIsMiss(t *testing.T) {
	cacheFile := treeCacheFilePath()
	_ = os.Remove(cacheFile)
	defer func() { _ = os.Remove(cacheFile) }()

	_, found := readTreeCache(treeCacheKey("npm", "/tmp/some-project"), "hash-1")
	assert.Assert(t, !found)
}

func TestTreeCache_ExpiredEntryIsMiss(t *testing.T) {
	cacheFile := treeCacheFilePath()
	defer func() { _ = os.Remove(cacheFile) }()

	key := treeCacheKey("npm", "/tmp/some-project")
	cache := treeCacheFile{Entries: map[string]treeCacheEntry{
		key: {
			ManifestHash: "hash-1",
			Dependencies: []Dependency{{Name: "lodash"}},
			CachedAt:     time.Now().Add(-treeCacheTTL - time.Minute),
		},
	}}
	data, err := json.Marshal(cache)
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(cacheFile, data, ownerOnlyFilePerm))

	_, found := readTreeCache(key, "hash-1")
	assert.Assert(t, !found)
}

func TestTreeCache_DistinctPackageManagersDoNotCollide(t *testing.T) {
	cacheFile := treeCacheFilePath()
	defer func() { _ = os.Remove(cacheFile) }()

	npmKey := treeCacheKey("npm", "/tmp/polyglot-project")
	mavenKey := treeCacheKey("maven", "/tmp/polyglot-project")

	assert.NilError(t, writeTreeCache(npmKey, "npm-hash", []Dependency{{Name: "lodash"}}))
	assert.NilError(t, writeTreeCache(mavenKey, "maven-hash", []Dependency{{Name: "org.springframework:spring-core"}}))

	npmDeps, found := readTreeCache(npmKey, "npm-hash")
	assert.Assert(t, found)
	assert.Equal(t, npmDeps[0].Name, "lodash")

	mavenDeps, found := readTreeCache(mavenKey, "maven-hash")
	assert.Assert(t, found)
	assert.Equal(t, mavenDeps[0].Name, "org.springframework:spring-core")
}
