package ossrealtime

import (
	"fmt"
	"log"
	"strings"

	"github.com/Checkmarx/manifest-parser/pkg/parser"
	"github.com/Checkmarx/manifest-parser/pkg/parser/models"
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

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
		return nil, errors.New("file path is required")
	}

	//if enabled, err := o.isFeatureFlagEnabled(); err != nil || !enabled {
	//	return nil, err
	//}

	if err := o.ensureLicense(); err != nil {
		return nil, err
	}

	pkgs, err := parseManifest(filePath)
	if err != nil {
		return nil, err
	}

	response, toScan := prepareScan(pkgs)

	if len(toScan.Packages) > 0 {
		result, err := o.scanAndCache(toScan)
		if err != nil {
			return nil, errors.Wrap(err, "scanning packages via realtime service")
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
	for _, pkg := range result.Packages {
		entry := getPackageEntryFromPackageMap(packageMap, &pkg)
		response.Packages = append(response.Packages, OssPackage{
			PackageManager:  pkg.PackageManager,
			PackageName:     pkg.PackageName,
			PackageVersion:  pkg.Version,
			FilePath:        entry.FilePath,
			LineStart:       entry.LineStart,
			LineEnd:         entry.LineEnd,
			StartIndex:      entry.StartIndex,
			EndIndex:        entry.EndIndex,
			Status:          pkg.Status,
			Vulnerabilities: NewOssVulnerabilitiesFromRealtimeScannerVulnerabilities(pkg.Vulnerabilities),
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
		return errors.New("jwt wrapper not provided")
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
	var req wrappers.RealtimeScannerPackageRequest

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
				LineStart:       pkg.LineStart,
				LineEnd:         pkg.LineEnd,
				FilePath:        pkg.FilePath,
				StartIndex:      pkg.StartIndex,
				EndIndex:        pkg.EndIndex,
				Status:          cachedPkg.Status,
				Vulnerabilities: NewOssVulnerabilitiesFromOssCacheVulnerabilities(cachedPkg.Vulnerabilities),
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
			LineStart:      pkg.LineStart,
			LineEnd:        pkg.LineEnd,
			StartIndex:     pkg.StartIndex,
			EndIndex:       pkg.EndIndex,
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
		return nil, errors.Wrap(err, "scanning packages via realtime service")
	}
	if len(result.Packages) == 0 {
		return nil, errors.New("empty response from oss-realtime scan")
	}

	requestPackageMap := make(map[string]wrappers.RealtimeScannerPackage)
	for _, pkg := range requestPackages.Packages {
		key := fmt.Sprintf("%s|%s", strings.ToLower(pkg.PackageManager), strings.ToLower(pkg.PackageName))
		requestPackageMap[key] = pkg
	}

	versionMapping := make(map[string]osscache.VersionMapping)
	for _, resPkg := range result.Packages {
		key := fmt.Sprintf("%s|%s", strings.ToLower(resPkg.PackageManager), strings.ToLower(resPkg.PackageName))
		if pkg, found := requestPackageMap[key]; found {
			versionMapping[osscache.GenerateCacheKey(pkg.PackageManager, pkg.PackageName, resPkg.Version)] = osscache.VersionMapping{
				RequestedVersion: pkg.Version,
				ActualVersion:    resPkg.Version,
			}
		}
	}

	if err := osscache.AppendToCache(result, versionMapping); err != nil {
		log.Printf("ossrealtime: failed to update cache: %v", err)
	}

	return result, nil
}

// pkgToRequest transforms a parsed package into a scan request.
func pkgToRequest(pkg *models.Package) wrappers.RealtimeScannerPackage {
	return wrappers.RealtimeScannerPackage{
		PackageManager: pkg.PackageManager,
		PackageName:    pkg.PackageName,
		Version:        pkg.Version,
	}
}
