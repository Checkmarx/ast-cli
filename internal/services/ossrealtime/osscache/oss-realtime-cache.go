package osscache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	cacheFileName  = "oss-realtime-cache.json"
	ttlHoursNumber = 4
	ttl            = ttlHoursNumber * time.Hour
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

func AppendToCache(packages *wrappers.RealtimeScannerPackageResponse) error {
	cache := ReadCache()
	if cache == nil {
		cache = &Cache{
			TTL:      time.Now().Add(ttl),
			Packages: make([]PackageEntry, 0),
		}
	}

	for _, pkg := range packages.Packages {
		if pkg.Status != "Unknown" {
			cache.Packages = append(cache.Packages, PackageEntry{
				PackageManager: pkg.PackageManager,
				PackageName:    pkg.PackageName,
				PackageVersion: pkg.Version,
				Status:         pkg.Status,
			})
		}
	}
	return WriteCache(*cache, &cache.TTL)
}

func GetCacheFilePath() string {
	tempFolder := os.TempDir()
	return fmt.Sprint(tempFolder, "/", cacheFileName)
}

// BuildCacheMap creates a lookup map from cache entries.
func BuildCacheMap(cache Cache) map[string]string {
	m := make(map[string]string, len(cache.Packages))
	for _, pkg := range cache.Packages {
		m[GenerateCacheKey(pkg.PackageManager, pkg.PackageName, pkg.PackageVersion)] = pkg.Status
	}
	return m
}

// GenerateCacheKey constructs a unique key for a package.
func GenerateCacheKey(manager, name, version string) string {
	return fmt.Sprintf("%s-%s-%s", manager, name, version)
}
