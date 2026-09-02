package osscache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/gofrs/flock"
)

const (
	cacheFileName  = "oss-realtime-cache.json"
	lockFileSuffix = ".lock"
	ttlHoursNumber = 4
	ttl            = ttlHoursNumber * time.Hour
	// maxCacheEntries caps the cache file size so it can't grow unbounded across
	// repeated scans (e.g. an IDE re-scanning on every save).
	maxCacheEntries = 5000
)

func ReadCache() *Cache {
	tempFolder := os.TempDir()
	cacheFilePath := fmt.Sprint(tempFolder, "/", cacheFileName)
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return nil
	}
	file, err := os.Open(cacheFilePath)
	if err != nil {
		return nil
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	var cache Cache
	if err = json.NewDecoder(file).Decode(&cache); err != nil {
		return nil
	}
	if time.Now().After(cache.TTL) {
		return nil
	}
	return &cache
}

func WriteCache(cache Cache, cacheTTL *time.Time) error {
	cacheFilePath := GetCacheFilePath()

	// Guard the write with a file lock so concurrent scans (e.g. an IDE scanning
	// multiple manifests at once) can't interleave writes and corrupt the cache file.
	fileLock := flock.New(cacheFilePath + lockFileSuffix)
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("locking oss-realtime cache file: %w", err)
	}
	if !locked {
		return fmt.Errorf("oss-realtime cache file lock is held by another process")
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	file, err := os.Create(cacheFilePath)
	if err != nil {
		return fmt.Errorf("failed to create osscache file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	if cacheTTL == nil {
		cache.TTL = time.Now().Add(ttl)
	} else {
		cache.TTL = *cacheTTL
	}
	if err = json.NewEncoder(file).Encode(cache); err != nil {
		return fmt.Errorf("failed to encode osscache file: %w", err)
	}
	return nil
}

func AppendToCache(packages *wrappers.RealtimeScannerPackageResponse, versionMapping map[string]string) error {
	vulnerabilityMapper := NewOssCacheVulnerabilityMapper()
	cache := ReadCache()
	if cache == nil {
		cache = &Cache{
			TTL:      time.Now().Add(ttl),
			Packages: make([]PackageEntry, 0),
		}
	}

	// Index existing entries by PackageID so re-scanned packages overwrite their
	// existing entry instead of appending a duplicate (the cache file otherwise
	// grows without bound across repeated scans of the same packages).
	index := make(map[string]int, len(cache.Packages))
	for i, entry := range cache.Packages {
		index[entry.PackageID] = i
	}

	for _, pkg := range packages.Packages {
		key := GenerateCacheKey(pkg.PackageManager, pkg.PackageName, pkg.Version)
		vulnerabilities := vulnerabilityMapper.FromRealtimeScannerVulnerability(pkg.Vulnerabilities)

		if requestedVersion, exists := versionMapping[key]; exists {
			if !strings.EqualFold(requestedVersion, pkg.Version) && strings.EqualFold("latest", requestedVersion) {
				upsertCacheEntry(cache, index, createPackageEntry(&pkg, requestedVersion, vulnerabilities))
			}
		}
		upsertCacheEntry(cache, index, createPackageEntry(&pkg, pkg.Version, vulnerabilities))
	}

	// Evict the oldest entries once the cache grows past the cap.
	if len(cache.Packages) > maxCacheEntries {
		cache.Packages = cache.Packages[len(cache.Packages)-maxCacheEntries:]
	}

	return WriteCache(*cache, &cache.TTL)
}

// upsertCacheEntry replaces an existing cache entry with the same PackageID, or
// appends it as a new entry if it isn't present yet.
func upsertCacheEntry(cache *Cache, index map[string]int, entry PackageEntry) {
	if i, exists := index[entry.PackageID]; exists {
		cache.Packages[i] = entry
		return
	}
	index[entry.PackageID] = len(cache.Packages)
	cache.Packages = append(cache.Packages, entry)
}

func createPackageEntry(pkg *wrappers.RealtimeScannerResults, version string, vulnerabilities []Vulnerability) PackageEntry {
	return PackageEntry{
		PackageID:       GenerateCacheKey(pkg.PackageManager, pkg.PackageName, version),
		PackageManager:  pkg.PackageManager,
		PackageName:     pkg.PackageName,
		PackageVersion:  version,
		Status:          pkg.Status,
		Vulnerabilities: vulnerabilities,
	}
}

func GetCacheFilePath() string {
	tempFolder := os.TempDir()
	return fmt.Sprint(tempFolder, "/", cacheFileName)
}

// BuildCacheMap creates a lookup map from cache entries.
func BuildCacheMap(cache Cache) map[string]PackageEntry {
	packagesMap := make(map[string]PackageEntry, len(cache.Packages))
	for _, pkg := range cache.Packages {
		packagesMap[pkg.PackageID] = pkg
	}
	return packagesMap
}

// GenerateCacheKey constructs a unique key for a package.
func GenerateCacheKey(manager, name, version string) string {
	return fmt.Sprintf("%s-%s-%s", manager, name, version)
}
