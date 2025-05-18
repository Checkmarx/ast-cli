package ossrealtime

import (
	"log"

	"github.com/Checkmarx/manifest-parser/pkg/models"
	"github.com/Checkmarx/manifest-parser/pkg/parser"
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type RealtimeScannerWrapperParams struct {
	JwtWrapper             wrappers.JWTWrapper
	FeatureFlagWrapper     wrappers.FeatureFlagsWrapper
	RealtimeScannerWrapper wrappers.RealtimeScannerWrapper
}

// RunOssRealtimeScan performs an OSS realtime scan on the given manifest file.
func RunOssRealtimeScan(realtimeScannerWrapperParams *RealtimeScannerWrapperParams, filePath string) (*wrappers.OssPackageResponse, error) {
	if filePath == "" {
		return nil, nil
	}

	if err := ensureLicense(realtimeScannerWrapperParams); err != nil {
		return nil, err
	}

	pkgs, err := parseManifest(filePath)
	if err != nil {
		return nil, err
	}

	response, toScan := prepareScan(pkgs)
	if len(toScan.Packages) > 0 {
		if err := scanAndCache(realtimeScannerWrapperParams, toScan, response); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func ensureLicense(realtimeScannerWrapperParams *RealtimeScannerWrapperParams) error {
	// For the 60-day delivery, there is no license check
	return nil
}

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

// prepareScan processes a list of packages, separates cached and uncached packages, and returns both response and request data to be sent to the scanner.
func prepareScan(pkgs []models.Package) (*wrappers.OssPackageResponse, *wrappers.OssPackageRequest) {
	var resp wrappers.OssPackageResponse
	var req wrappers.OssPackageRequest

	resp.Packages = make([]wrappers.OssResults, 0, len(pkgs))

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
		if status, found := cacheMap[key]; found {
			resp.Packages = append(resp.Packages, wrappers.OssResults{
				PackageManager: pkg.PackageManager,
				PackageName:    pkg.PackageName,
				Version:        pkg.Version,
				Status:         status,
			})
		} else {
			req.Packages = append(req.Packages, pkgToRequest(&pkg))
		}
	}
	return &resp, &req
}

// scanAndCache performs a scan on the provided packages and caches the results.
func scanAndCache(realtimeScannerWrapperParams *RealtimeScannerWrapperParams, requestPackages *wrappers.OssPackageRequest, resp *wrappers.OssPackageResponse) error {
	result, err := realtimeScannerWrapperParams.RealtimeScannerWrapper.Scan(requestPackages)
	if err != nil {
		return errors.Wrap(err, "scanning packages via realtime service")
	}
	if len(result.Packages) == 0 {
		return errors.New("empty response from oss-realtime scan")
	}

	for _, pkg := range result.Packages {
		resp.Packages = append(resp.Packages, wrappers.OssResults{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			Version:        pkg.Version,
			Status:         pkg.Status,
		})
	}

	if err = osscache.AppendToCache(result); err != nil {
		log.Printf("ossrealtime: failed to update cache: %v", err)
	}
	return nil
}

// pkgToRequest transforms a parsed package into a scan request.
func pkgToRequest(pkg *models.Package) wrappers.OssPackage {
	return wrappers.OssPackage{
		PackageManager: pkg.PackageManager,
		PackageName:    pkg.PackageName,
		Version:        pkg.Version,
	}
}
