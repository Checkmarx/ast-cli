package ossrealtime

import (
	"os"
	"testing"
	"time"

	"github.com/Checkmarx/manifest-parser/pkg/models"
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func setupPackages() []models.Package {
	return []models.Package{
		{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21"},
		{PackageManager: "npm", PackageName: "express", Version: "4.17.1"},
		{PackageManager: "npm", PackageName: "axios", Version: "0.21.1"},
	}
}

func setupSinglePackage() []models.Package {
	return []models.Package{
		{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21"},
	}
}

func setupCache(packages []osscache.PackageEntry, ttl time.Time) osscache.Cache {
	return osscache.Cache{TTL: ttl, Packages: packages}
}

func cleanCacheFile(t *testing.T) {
	cacheFile := osscache.GetCacheFilePath()
	_ = os.Remove(cacheFile)
	t.Cleanup(func() { _ = os.Remove(cacheFile) })
}

func TestRunOssRealtimeScan_ValidLicenseAndManifest_ScanSuccess(t *testing.T) {
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../commands/data/manifests/package.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath)

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Greater(t, len(response.Packages), 0)
}

func TestRunOssRealtimeScan_InvalidLicenseAndValidManifest_ScanFail(t *testing.T) {
	t.Skip() // Skip this test for now, no license check implemented
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{AIEnabled: mock.AIProtectionDisabled},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../commands/data/manifests/package.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath)

	assert.NotNil(t, err)
	assert.Nil(t, response)
}

func TestRunOssRealtimeScan_ValidLicenseAndInvalidManifest_ScanFail(t *testing.T) {
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "not-supported-manifest.ruby"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath)

	assert.NotNil(t, err)
	assert.Nil(t, response)
}

func TestPrepareScan_CacheExistsAndContainsPartialResults_RealtimeScannerRequestIsCalledPartially(t *testing.T) {
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	allPackages := setupPackages()
	storedPackages := []osscache.PackageEntry{
		{PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := ossRealtimeService.prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 1)
	assert.Equal(t, "lodash", resp.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", resp.Packages[0].Version)
	assert.Equal(t, "OK", resp.Packages[0].Status)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 2)
}

func TestPrepareScan_CacheExpiredAndContainsPartialResults_RealtimeScannerRequestIsCalledFully(t *testing.T) {
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	allPackages := setupPackages()
	storedPackages := []osscache.PackageEntry{
		{PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now())

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := ossRealtimeService.prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 0)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 3)
}

func TestPrepareScan_NoCache_RealtimeScannerRequestIsCalledFully(t *testing.T) {
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	allPackages := setupPackages()

	resp, toScan := ossRealtimeService.prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 0)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 3)
}

func TestPrepareScan_AllDataInCache_RealtimeScannerRequestIsEmpty(t *testing.T) {
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	singlePackage := setupSinglePackage()
	storedPackages := []osscache.PackageEntry{
		{PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := ossRealtimeService.prepareScan(singlePackage)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 1)
	assert.Equal(t, "lodash", resp.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", resp.Packages[0].Version)
	assert.Equal(t, "OK", resp.Packages[0].Status)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 0)
}

func TestScanAndCache_NoCacheAndScanSuccess_CacheUpdated(t *testing.T) {
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{
			CustomScan: func(packages *wrappers.OssPackageRequest) (*wrappers.OssPackageResponse, error) {
				var response wrappers.OssPackageResponse
				for _, pkg := range packages.Packages {
					response.Packages = append(response.Packages, wrappers.OssResults{
						PackageManager: pkg.PackageManager,
						PackageName:    pkg.PackageName,
						Version:        pkg.Version,
						Status:         "OK",
					})
				}
				return &response, nil
			},
		},
	)

	pkgs := setupSinglePackage()

	resp, toScan := ossRealtimeService.prepareScan(pkgs)

	err := ossRealtimeService.scanAndCache(toScan, resp)
	assert.Nil(t, err)

	cache := osscache.ReadCache()
	assert.NotNil(t, cache)
	assert.Len(t, cache.Packages, 1)
	assert.Equal(t, "lodash", cache.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", cache.Packages[0].PackageVersion)
}

func TestScanAndCache_CacheExistsAndScanSuccess_CacheUpdated(t *testing.T) {
	cleanCacheFile(t)

	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	pkgs := []models.Package{
		{PackageManager: "npm", PackageName: "lodash", Version: "4.17.21"},
		{PackageManager: "npm", PackageName: "express", Version: "4.17.1"},
	}

	storedPackages := []osscache.PackageEntry{
		{PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	err := osscache.WriteCache(storedCache, &storedCache.TTL)
	assert.Nil(t, err)

	resp, toScan := ossRealtimeService.prepareScan(pkgs)

	ossRealtimeService.RealtimeScannerWrapper = &mock.RealtimeScannerMockWrapper{
		CustomScan: func(packages *wrappers.OssPackageRequest) (*wrappers.OssPackageResponse, error) {
			var response wrappers.OssPackageResponse
			for _, pkg := range packages.Packages {
				status := "OK"
				if pkg.PackageName == "express" {
					status = "Malicious"
				}
				response.Packages = append(response.Packages, wrappers.OssResults{
					PackageManager: pkg.PackageManager,
					PackageName:    pkg.PackageName,
					Version:        pkg.Version,
					Status:         status,
				})
			}
			return &response, nil
		},
	}

	err = ossRealtimeService.scanAndCache(toScan, resp)
	assert.Nil(t, err)

	cache := osscache.ReadCache()
	assert.NotNil(t, cache)
	assert.Len(t, cache.Packages, 2)

	assert.Equal(t, "npm", cache.Packages[0].PackageManager)
	assert.Equal(t, "lodash", cache.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", cache.Packages[0].PackageVersion)
	assert.Equal(t, "OK", cache.Packages[0].Status)

	assert.Equal(t, "npm", cache.Packages[1].PackageManager)
	assert.Equal(t, "express", cache.Packages[1].PackageName)
	assert.Equal(t, "4.17.1", cache.Packages[1].PackageVersion)
	assert.Equal(t, "Malicious", cache.Packages[1].Status)
}
