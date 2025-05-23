package osscache

import (
	"os"
	"reflect"
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

	// prepare a cache object
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

	// write it
	if err := WriteCache(want, &want.TTL); err != nil {
		t.Fatalf("WriteCache() error = %v; want no error", err)
	}

	// read it back
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

	// first batch
	first := &wrappers.RealtimeScannerPackageResponse{
		Packages: []wrappers.RealtimeScannerResults{
			{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21", Status: "OK"},
		},
	}
	if err := AppendToCache(first); err != nil {
		t.Fatalf("AppendToCache(first) error = %v; want no error", err)
	}

	// second batch
	second := &wrappers.RealtimeScannerPackageResponse{
		Packages: []wrappers.RealtimeScannerResults{
			{PackageManager: "npm", PackageName: "express", Version: "4.17.1", Status: "Malicious"},
		},
	}
	if err := AppendToCache(second); err != nil {
		t.Fatalf("AppendToCache(second) error = %v; want no error", err)
	}

	// now read & verify we have both entries
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

func Test_buildCacheMap(t *testing.T) {
	type args struct {
		cache Cache
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildCacheMap(tt.args.cache); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCacheMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cacheKey(t *testing.T) {
	type args struct {
		manager string
		name    string
		version string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateCacheKey(tt.args.manager, tt.args.name, tt.args.version); got != tt.want {
				t.Errorf("cacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
