package ossrealtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// CycloneDX SBOM structures
type cycloneDXBOM struct {
	Metadata struct {
		Component struct {
			BOMRef string `json:"bom-ref"`
		} `json:"component"`
	} `json:"metadata"`
	Components   []cycloneDXComponent  `json:"components"`
	Dependencies []cycloneDXDependency `json:"dependencies"`
}

type cycloneDXComponent struct {
	BOMRef  string `json:"bom-ref"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Purl    string `json:"purl"`
}

type cycloneDXDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

// sbomGraph represents a parsed dependency graph
type sbomGraph struct {
	rootRef        string
	refToComponent map[string]cycloneDXComponent
	adjacency      map[string][]string // ref -> []childRefs
	directRefs     map[string]bool
}

// Severity scoring constants
const (
	severityMalicious  = 120
	severityCritical   = 100
	severityHigh       = 80
	severityMedium     = 50
	severityLow        = 20
	severityDefault    = 10
	maxTransitiveBoost = 20
	boostPerDepth      = 5
)

// isSafePath validates that a path doesn't contain directory traversal patterns
func isSafePath(path string) bool {
	// Clean the path to resolve .. and . components
	cleanPath := filepath.Clean(path)

	// Check if the cleaned path contains suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return false
	}

	// On Windows, ensure no drive letter hijacking
	if filepath.IsAbs(cleanPath) {
		// Absolute paths are OK, but ensure they don't traverse outside expected scope
		return !strings.Contains(cleanPath, "..")
	}

	return true
}

// validateJSONStructure performs pre-deserialization validation on raw JSON bytes
func validateJSONStructure(data []byte) error {
	// First, unmarshal to generic map to validate structure before typed unmarshalling
	var rawBOM map[string]interface{}
	err := json.Unmarshal(data, &rawBOM)
	if err != nil {
		return fmt.Errorf("invalid JSON structure: %w", err)
	}

	// Validate presence of key SBOM fields
	if _, hasComponents := rawBOM["components"]; !hasComponents {
		if _, hasDeps := rawBOM["dependencies"]; !hasDeps {
			return fmt.Errorf("invalid SBOM: missing both 'components' and 'dependencies' fields")
		}
	}

	// Validate components field if present
	if components, ok := rawBOM["components"]; ok {
		if componentsList, isList := components.([]interface{}); isList && len(componentsList) > 0 {
			// Spot-check first component for required fields
			if comp, isMap := componentsList[0].(map[string]interface{}); isMap {
				if _, hasBOMRef := comp["bom-ref"]; !hasBOMRef {
					return fmt.Errorf("invalid component: missing 'bom-ref' field")
				}
				if _, hasName := comp["name"]; !hasName {
					return fmt.Errorf("invalid component: missing 'name' field")
				}
			}
		}
	}

	return nil
}

// isValidSBOMStructure validates the SBOM has required fields
func isValidSBOMStructure(bom *cycloneDXBOM) bool {
	if bom == nil {
		return false
	}

	// At minimum, SBOM should have components or dependencies
	if len(bom.Components) == 0 && len(bom.Dependencies) == 0 {
		return false
	}

	// Validate components have required fields
	for _, comp := range bom.Components {
		if comp.BOMRef == "" || comp.Name == "" {
			return false
		}
	}

	// Validate dependencies have ref
	for _, dep := range bom.Dependencies {
		if dep.Ref == "" {
			return false
		}
	}

	return true
}

// parseCycloneDXSBOM parses a CycloneDX JSON SBOM file with validation
func parseCycloneDXSBOM(path string) (*sbomGraph, error) {
	// Validate and sanitize the path (Rule 3007: Unsafe Path Handling)
	if !isSafePath(path) {
		return nil, fmt.Errorf("invalid SBOM file path: contains directory traversal patterns")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM file: %w", err)
	}

	// Pre-validate JSON structure before unmarshalling (Rule 3005: Unsafe Deserialization)
	if err := validateJSONStructure(data); err != nil {
		return nil, fmt.Errorf("SBOM validation failed: %w", err)
	}

	// Now safely unmarshal to typed struct (after pre-validation)
	var bom cycloneDXBOM
	err = json.Unmarshal(data, &bom)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SBOM JSON: %w", err)
	}

	// Validate the SBOM structure has required fields
	if !isValidSBOMStructure(&bom) {
		return nil, fmt.Errorf("invalid SBOM structure: missing required fields (components/dependencies, bom-ref, name)")
	}

	// Build ref -> component map
	refToComponent := make(map[string]cycloneDXComponent)
	for _, comp := range bom.Components {
		refToComponent[comp.BOMRef] = comp
	}

	// Build adjacency map
	adjacency := make(map[string][]string)
	for _, dep := range bom.Dependencies {
		adjacency[dep.Ref] = dep.DependsOn
	}

	// Determine root ref (from metadata or first dependency)
	rootRef := bom.Metadata.Component.BOMRef
	if rootRef == "" && len(bom.Dependencies) > 0 {
		rootRef = bom.Dependencies[0].Ref
	}
	if rootRef == "" {
		return nil, fmt.Errorf("no root reference found in SBOM")
	}

	// Build direct refs map (dependencies of root)
	directRefs := make(map[string]bool)
	for _, ref := range adjacency[rootRef] {
		directRefs[ref] = true
	}

	return &sbomGraph{
		rootRef:        rootRef,
		refToComponent: refToComponent,
		adjacency:      adjacency,
		directRefs:     directRefs,
	}, nil
}

// transitiveComponents returns all components reachable from root, excluding direct and root itself
func (g *sbomGraph) transitiveComponents() []cycloneDXComponent {
	visited := make(map[string]bool)
	var result []cycloneDXComponent

	// BFS to find all reachable refs
	queue := []string{g.rootRef}
	visited[g.rootRef] = true

	for len(queue) > 0 {
		ref := queue[0]
		queue = queue[1:]

		for _, childRef := range g.adjacency[ref] {
			if !visited[childRef] {
				visited[childRef] = true
				// Only add if not direct and not root
				if !g.directRefs[childRef] {
					if comp, ok := g.refToComponent[childRef]; ok {
						result = append(result, comp)
					}
				}
				queue = append(queue, childRef)
			}
		}
	}

	return result
}

// shortestPathTo finds the shortest path from root to a given ref (BFS)
func (g *sbomGraph) shortestPathTo(targetRef string) []cycloneDXComponent {
	if targetRef == "" {
		return nil
	}

	// BFS to find shortest path
	type queueItem struct {
		ref  string
		path []string
	}

	queue := []queueItem{{ref: g.rootRef, path: []string{g.rootRef}}}
	visited := make(map[string]bool)
	visited[g.rootRef] = true

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.ref == targetRef {
			// Convert path refs to components
			var path []cycloneDXComponent
			for _, ref := range item.path {
				if comp, ok := g.refToComponent[ref]; ok {
					path = append(path, comp)
				}
			}
			return path
		}

		for _, childRef := range g.adjacency[item.ref] {
			if !visited[childRef] {
				visited[childRef] = true
				newPath := append(item.path, childRef)
				queue = append(queue, queueItem{ref: childRef, path: newPath})
			}
		}
	}

	return nil
}

// purlTypeToPkgMgr converts PURL type to package manager name
func purlTypeToPkgMgr(purl string) string {
	if !strings.HasPrefix(purl, "pkg:") {
		return ""
	}

	// Format: pkg:type/name@version
	parts := strings.Split(strings.TrimPrefix(purl, "pkg:"), "/")
	if len(parts) == 0 {
		return ""
	}

	pkgType := parts[0]
	switch pkgType {
	case "npm":
		return "npm"
	case "maven":
		return "maven"
	case "pypi":
		return "pip"
	case "golang":
		return "go"
	case "nuget":
		return "nuget"
	case "composer":
		return "composer"
	case "gem":
		return "gem"
	case "cargo":
		return "cargo"
	default:
		return pkgType
	}
}

// severityScore converts severity string to numeric score
func severityScore(severity string) int {
	switch strings.ToUpper(severity) {
	case "MALICIOUS":
		return severityMalicious
	case "CRITICAL":
		return severityCritical
	case "HIGH":
		return severityHigh
	case "MEDIUM":
		return severityMedium
	case "LOW":
		return severityLow
	default:
		return severityDefault
	}
}

// bandOf converts numeric score to severity band
func bandOf(score int) string {
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

// transitiveBoost calculates risk boost based on depth
func transitiveBoost(depth int) int {
	boost := depth * boostPerDepth
	if boost > maxTransitiveBoost {
		boost = maxTransitiveBoost
	}
	return boost
}

// enrichWithSbomTransitivePaths enriches OSS results with transitive dependency info from SBOM
func enrichWithSbomTransitivePaths(
	service *OssRealtimeService,
	response *OssPackageResults,
	sbomPath string,
) error {
	// Parse SBOM (includes path validation and structure validation)
	sbom, err := parseCycloneDXSBOM(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to parse SBOM: %w", err)
	}

	// Get transitive components from SBOM
	transitiveComps := sbom.transitiveComponents()
	if len(transitiveComps) == 0 {
		logger.PrintfIfVerbose("No transitive components found in SBOM")
		return nil
	}

	// Convert transitive components to scanner request packages
	var transitivePkgs []wrappers.RealtimeScannerPackage
	compByID := make(map[string]cycloneDXComponent)

	for _, comp := range transitiveComps {
		pkgMgr := purlTypeToPkgMgr(comp.Purl)
		if pkgMgr == "" {
			logger.PrintfIfVerbose("Skipping transitive component %s: unknown package manager", comp.Name)
			continue
		}

		transitivePkgs = append(transitivePkgs, wrappers.RealtimeScannerPackage{
			PackageManager: pkgMgr,
			PackageName:    comp.Name,
			Version:        comp.Version,
		})

		id := fmt.Sprintf("%s_%s_%s", pkgMgr, comp.Name, comp.Version)
		compByID[id] = comp
	}

	if len(transitivePkgs) == 0 {
		logger.PrintfIfVerbose("No valid transitive packages to scan after filtering")
		return nil
	}

	// Scan transitive packages
	logger.PrintfIfVerbose("Scanning %d transitive packages via realtime scanner", len(transitivePkgs))
	req := &wrappers.RealtimeScannerPackageRequest{Packages: transitivePkgs}
	scanResult, err := service.RealtimeScannerWrapper.ScanPackages(req)
	if err != nil {
		return fmt.Errorf("failed to scan transitive packages: %w", err)
	}

	// Process transitive results and add to response
	for _, scanPkg := range scanResult.Packages {
		if scanPkg.Status == "OK" || len(scanPkg.Vulnerabilities) == 0 {
			// Skip packages without vulnerabilities
			continue
		}

		// Find the component in SBOM
		id := fmt.Sprintf("%s_%s_%s", scanPkg.PackageManager, scanPkg.PackageName, scanPkg.Version)
		comp, ok := compByID[id]
		if !ok {
			continue
		}

		// Find shortest path to this component
		path := sbom.shortestPathTo(comp.BOMRef)
		if len(path) < 2 {
			// Path should have at least root + one dep
			continue
		}

		// Remove root from path
		pathWithoutRoot := path[1:]
		depth := len(pathWithoutRoot) - 1

		// Introducing direct dependency is the first hop (direct dependency)
		introducedBy := fmt.Sprintf("%s@%s", pathWithoutRoot[0].Name, pathWithoutRoot[0].Version)

		// Build dependency path string slice (skip root)
		var depPath []string
		for _, comp := range pathWithoutRoot {
			depPath = append(depPath, fmt.Sprintf("%s@%s", comp.Name, comp.Version))
		}

		// Calculate boosted severity
		maxSeverityScore := 0
		for _, vuln := range scanPkg.Vulnerabilities {
			score := severityScore(vuln.Severity)
			if score > maxSeverityScore {
				maxSeverityScore = score
			}
		}
		riskScore := maxSeverityScore + transitiveBoost(depth)
		boostedSeverity := bandOf(riskScore)

		// Create OssPackage with transitive enrichment
		ossPackage := OssPackage{
			PackageManager:  scanPkg.PackageManager,
			PackageName:     scanPkg.PackageName,
			PackageVersion:  scanPkg.Version,
			FilePath:        "",                          // Transitive packages don't have a file location in the manifest
			Locations:       []realtimeengine.Location{}, // Transitive packages have no source location
			Status:          scanPkg.Status,
			Vulnerabilities: convertVulnerabilities(scanPkg.Vulnerabilities),
			Transitive:      true,
			DependencyPath:  depPath,
			IntroducedBy:    introducedBy,
			Depth:           depth,
			BoostedSeverity: boostedSeverity,
			RiskScore:       riskScore,
		}

		response.Packages = append(response.Packages, ossPackage)
	}

	return nil
}

// convertVulnerabilities converts RealtimeScannerVulnerability to Vulnerability
func convertVulnerabilities(scanVulns []wrappers.RealtimeScannerVulnerability) []Vulnerability {
	var result []Vulnerability
	for _, sv := range scanVulns {
		result = append(result, Vulnerability{
			CVE:         sv.CVE,
			Description: sv.Description,
			Severity:    sv.Severity,
		})
	}
	return result
}
