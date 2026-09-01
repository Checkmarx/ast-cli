package dependency_resolver

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
)

// DependencyResolver interface for pluggable implementations
type DependencyResolver interface {
	ResolveDependencies(projectPath string) (*DependencyTreeResult, error)
	SupportedFiles() []string
}

// CacheEntry holds cached dependency results
type CacheEntry struct {
	Dependencies []Dependency
	Timestamp    time.Time
	FileHash     string
}

// DependencyResolverService orchestrates all resolvers
type DependencyResolverService struct {
	npmResolver   DependencyResolver
	mavenResolver DependencyResolver
	goResolver    DependencyResolver

	// Caching
	cache      map[string]CacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewDependencyResolverService creates a new service with all resolvers
func NewDependencyResolverService() *DependencyResolverService {
	return &DependencyResolverService{
		npmResolver:   &NpmResolver{},
		mavenResolver: &MavenResolver{},
		goResolver:    &GoResolver{},
		cache:         make(map[string]CacheEntry),
		cacheTTL:      5 * time.Minute, // Cache for 5 minutes
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
		} else {
			deps, err := s.resolveWithCache("npm", projectPath, lockFilePath)
			if err != nil {
				result.Error = fmt.Sprintf("npm resolution failed: %v", err)
				logger.PrintfIfVerbose("❌ %s", result.Error)
			} else {
				allDeps = append(allDeps, deps...)
				result.Success = true
			}
		}
	}

	// Maven Resolution
	if fileExists(filepath.Join(projectPath, "pom.xml")) {
		if !commandExists("mvn") {
			result.Warning = "Maven not found; maven transitive deps skipped. " +
				"Install Maven (apt-get install maven) for full coverage."
			logger.PrintfIfVerbose("⚠️  %s", result.Warning)
		} else {
			pomPath := filepath.Join(projectPath, "pom.xml")
			deps, err := s.resolveWithCache("maven", projectPath, pomPath)
			if err != nil {
				result.Error = fmt.Sprintf("maven resolution failed: %v", err)
				logger.PrintfIfVerbose("❌ %s", result.Error)
			} else {
				allDeps = append(allDeps, deps...)
				result.Success = true
			}
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
		} else {
			deps, err := s.resolveWithCache("go", projectPath, sumFilePath)
			if err != nil {
				result.Error = fmt.Sprintf("go resolution failed: %v", err)
				logger.PrintfIfVerbose("❌ %s", result.Error)
			} else {
				allDeps = append(allDeps, deps...)
				result.Success = true
			}
		}
	}

	result.Dependencies = allDeps
	return allDeps, result
}

// resolveWithCache checks cache first, then calls appropriate resolver
func (s *DependencyResolverService) resolveWithCache(
	pkgMgr, projectPath, checkFilePath string,
) ([]Dependency, error) {
	// Generate cache key
	cacheKey := s.generateCacheKey(pkgMgr, projectPath, checkFilePath)

	// Check cache
	if cached := s.getFromCache(cacheKey, checkFilePath); cached != nil {
		logger.PrintfIfVerbose("📦 Using cached %s dependencies", pkgMgr)
		return cached, nil
	}

	// Resolve based on package manager
	var deps []Dependency

	switch pkgMgr {
	case "npm":
		result, err := s.npmResolver.ResolveDependencies(projectPath)
		if err != nil {
			return nil, err
		}
		deps = result.Dependencies
	case "maven":
		result, err := s.mavenResolver.ResolveDependencies(projectPath)
		if err != nil {
			return nil, err
		}
		deps = result.Dependencies
	case "go":
		result, err := s.goResolver.ResolveDependencies(projectPath)
		if err != nil {
			return nil, err
		}
		deps = result.Dependencies
	default:
		return nil, fmt.Errorf("unknown package manager: %s", pkgMgr)
	}

	// Store in cache
	fileHash := s.hashFile(checkFilePath)
	s.setInCache(cacheKey, deps, fileHash)

	logger.Printf("✅ Resolved %s dependencies (%d total)", pkgMgr, len(deps))
	return deps, nil
}

// generateCacheKey creates a cache key from package manager and project path
func (s *DependencyResolverService) generateCacheKey(pkgMgr, projectPath, filePath string) string {
	return fmt.Sprintf("%s:%s", pkgMgr, projectPath)
}

// hashFile computes MD5 hash of a file for cache invalidation
func (s *DependencyResolverService) hashFile(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// getFromCache retrieves dependencies from cache if valid
func (s *DependencyResolverService) getFromCache(key, filePath string) []Dependency {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, exists := s.cache[key]
	if !exists {
		return nil
	}

	// Check if cache expired
	if time.Since(entry.Timestamp) > s.cacheTTL {
		return nil
	}

	// Check if file has changed (cache invalidation)
	currentHash := s.hashFile(filePath)
	if currentHash != entry.FileHash {
		return nil
	}

	return entry.Dependencies
}

// setInCache stores dependencies in cache
func (s *DependencyResolverService) setInCache(key string, deps []Dependency, fileHash string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache[key] = CacheEntry{
		Dependencies: deps,
		Timestamp:    time.Now(),
		FileHash:     fileHash,
	}
}

// ClearCache clears all cached entries (for testing or manual refresh)
func (s *DependencyResolverService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = make(map[string]CacheEntry)
}

// Helper functions

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
