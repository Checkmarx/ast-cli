package ossrealtime

import (
	"os"
	"testing"
	"time"

	"github.com/Checkmarx/manifest-parser/pkg/parser/models"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime/osscache"
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
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../../commands/data/manifests/package.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, "")

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

	const filePath = "../../../commands/data/manifests/package.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, "")

	assert.NotNil(t, err)
	assert.Nil(t, response)
}

func TestRunOssRealtimeScan_ValidLicenseAndInvalidManifest_ScanFail(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "not-supported-manifest.ruby"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, "")

	assert.NotNil(t, err)
	assert.Nil(t, response)
}

func TestRunOssRealtimeScan_WithIgnoredPackage_IgnoresPackage(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../../commands/data/manifests/package.json"
	const ignoredPath = "../../../commands/data/checkmarxIgnoredTempList.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, ignoredPath)

	assert.Nil(t, err)
	assert.NotNil(t, response)

	for _, pkg := range response.Packages {
		assert.NotEqual(t, "coa", pkg.PackageName, "Package 'coa' should be ignored but was found in the results")
	}
}

func TestRunOssRealtimeScan_WithEmptyIgnoreFile_AllPackagesIncluded(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../../commands/data/manifests/package.json"
	const ignoredPath = "../../../commands/data/emptyIgnoreList.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, ignoredPath)

	assert.Nil(t, err)
	assert.NotNil(t, response)

	var found bool
	for _, pkg := range response.Packages {
		if pkg.PackageName == "coa" && pkg.PackageVersion == "3.1.3" {
			found = true
			break
		}
	}
	assert.True(t, found, "'coa' should not be ignored if ignore list is empty")
}

func TestRunOssRealtimeScan_IgnoredPackageWithDifferentVersion_IsNotIgnored(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	const filePath = "../../../commands/data/manifests/test.csproj"
	const ignoredPath = "../../../commands/data/checkmarxIgnoredTempListCsproj.json"

	response, err := ossRealtimeService.RunOssRealtimeScan(filePath, ignoredPath)

	assert.Nil(t, err)
	assert.NotNil(t, response)

	var found bool
	for _, pkg := range response.Packages {
		if pkg.PackageManager == "nuget" &&
			pkg.PackageName == "Microsoft.Extensions.Caching.Memory" &&
			pkg.PackageVersion == "6.0.1" {
			found = true
			break
		}
	}

	assert.True(t, found, "Package 'Microsoft.Extensions.Caching.Memory@6.0.1' should NOT be ignored since version in ignore file is 6.0.3")
}

func TestPrepareScan_CacheExistsAndContainsPartialResults_RealtimeScannerRequestIsCalledPartially(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	cleanCacheFile(t)
	allPackages := setupPackages()
	storedPackages := []osscache.PackageEntry{
		{PackageID: "npm-lodash-4.17.21", PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 1)
	assert.Equal(t, "lodash", resp.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", resp.Packages[0].PackageVersion)
	assert.Equal(t, "OK", resp.Packages[0].Status)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 2)
}

func TestPrepareScan_CacheExpiredAndContainsPartialResults_RealtimeScannerRequestIsCalledFully(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	cleanCacheFile(t)
	allPackages := setupPackages()
	storedPackages := []osscache.PackageEntry{
		{PackageID: "npm-lodash-4.17.21", PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now())

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 0)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 3)
}

func TestPrepareScan_NoCache_RealtimeScannerRequestIsCalledFully(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	cleanCacheFile(t)
	allPackages := setupPackages()

	resp, toScan := prepareScan(allPackages)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 0)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 3)
}

func TestPrepareScan_AllDataInCache_RealtimeScannerRequestIsEmpty(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	cleanCacheFile(t)
	singlePackage := setupSinglePackage()
	storedPackages := []osscache.PackageEntry{
		{PackageID: "npm-lodash-4.17.21", PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	_ = osscache.WriteCache(storedCache, &storedCache.TTL)

	resp, toScan := prepareScan(singlePackage)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Packages, 1)
	assert.Equal(t, "lodash", resp.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", resp.Packages[0].PackageVersion)
	assert.Equal(t, "OK", resp.Packages[0].Status)

	assert.NotNil(t, toScan)
	assert.Len(t, toScan.Packages, 0)
}

func TestScanAndCache_NoCacheAndScanSuccess_CacheUpdated(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	cleanCacheFile(t)
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{
			CustomScan: func(packages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
				var response wrappers.RealtimeScannerPackageResponse
				for _, pkg := range packages.Packages {
					response.Packages = append(response.Packages, wrappers.RealtimeScannerResults{
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

	_, toScan := prepareScan(pkgs)

	_, err := ossRealtimeService.scanAndCache(toScan)
	assert.Nil(t, err)

	cache := osscache.ReadCache()
	assert.NotNil(t, cache)
	assert.Len(t, cache.Packages, 1)
	assert.Equal(t, "lodash", cache.Packages[0].PackageName)
	assert.Equal(t, "4.17.21", cache.Packages[0].PackageVersion)
}

func TestScanAndCache_CacheExistsAndScanSuccess_CacheUpdated(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
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
		{PackageID: "npm-lodash-4.17.21", PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"},
	}
	storedCache := setupCache(storedPackages, time.Now().Add(time.Hour))

	err := osscache.WriteCache(storedCache, &storedCache.TTL)
	assert.Nil(t, err)

	_, toScan := prepareScan(pkgs)

	ossRealtimeService.RealtimeScannerWrapper = &mock.RealtimeScannerMockWrapper{
		CustomScan: func(packages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
			var response wrappers.RealtimeScannerPackageResponse
			for _, pkg := range packages.Packages {
				status := "OK"
				if pkg.PackageName == "express" {
					status = "Malicious"
				}
				response.Packages = append(response.Packages, wrappers.RealtimeScannerResults{
					PackageManager: pkg.PackageManager,
					PackageName:    pkg.PackageName,
					Version:        pkg.Version,
					Status:         status,
				})
			}
			return &response, nil
		},
	}

	_, err = ossRealtimeService.scanAndCache(toScan)
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

func TestOssRealtimeScan_CsprojFile_ReturnsLocations(t *testing.T) {
	// Arrange
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	ossRealtimeService := NewOssRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	response, err := ossRealtimeService.RunOssRealtimeScan("../../../commands/data/manifests/test.csproj", "")

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 6, len(response.Packages), "Should find exactly 5 packages in test.csproj")

	// Find the Microsoft.TeamFoundationServer.Client package that should have 3 locations
	var tfsPackage *OssPackage
	for _, pkg := range response.Packages {
		if pkg.PackageName == "Microsoft.TeamFoundationServer.Client" && pkg.PackageVersion == "19.225.1" {
			tfsPackage = &pkg
			break
		}
	}

	// Assert TFS package was found and has expected locations
	assert.NotNil(t, tfsPackage, "Should find Microsoft.TeamFoundationServer.Client package")
	assert.Equal(t, 3, len(tfsPackage.Locations), "TFS package should have exactly 3 locations")
	assert.Equal(t, "../../../commands/data/manifests/test.csproj", tfsPackage.FilePath)

	// Verify specific location details
	expectedLocations := []struct {
		line       int
		startIndex int
		endIndex   int
	}{
		{32, 4, 70}, // Location 0: Line=32, StartIndex=4, EndIndex=70
		{33, 6, 33}, // Location 1: Line=33, StartIndex=6, EndIndex=33
		{34, 4, 23}, // Location 2: Line=34, StartIndex=4, EndIndex=23
	}

	for i, expected := range expectedLocations {
		assert.Equal(t, expected.line, tfsPackage.Locations[i].Line,
			"Location %d line should be %d", i, expected.line)
		assert.Equal(t, expected.startIndex, tfsPackage.Locations[i].StartIndex,
			"Location %d startIndex should be %d", i, expected.startIndex)
		assert.Equal(t, expected.endIndex, tfsPackage.Locations[i].EndIndex,
			"Location %d endIndex should be %d", i, expected.endIndex)
	}
}
