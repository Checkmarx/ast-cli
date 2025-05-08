package osscache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	cacheFileName = "oss-realtime-cache.json"
	ttl           = 1 * time.Hour
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
	tempFolder := os.TempDir()
	cacheFilePath := fmt.Sprint(tempFolder, "/", cacheFileName)
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

func AppendToCache(packages *wrappers.OssPackageResponse) error {
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
