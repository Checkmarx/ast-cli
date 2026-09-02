package dependency_resolver

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

const (
	treeCacheFileName   = "oss-realtime-dependency-tree-cache.json"
	treeCacheLockSuffix = ".lock"
	// treeCacheTTL is shorter than the vulnerability cache TTL (osscache, 4h)
	// because a resolved tree can go stale without any manifest edit at all -
	// e.g. a Maven SNAPSHOT or version range resolving to a newly published
	// artifact upstream.
	treeCacheTTL = 30 * time.Minute
	// ownerOnlyFilePerm keeps the cache file readable/writable by its owner only,
	// since it lives in the shared system temp directory.
	ownerOnlyFilePerm = 0o600
	// maxTreeCacheFileBytes bounds how much of the cache file is read before
	// decoding, since it lives in a shared, multi-writer temp directory.
	maxTreeCacheFileBytes = 25 * 1024 * 1024
)

// xmlCommentPattern strips XML comments so pom.xml can be normalized before
// hashing; see computeManifestHash.
var xmlCommentPattern = regexp.MustCompile(`<!--[\s\S]*?-->`)

// treeCacheEntry is one cached dependency tree, keyed by package manager + project path.
type treeCacheEntry struct {
	ManifestHash string       `json:"manifestHash"`
	Dependencies []Dependency `json:"dependencies"`
	CachedAt     time.Time    `json:"cachedAt"`
}

// treeCacheFile is our own fixed, non-polymorphic schema for the on-disk cache: a
// malformed file can only fail to decode, never deserialize into arbitrary types.
type treeCacheFile struct {
	Entries map[string]treeCacheEntry `json:"entries"`
}

func treeCacheKey(pkgMgr, projectPath string) string {
	return pkgMgr + ":" + projectPath
}

// treeCacheFilePath returns the sanitized, absolute path to the shared dependency
// tree cache file in the system temp directory.
func treeCacheFilePath() string {
	return filepath.Clean(filepath.Join(os.TempDir(), treeCacheFileName))
}

// computeManifestHash returns the content hash used to decide whether a cached tree
// is still fresh. Lock files (package-lock.json, go.sum) are generated and only
// change when dependencies actually change, so their raw bytes are hashed directly.
// pom.xml is hand-edited, so comments and insignificant whitespace are stripped
// first, so purely cosmetic edits (formatting, comments) don't invalidate the cached
// tree. Returns "" on read failure, which never matches a cached entry and forces a
// fresh resolve.
func computeManifestHash(pkgMgr, filePath string) string {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return ""
	}
	if pkgMgr == "maven" {
		data = xmlCommentPattern.ReplaceAll(data, nil)
		// Collapse away all whitespace rather than to a single space: whitespace
		// between XML elements is never significant, and joining with a space
		// would make the hash depend on whether the file happened to have any
		// separating whitespace at all (e.g. minified vs. indented).
		data = []byte(strings.Join(strings.Fields(string(data)), ""))
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

// decodeTreeCacheFile reads the cache file at path into our own fixed schema
// (treeCacheFile has no interface{}/polymorphic fields). The read is bounded since
// the file lives in a shared, multi-writer temp directory.
func decodeTreeCacheFile(path string) (treeCacheFile, error) {
	var cache treeCacheFile
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return cache, err
	}
	defer func() { _ = file.Close() }()

	limited := io.LimitReader(file, maxTreeCacheFileBytes)
	err = json.NewDecoder(limited).Decode(&cache)
	return cache, err
}

// readTreeCache returns the cached dependency tree for key, if present, unexpired,
// and resolved from the same manifest content (manifestHash matches).
func readTreeCache(key, manifestHash string) ([]Dependency, bool) {
	if manifestHash == "" {
		return nil, false
	}
	cache, err := decodeTreeCacheFile(treeCacheFilePath())
	if err != nil {
		return nil, false
	}

	entry, found := cache.Entries[key]
	if !found || entry.ManifestHash != manifestHash || time.Since(entry.CachedAt) > treeCacheTTL {
		return nil, false
	}
	return entry.Dependencies, true
}

// writeTreeCache persists the resolved dependency tree under key, guarded by a file
// lock so concurrent scans can't corrupt the shared cache file.
func writeTreeCache(key, manifestHash string, deps []Dependency) error {
	cacheFilePath := treeCacheFilePath()

	fileLock := flock.New(cacheFilePath + treeCacheLockSuffix)
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("locking dependency tree cache file: %w", err)
	}
	if !locked {
		return fmt.Errorf("dependency tree cache file lock is held by another process")
	}
	defer func() { _ = fileLock.Unlock() }()

	cache, _ := decodeTreeCacheFile(cacheFilePath) // best-effort; start fresh on any error
	if cache.Entries == nil {
		cache.Entries = map[string]treeCacheEntry{}
	}

	cache.Entries[key] = treeCacheEntry{
		ManifestHash: manifestHash,
		Dependencies: deps,
		CachedAt:     time.Now(),
	}

	file, err := os.OpenFile(filepath.Clean(cacheFilePath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, ownerOnlyFilePerm)
	if err != nil {
		return fmt.Errorf("failed to create dependency tree cache file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return json.NewEncoder(file).Encode(cache)
}
