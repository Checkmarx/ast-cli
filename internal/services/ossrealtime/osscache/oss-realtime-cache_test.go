package osscache

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestReadCache_Empty(t *testing.T) {
	cacheFile := GetCacheFilePath()
	// ensure no cache file exists
	_ = os.Remove(cacheFile)
	defer os.Remove(cacheFile)

	if got := ReadCache(); got != nil {
		t.Errorf("ReadCache() = %v; want nil when no file", got)
	}
}

func TestWriteAndReadCache(t *testing.T) {
	cacheFile := GetCacheFilePath()
	defer os.Remove(cacheFile)

	ttl := time.Now().Add(time.Hour).Truncate(time.Second)
	want := Cache{
		TTL: ttl,
		Packages: []PackageEntry{
			{
				PackageManager: "npm",
				PackageName:    "lodash",
				PackageVersion: "4.17.21",
				Status:         "OK",
			},
		},
	}

	if err := WriteCache(want, &want.TTL); err != nil {
		t.Fatalf("WriteCache() error = %v; want no error", err)
	}

	got := ReadCache()
	if got == nil {
		t.Fatal("ReadCache() returned nil; want non-nil")
	}
	assert.Equal(t, want.Packages[0].PackageName, got.Packages[0].PackageName)
	assert.Equal(t, want.Packages[0].PackageVersion, got.Packages[0].PackageVersion)
	assert.Equal(t, want.Packages[0].PackageManager, got.Packages[0].PackageManager)
	assert.Equal(t, want.Packages[0].Status, got.Packages[0].Status)
	asserts.True(t, want.TTL.Equal(got.TTL))
}

func TestAppendToCache(t *testing.T) {
	cacheFile := GetCacheFilePath()
	defer os.Remove(cacheFile)

	first := &wrappers.RealtimeScannerPackageResponse{
		Packages: []wrappers.RealtimeScannerResults{
			{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21", Status: "OK"},
		},
	}
	if err := AppendToCache(first, nil); err != nil {
		t.Fatalf("AppendToCache(first) error = %v; want no error", err)
	}

	second := &wrappers.RealtimeScannerPackageResponse{
		Packages: []wrappers.RealtimeScannerResults{
			{PackageManager: "npm", PackageName: "express", Version: "4.17.1", Status: "Malicious"},
		},
	}
	if err := AppendToCache(second, nil); err != nil {
		t.Fatalf("AppendToCache(second) error = %v; want no error", err)
	}

	cache := ReadCache()
	if cache == nil {
		t.Fatal("ReadCache() returned nil; want non-nil")
	}

	var got []wrappers.RealtimeScannerResults
	for _, e := range cache.Packages {
		got = append(got, wrappers.RealtimeScannerResults{
			PackageManager: e.PackageManager,
			PackageName:    e.PackageName,
			Version:        e.PackageVersion,
			Status:         e.Status,
		})
	}
	want := append([]wrappers.RealtimeScannerResults{}, first.Packages...)
	want = append(want, second.Packages...)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cached packages = %+v; want %+v", got, want)
	}

	if time.Now().After(cache.TTL) {
		t.Errorf("cache TTL expired (%v); want TTL in the future", cache.TTL)
	}
}

func TestAppendToCache_WithVersionMapping(t *testing.T) {
	cacheFile := GetCacheFilePath()
	var latestVersionInCache bool
	var exactVersionInCache bool
	defer os.Remove(cacheFile)

	pkgResponse := &wrappers.RealtimeScannerPackageResponse{
		Packages: []wrappers.RealtimeScannerResults{
			{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21", Status: "OK"},
			{PackageManager: "npm", PackageName: "express", Version: "1.1.1", Status: "Malicious"},
		},
	}
	versionMapping := map[string]string{
		"npm-express-1.1.1": "latest",
	}
	if err := AppendToCache(pkgResponse, versionMapping); err != nil {
		t.Fatalf("AppendToCache(first) error = %v; want no error", err)
	}

	cache := ReadCache()
	if cache == nil {
		t.Fatal("ReadCache() returned nil; want non-nil")
	}

	var got []wrappers.RealtimeScannerResults
	for _, e := range cache.Packages {
		pkg := wrappers.RealtimeScannerResults{
			PackageManager: e.PackageManager,
			PackageName:    e.PackageName,
			Version:        e.PackageVersion,
			Status:         e.Status,
		}
		got = append(got, pkg)
		if strings.EqualFold(pkg.PackageName, "express") && strings.EqualFold(pkg.Version, "latest") {
			latestVersionInCache = true
		} else if strings.EqualFold(pkg.PackageName, "express") && strings.EqualFold(pkg.Version, "1.1.1") {
			exactVersionInCache = true
		}
	}
	asserts.True(t, latestVersionInCache, "Expected latest version of express to be in cache")
	asserts.True(t, exactVersionInCache, "Expected exact versions of express to be in cache")
	asserts.Greater(t, len(got), len(pkgResponse.Packages)) // Ensure that the cache contains the exact version and the latest version of express
}
