package ossrealtime

import (
	"encoding/json"
	"fmt"
	"os"
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

// convertLocations converts models.Location to realtimeengine.Location
func convertLocations(locations []models.Location) []realtimeengine.Location {
	var result []realtimeengine.Location
	for _, loc := range locations {
		result = append(result, realtimeengine.Location{
			Line:       loc.Line,
			StartIndex: loc.StartIndex,
			EndIndex:   loc.EndIndex,
		})
	}
	return result
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
func (o *OssRealtimeService) RunOssRealtimeScan(filePath, ignoredFilePath string) (results *OssPackageResults, err error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError("file path is required").Error()
	}

	if enabled, err := o.isFeatureFlagEnabled(); err != nil || !enabled {
		logger.PrintfIfVerbose("Containers Realtime scan is not available (feature flag disabled or error: %v)", err)
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

	if ignoredFilePath != "" {
		ignoredPkgs, err := loadIgnoredPackages(ignoredFilePath)
		if err != nil {
			return nil, errorconstants.NewRealtimeEngineError("failed to load ignored packages").Error()
		}

		ignoreMap := buildIgnoreMap(ignoredPkgs)
		response.Packages = filterIgnoredPackages(response.Packages, ignoreMap)
	}

	return response, nil
}

func buildIgnoreMap(ignored []IgnoredPackage) map[string]bool {
	m := make(map[string]bool)
	for _, ign := range ignored {
		m[ign.GetID()] = true
	}
	return m
}

func isIgnored(pkg *OssPackage, ignoreMap map[string]bool) bool {
	return ignoreMap[pkg.GetID()]
}

func loadIgnoredPackages(path string) ([]IgnoredPackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ignored []IgnoredPackage
	err = json.Unmarshal(data, &ignored)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func filterIgnoredPackages(packages []OssPackage, ignoreMap map[string]bool) []OssPackage {
	filtered := make([]OssPackage, 0, len(packages))
	for i := range packages {
		pkg := &packages[i]
		if !isIgnored(pkg, ignoreMap) {
			filtered = append(filtered, *pkg)
		}
	}
	return filtered
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
				Locations:       convertLocations(pkg.Locations),
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
			Locations:      convertLocations(pkg.Locations),
		}
	}
	return packageMap
}

// generatePackageMapEntry generates a unique key for the package map.
func generatePackageMapEntry(pkgManager, pkgName, pkgVersion string) string {
	return fmt.Sprintf("%s_%s_%s", pkgManager, pkgName, pkgVersion)
}

// scanAndCache performs a scan on the provided packages and caches the results.
func (o *OssRealtimeService) scanAndCache(requestPackages *wrappers.RealtimeScannerPackageRequest) (results *wrappers.RealtimeScannerPackageResponse, err error) {
	result, err := o.RealtimeScannerWrapper.ScanPackages(requestPackages)
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
