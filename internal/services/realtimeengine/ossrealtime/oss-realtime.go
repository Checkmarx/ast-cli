package ossrealtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Checkmarx/manifest-parser/pkg/parser"
	"github.com/Checkmarx/manifest-parser/pkg/parser/models"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime/dependency_resolver"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime/osscache"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

const (
	pkgManagerGradle    = "gradle"
	pkgManagerSbt       = "sbt"
	pkgManagerMvn       = "mvn"
	pkgManagerCocoapods = "cocoapods"
	pkgManagerCarthage  = "carthage"
	pkgManagerSwift     = "swift"
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
func (o *OssRealtimeService) RunOssRealtimeScan(filePath, ignoredFilePath, sbomFilePath string) (results *OssPackageResults, err error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError("file path is required").Error()
	}

	if enabled, err := realtimeengine.IsFeatureFlagEnabled(o.FeatureFlagWrapper, wrappers.OssRealtimeEnabled); err != nil || !enabled {
		logger.PrintfIfVerbose("Containers Realtime scan is not available (feature flag disabled or error: %v)", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	if err := realtimeengine.EnsureLicense(o.JwtWrapper); err != nil {
		return nil, errorconstants.NewRealtimeEngineError(err.Error()).Error()
	}

	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}

	if err := validateSupportedManifestFile(filePath); err != nil {
		return nil, err
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

	// Enrich with transitive paths
	projectPath := filepath.Dir(filePath)
	if sbomFilePath != "" {
		// Option A: Use provided SBOM file
		if err := enrichWithSbomTransitivePaths(o, response, sbomFilePath); err != nil {
			logger.PrintfIfVerbose("SBOM enrichment skipped: %v", err)
			// POC: never fail the scan on enrichment error
		}
	} else {
		// Option B: Generate transitive deps on-the-fly using native resolvers
		if err := enrichWithGeneratedDeps(o, response, projectPath); err != nil {
			logger.PrintfIfVerbose("Generated deps enrichment skipped: %v", err)
			// Never fail the scan on enrichment error
		}
	}

	if ignoredFilePath != "" {
		ignoredPkgs, err := loadIgnoredPackages(ignoredFilePath)
		if err != nil {
			logger.PrintfIfVerbose("oss-realtime: failed to load ignore file %s: %v; continuing without ignore filtering", ignoredFilePath, err)
		} else {
			ignoreMap := buildIgnoreMap(ignoredPkgs)
			response.Packages = filterIgnoredPackages(response.Packages, ignoreMap)
		}
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
			PackageManager:  entry.PackageManager,
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

// IsSupportedManifestFile reports whether filePath names a manifest file the
// OSS realtime scanner is able to parse and scan. Exported so other callers
// (e.g. the agent-hooks SCA guardrails) can gate on the same rule set instead
// of re-implementing their own list of supported manifest files.
func IsSupportedManifestFile(filePath string) bool {
	return validateSupportedManifestFile(filePath) == nil
}

// validateSupportedManifestFile checks if the manifest file format is supported by OSS realtime scanner.
func validateSupportedManifestFile(filePath string) error {
	manifestFileName := filepath.Base(filePath)
	manifestFileExtension := filepath.Ext(manifestFileName)

	// Check supported extensions
	supportedExtensions := map[string]bool{
		".csproj":  true,
		".sbt":     true,
		".podspec": true,
	}

	// Check supported filenames
	supportedFilenames := map[string]bool{
		"pom.xml":                  true,
		"package.json":             true,
		"bower.json":               true,
		"Directory.Packages.props": true,
		"packages.config":          true,
		"go.mod":                   true,
		"build.gradle":             true,
		"build.gradle.kts":         true,
		"libs.versions.toml":       true,
		"setup.cfg":                true,
		"setup.py":                 true,
		"pyproject.toml":           true,
		"Podfile":                  true,
		"Cartfile":                 true,
		"Cartfile.private":         true,
		"Gemfile":                  true,
		"composer.json":            true,
		"pubspec.yaml":             true,
		"Package.swift":            true,
	}

	// Check by extension
	if supportedExtensions[manifestFileExtension] {
		return nil
	}

	// Check by filename
	if supportedFilenames[manifestFileName] {
		return nil
	}

	// Special handling for .txt files (check prefix)
	if manifestFileExtension == ".txt" {
		if strings.HasPrefix(manifestFileName, "requirement") ||
			strings.HasPrefix(manifestFileName, "packages") ||
			strings.HasPrefix(manifestFileName, "constraint") {
			return nil
		}
	}

	// Special handling for .podspec.json files (CocoaPods pod specifications in JSON format)
	if strings.HasSuffix(manifestFileName, ".podspec.json") {
		return nil
	}

	// Special handling for Package@swift-X.Y.swift multi-toolchain variant files
	if strings.HasPrefix(manifestFileName, "Package@swift-") && strings.HasSuffix(manifestFileName, ".swift") {
		return nil
	}

	// Manifest format is not supported
	return errorconstants.NewRealtimeEngineError(fmt.Sprintf("OSS Realtime scanner doesn't currently support scanning '%s' file.", manifestFileName)).Error()
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
		entry := OssPackage{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			PackageVersion: pkg.Version,
			FilePath:       pkg.FilePath,
			Locations:      convertLocations(pkg.Locations),
		}
		packageMap[generatePackageMapEntry(pkg.PackageManager, pkg.PackageName, pkg.Version)] = entry
		if pkg.PackageManager == pkgManagerGradle || pkg.PackageManager == pkgManagerSbt {
			packageMap[generatePackageMapEntry(pkgManagerMvn, pkg.PackageName, pkg.Version)] = entry
		}
		if pkg.PackageManager == pkgManagerCocoapods || pkg.PackageManager == pkgManagerCarthage {
			packageMap[generatePackageMapEntry(pkgManagerSwift, pkg.PackageName, pkg.Version)] = entry
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
	pkgManager := pkg.PackageManager
	if pkg.PackageManager == pkgManagerGradle || pkg.PackageManager == pkgManagerSbt {
		pkgManager = pkgManagerMvn
	}
	if pkg.PackageManager == pkgManagerCocoapods || pkg.PackageManager == pkgManagerCarthage {
		pkgManager = pkgManagerSwift
	}
	return wrappers.RealtimeScannerPackage{
		PackageManager: pkgManager,
		PackageName:    pkg.PackageName,
		Version:        pkg.Version,
	}
}

// enrichWithGeneratedDeps builds transitive dependencies using native resolvers and enriches response
func enrichWithGeneratedDeps(
	service *OssRealtimeService,
	response *OssPackageResults,
	projectPath string,
) error {
	// Create resolver service
	resolverService := dependency_resolver.NewDependencyResolverService()

	// Resolve dependencies (npm, maven, go)
	deps, result := resolverService.ResolveDependencies(projectPath)

	// Log warnings to user if applicable
	if result.Warning != "" {
		logger.PrintfIfVerbose("⚠️  %s", result.Warning)
	}
	if result.Error != "" {
		logger.PrintfIfVerbose("❌ %s", result.Error)
	}

	// If no transitive deps found, return early (not an error)
	if len(deps) == 0 {
		return nil
	}

	// Extract transitive packages (not direct)
	var transitivePkgs []wrappers.RealtimeScannerPackage
	for _, dep := range deps {
		if !dep.IsDirect {
			transitivePkgs = append(transitivePkgs, wrappers.RealtimeScannerPackage{
				PackageManager: dep.PackageType,
				PackageName:    dep.Name,
				Version:        dep.Version,
			})
		}
	}

	// If no transitive packages, nothing to scan
	if len(transitivePkgs) == 0 {
		logger.PrintfIfVerbose("📦 No transitive dependencies found")
		return nil
	}

	// Scan transitive packages via realtime API
	logger.PrintfIfVerbose("🔍 Scanning %d transitive packages via realtime scanner", len(transitivePkgs))
	req := &wrappers.RealtimeScannerPackageRequest{Packages: transitivePkgs}
	scanResult, err := service.RealtimeScannerWrapper.ScanPackages(req)
	if err != nil {
		return fmt.Errorf("failed to scan transitive packages: %w", err)
	}

	// Enrich response with transitive results (using same logic as SBOM enrichment)
	enrichResponseWithTransitiveDependencies(response, scanResult.Packages, deps)

	return nil
}

// enrichResponseWithTransitiveDependencies enriches response with transitive vulnerability data
func enrichResponseWithTransitiveDependencies(
	response *OssPackageResults,
	vulnPackages []wrappers.RealtimeScannerResults,
	allDeps []dependency_resolver.Dependency,
) {
	// Build dependency graph for path finding
	depGraph := buildDependencyGraph(allDeps)

	for _, vulnPkg := range vulnPackages {
		if vulnPkg.Status == "OK" || len(vulnPkg.Vulnerabilities) == 0 {
			continue // Skip packages without vulnerabilities
		}

		// Find this package in the dependency list
		var depInfo *dependency_resolver.Dependency
		for i := range allDeps {
			if allDeps[i].Name == vulnPkg.PackageName && allDeps[i].Version == vulnPkg.Version {
				depInfo = &allDeps[i]
				break
			}
		}

		if depInfo == nil {
			continue // Package not found in dependency graph
		}

		// Find shortest path from root to this package
		path := findShortestPath(depGraph, depInfo.Name+"@"+depInfo.Version)
		if len(path) < 2 {
			continue // No valid path found
		}

		// Calculate depth (skip root and the package itself)
		depth := len(path) - 2
		if depth < 1 {
			continue // Not really transitive
		}

		// Find introducing direct dependency (first hop after root)
		introducedBy := ""
		if len(path) > 1 {
			introducedBy = path[1] // Second element is the direct dep
		}

		// Calculate boosted severity based on depth
		maxSeverityScore := 0
		for _, vuln := range vulnPkg.Vulnerabilities {
			score := severityToScore(vuln.Severity)
			if score > maxSeverityScore {
				maxSeverityScore = score
			}
		}
		boost := min(20, depth*5)
		boostedScore := maxSeverityScore + boost
		boostedSeverity := scoreToSeverity(boostedScore)

		// Create enriched package entry
		ossPackage := OssPackage{
			PackageManager:  vulnPkg.PackageManager,
			PackageName:     vulnPkg.PackageName,
			PackageVersion:  vulnPkg.Version,
			FilePath:        "", // Transitive packages have no file location
			Locations:       []realtimeengine.Location{},
			Status:          vulnPkg.Status,
			Vulnerabilities: convertVulnerabilities(vulnPkg.Vulnerabilities),
			Transitive:      true,
			DependencyPath:  path,
			IntroducedBy:    introducedBy,
			Depth:           depth,
			BoostedSeverity: boostedSeverity,
			RiskScore:       boostedScore,
		}

		response.Packages = append(response.Packages, ossPackage)
	}
}

// buildDependencyGraph creates an adjacency map for path finding
func buildDependencyGraph(deps []dependency_resolver.Dependency) map[string][]string {
	graph := make(map[string][]string)
	for _, dep := range deps {
		key := dep.Name + "@" + dep.Version
		for _, child := range dep.Children {
			graph[key] = append(graph[key], child)
		}
	}
	return graph
}

// findShortestPath finds path from root to target package using BFS
func findShortestPath(graph map[string][]string, target string) []string {
	// BFS to find shortest path
	type queueItem struct {
		node string
		path []string
	}

	queue := []queueItem{{node: "root", path: []string{"root"}}}
	visited := make(map[string]bool)
	visited["root"] = true

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.node == target {
			return item.path
		}

		// Get children from graph
		children := graph[item.node]
		for _, child := range children {
			if !visited[child] {
				visited[child] = true
				newPath := append(item.path, child)
				queue = append(queue, queueItem{node: child, path: newPath})
			}
		}
	}

	return nil // No path found
}

// Severity scoring helpers
func severityToScore(severity string) int {
	switch severity {
	case "CRITICAL":
		return 100
	case "HIGH":
		return 80
	case "MEDIUM":
		return 50
	case "LOW":
		return 20
	default:
		return 10
	}
}

func scoreToSeverity(score int) string {
	switch {
	case score >= 100:
		return "CRITICAL"
	case score >= 80:
		return "HIGH"
	case score >= 50:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
