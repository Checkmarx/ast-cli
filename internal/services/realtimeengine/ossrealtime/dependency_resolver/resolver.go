package dependency_resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
)

// DependencyResolver interface for pluggable implementations
type DependencyResolver interface {
	ResolveDependencies(projectPath string) (*DependencyTreeResult, error)
	SupportedFiles() []string
}

// DependencyResolverService orchestrates all resolvers. Each resolved dependency
// tree is cached on disk (see dependency_tree_cache.go), keyed by package manager +
// project path and validated against a content hash of the manifest/lock file, so
// repeated scans of an unchanged project skip re-invoking the build tool.
type DependencyResolverService struct {
	resolvers map[string]DependencyResolver
}

// NewDependencyResolverService creates a new service with all resolvers
func NewDependencyResolverService() *DependencyResolverService {
	return &DependencyResolverService{
		resolvers: map[string]DependencyResolver{
			"npm":   &NpmResolver{},
			"maven": &MavenResolver{},
			"go":    &GoResolver{},
		},
	}
}

// ResolveDependencies detects package managers and resolves all dependencies with caching
func (s *DependencyResolverService) ResolveDependencies(projectPath string) ([]Dependency, ResolutionResult) {
	var allDeps []Dependency
	result := ResolutionResult{Success: false}

	// NPM Resolution
	if fileExists(filepath.Join(projectPath, "package.json")) {
		lockFilePath := filepath.Join(projectPath, "package-lock.json")
		if !fileExists(lockFilePath) {
			result.Warning = "package-lock.json not found; npm transitive deps skipped. " +
				"Run 'npm install' to generate lock file for full coverage."
			logger.PrintfIfVerbose("⚠️  %s", result.Warning)
		} else if deps, err := s.resolveWithCache("npm", projectPath, lockFilePath); err != nil {
			result.Error = fmt.Sprintf("npm resolution failed: %v", err)
			logger.PrintfIfVerbose("❌ %s", result.Error)
		} else {
			allDeps = append(allDeps, deps...)
			result.Success = true
		}
	}

	// Maven Resolution
	if fileExists(filepath.Join(projectPath, "pom.xml")) {
		if !commandExists("mvn") {
			result.Warning = "Maven not found; maven transitive deps skipped. " +
				"Install Maven (apt-get install maven) for full coverage."
			logger.PrintfIfVerbose("⚠️  %s", result.Warning)
		} else if deps, err := s.resolveWithCache("maven", projectPath, filepath.Join(projectPath, "pom.xml")); err != nil {
			result.Error = fmt.Sprintf("maven resolution failed: %v", err)
			logger.PrintfIfVerbose("❌ %s", result.Error)
		} else {
			allDeps = append(allDeps, deps...)
			result.Success = true
		}
	}

	// Go Resolution
	if fileExists(filepath.Join(projectPath, "go.mod")) {
		sumFilePath := filepath.Join(projectPath, "go.sum")
		if !fileExists(sumFilePath) {
			result.Warning = "go.sum not found; go transitive deps skipped. " +
				"Run 'go mod download' to generate sum file for full coverage."
			logger.PrintfIfVerbose("⚠️  %s", result.Warning)
		} else if !commandExists("go") {
			result.Warning = "Go not found; go transitive deps skipped. " +
				"Install Go (https://golang.org/dl) for full coverage."
			logger.PrintfIfVerbose("⚠️  %s", result.Warning)
		} else if deps, err := s.resolveWithCache("go", projectPath, sumFilePath); err != nil {
			result.Error = fmt.Sprintf("go resolution failed: %v", err)
			logger.PrintfIfVerbose("❌ %s", result.Error)
		} else {
			allDeps = append(allDeps, deps...)
			result.Success = true
		}
	}

	result.Dependencies = allDeps
	return allDeps, result
}

// resolveWithCache checks the on-disk dependency-tree cache first (keyed by package
// manager + project path, validated against a content hash of checkFilePath), then
// falls back to the resolver for pkgMgr on a cache miss.
func (s *DependencyResolverService) resolveWithCache(pkgMgr, projectPath, checkFilePath string) ([]Dependency, error) {
	resolver, ok := s.resolvers[pkgMgr]
	if !ok {
		return nil, fmt.Errorf("unknown package manager: %s", pkgMgr)
	}

	key := treeCacheKey(pkgMgr, projectPath)
	manifestHash := computeManifestHash(pkgMgr, checkFilePath)

	if deps, found := readTreeCache(key, manifestHash); found {
		logger.PrintfIfVerbose("oss-realtime: using cached %s dependency tree", pkgMgr)
		return deps, nil
	}

	result, err := resolver.ResolveDependencies(projectPath)
	if err != nil {
		return nil, err
	}

	if cacheErr := writeTreeCache(key, manifestHash, result.Dependencies); cacheErr != nil {
		logger.PrintfIfVerbose("oss-realtime: failed to update dependency tree cache: %v", cacheErr)
	}

	logger.PrintfIfVerbose("oss-realtime: resolved %s dependency tree (%d total)", pkgMgr, len(result.Dependencies))
	return result.Dependencies, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
