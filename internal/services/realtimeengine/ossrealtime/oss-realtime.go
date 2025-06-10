package ossrealtime

import (
	"fmt"
	"strings"

	"github.com/Checkmarx/manifest-parser/pkg/parser"
	"github.com/Checkmarx/manifest-parser/pkg/parser/models"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

// createLocations creates a Locations array from line and index information
func createLocations(lineStart, lineEnd, startIndex, endIndex int) []realtimeengine.Location {
	var locations []realtimeengine.Location
	for i := 0; i <= lineEnd-lineStart; i++ {
		locations = append(locations, realtimeengine.Location{
			Line:       lineStart + i,
			StartIndex: startIndex,
			EndIndex:   endIndex,
		})
	}
	return locations
}

// OssRealtimeService is the service responsible for performing real-time OSS scanning.
type OssRealtimeService struct {
	JwtWrapper             wrappers.JWTWrapper
	FeatureFlagWrapper     wrappers.FeatureFlagsWrapper
	RealtimeScannerWrapper wrappers.RealtimeScannerWrapper
}

// NewOssRealtimeService creates a new OssRealtimeService.
func NewOssRealtimeService(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
	realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
) *OssRealtimeService {
	return &OssRealtimeService{
		JwtWrapper:             jwtWrapper,
		FeatureFlagWrapper:     featureFlagWrapper,
		RealtimeScannerWrapper: realtimeScannerWrapper,
	}
}

// RunOssRealtimeScan performs an OSS real-time scan on the given manifest file.
func (o *OssRealtimeService) RunOssRealtimeScan(filePath string) (*OssPackageResults, error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError("file path is required").Error()
	}

	if enabled, err := o.isFeatureFlagEnabled(); err != nil || !enabled {
		logger.PrintfIfVerbose("Failed to print OSS Realtime scan results: %v", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	if err := o.ensureLicense(); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}

	pkgs, err := parseManifest(filePath)
	if err != nil {
		logger.PrintfIfVerbose("Failed to parse manifest file %s: %v", filePath, err)
		return nil, errorconstants.NewRealtimeEngineError("failed to parse manifest file").Error()
	}

	response, toScan := prepareScan(pkgs)

	if len(toScan.Packages) > 0 {
		result, err := o.scanAndCache(toScan)
		if err != nil {
			logger.PrintfIfVerbose("Failed to scan packages via realtime service: %v", err)
			return nil, errorconstants.NewRealtimeEngineError("Realtime scanner engine failed").Error()
		}
		packageMap := createPackageMap(pkgs)
		enrichResponseWithRealtimeScannerResults(response, result, packageMap)
	}
	return response, nil
}

func enrichResponseWithRealtimeScannerResults(
	response *OssPackageResults,
	result *wrappers.RealtimeScannerPackageResponse,
	packageMap map[string]OssPackage,
) {
	vulnerabilityMapper := NewOssVulnerabilityMapper()
	for _, pkg := range result.Packages {
		entry := getPackageEntryFromPackageMap(packageMap, &pkg)
		response.Packages = append(response.Packages, OssPackage{
			PackageManager:  pkg.PackageManager,
			PackageName:     pkg.PackageName,
			PackageVersion:  pkg.Version,
			FilePath:        entry.FilePath,
			Locations:       entry.Locations,
			Status:          pkg.Status,
			Vulnerabilities: vulnerabilityMapper.FromRealtimeScanner(pkg.Vulnerabilities),
		})
	}
}

func getPackageEntryFromPackageMap(
	packageMap map[string]OssPackage,
	pkg *wrappers.RealtimeScannerResults,
) *OssPackage {
	var entry OssPackage
	if value, found := packageMap[generatePackageMapEntry(pkg.PackageManager, pkg.PackageName, pkg.Version)]; found {
		entry = value
	} else {
		entry = packageMap[generatePackageMapEntry(pkg.PackageManager, pkg.PackageName, "latest")]
	}
	return &entry
}

// isFeatureFlagEnabled checks if the OSS Realtime feature flag is enabled.
func (o *OssRealtimeService) isFeatureFlagEnabled() (bool, error) {
	enabled, err := o.FeatureFlagWrapper.GetSpecificFlag(wrappers.OssRealtimeEnabled)
	if err != nil {
		return false, errors.Wrap(err, "failed to get feature flag")
	}
	return enabled.Status, nil
}

// ensureLicense validates that a valid JWT wrapper is available.
func (o *OssRealtimeService) ensureLicense() error {
	if o.JwtWrapper == nil {
		return errors.New("JWT wrapper is not initialized, cannot ensure license")
	}
	return nil
}

// parseManifest parses the manifest file and returns a list of packages.
func parseManifest(filePath string) ([]models.Package, error) {
	manifestParser := parser.ParsersFactory(filePath)
	if manifestParser == nil {
		return nil, errors.Errorf("no parser available for file: %s", filePath)
	}
	pkgs, err := manifestParser.Parse(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "parsing manifest file error")
	}
	return pkgs, nil
}

// prepareScan processes the list of packages and separates cached and uncached packages.
func prepareScan(pkgs []models.Package) (*OssPackageResults, *wrappers.RealtimeScannerPackageRequest) {
	var resp OssPackageResults
	resp.Packages = make([]OssPackage, 0, len(pkgs))
	var req wrappers.RealtimeScannerPackageRequest
	vulnerabilityMapper := NewOssVulnerabilityMapper()

	cache := osscache.ReadCache()
	if cache == nil {
		for _, pkg := range pkgs {
			req.Packages = append(req.Packages, pkgToRequest(&pkg))
		}
		return &resp, &req
	}

	cacheMap := osscache.BuildCacheMap(*cache)
	for _, pkg := range pkgs {
		key := osscache.GenerateCacheKey(pkg.PackageManager, pkg.PackageName, pkg.Version)
		if cachedPkg, found := cacheMap[key]; found {
			resp.Packages = append(resp.Packages, OssPackage{
				PackageManager:  pkg.PackageManager,
				PackageName:     pkg.PackageName,
				PackageVersion:  pkg.Version,
				FilePath:        pkg.FilePath,
				Locations:       createLocations(pkg.LineStart, pkg.LineEnd, pkg.StartIndex, pkg.EndIndex),
				Status:          cachedPkg.Status,
				Vulnerabilities: vulnerabilityMapper.FromCache(cachedPkg.Vulnerabilities),
			})
		} else {
			req.Packages = append(req.Packages, pkgToRequest(&pkg))
		}
	}
	return &resp, &req
}

// createPackageMap generates a map of packages for quicker access during scanning.
func createPackageMap(pkgs []models.Package) map[string]OssPackage {
	packageMap := make(map[string]OssPackage)
	for _, pkg := range pkgs {
		packageMap[generatePackageMapEntry(pkg.PackageManager, pkg.PackageName, pkg.Version)] = OssPackage{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			PackageVersion: pkg.Version,
			FilePath:       pkg.FilePath,
			Locations:      createLocations(pkg.LineStart, pkg.LineEnd, pkg.StartIndex, pkg.EndIndex),
		}
	}
	return packageMap
}

// generatePackageMapEntry generates a unique key for the package map.
func generatePackageMapEntry(pkgManager, pkgName, pkgVersion string) string {
	return fmt.Sprintf("%s_%s_%s", pkgManager, pkgName, pkgVersion)
}

// scanAndCache performs a scan on the provided packages and caches the results.
func (o *OssRealtimeService) scanAndCache(requestPackages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
	result, err := o.RealtimeScannerWrapper.Scan(requestPackages)
	if err != nil {
		logger.PrintfIfVerbose("Failed to scan packages via realtime service: %v", err)
		return nil, errors.Wrap(err, "scanning packages via realtime service")
	}
	if len(result.Packages) == 0 {
		logger.PrintfIfVerbose("Received empty response from oss-realtime scan for packages: %v", requestPackages.Packages)
		return nil, errors.New("empty response from oss-realtime scan")
	}

	versionMapping := createVersionMapping(requestPackages, result)

	if err := osscache.AppendToCache(result, versionMapping); err != nil {
		logger.PrintfIfVerbose("oss-realtime: failed to update cache: %v", err)
	}

	return result, nil
}

func createVersionMapping(requestPackages *wrappers.RealtimeScannerPackageRequest, result *wrappers.RealtimeScannerPackageResponse) map[string]string {
	requestedPackagesVersion := make(map[string]string)
	for _, pkg := range requestPackages.Packages {
		key := fmt.Sprintf("%s|%s", strings.ToLower(pkg.PackageManager), strings.ToLower(pkg.PackageName))
		requestedPackagesVersion[key] = pkg.Version
	}

	versionMapping := make(map[string]string)
	for _, resPkg := range result.Packages {
		key := fmt.Sprintf("%s|%s", strings.ToLower(resPkg.PackageManager), strings.ToLower(resPkg.PackageName))
		if requestedVersion, found := requestedPackagesVersion[key]; found {
			versionMapping[osscache.GenerateCacheKey(resPkg.PackageManager, resPkg.PackageName, resPkg.Version)] = requestedVersion
		}
	}

	return versionMapping
}

// pkgToRequest transforms a parsed package into a scan request.
func pkgToRequest(pkg *models.Package) wrappers.RealtimeScannerPackage {
	return wrappers.RealtimeScannerPackage{
		PackageManager: pkg.PackageManager,
		PackageName:    pkg.PackageName,
		Version:        pkg.Version,
	}
}
