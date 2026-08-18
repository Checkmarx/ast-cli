# Architecture: Pure Go Dependency Resolver for ast-cli

**Branch:** `feature/go-dependency-resolver`

**Goal:** Implement native Go resolvers for npm, Maven, and Go modules to build complete dependency trees with transitive dependencies without external binaries.

**Status:** Architecture Design (Awaiting Approval)

---

## 1. Overview

### What This Solves
- Direct dependencies only (manifest parsing) → Complete transitive dependency tree
- Reuses existing Phase 1 SBOM enrichment logic
- No dependency on SCA Resolver binary
- Pure Go implementation

### Supported Package Managers
| Manager | File | Detection | Method |
|---------|------|-----------|--------|
| **npm** | `package-lock.json` | ✅ Check file exists | Parse JSON recursively |
| **Maven** | `pom.xml` | ✅ Check file exists | Run `mvn dependency:tree` |
| **Go** | `go.mod` | ✅ Check file exists | Run `go mod graph` |

### Languages Covered
- **JavaScript/TypeScript** (npm)
- **Java** (Maven, Gradle future)
- **Go**
- **Total coverage:** ~60-70% of enterprise projects

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          ast-cli                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  cx scan oss-realtime --file-source ./package.json             │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ oss-realtime-engine.go (existing)                        │  │
│  │   RunScanOssRealtimeCommand()                            │  │
│  │     └─ call: ossRealtimeService.RunOssRealtimeScan()    │  │
│  └────────────────┬─────────────────────────────────────────┘  │
│                   │                                             │
│  ┌────────────────▼─────────────────────────────────────────┐  │
│  │ oss-realtime.go (MODIFIED - Phase 1)                     │  │
│  │   RunOssRealtimeScan(filePath, ignoredPath, sbomPath)   │  │
│  │                                                          │  │
│  │   1. Scan direct dependencies                           │  │
│  │      └─ uses RealtimeScannerWrapper.ScanPackages()     │  │
│  │                                                          │  │
│  │   2. Enrich with transitive (NEW - Phase 1)            │  │
│  │      └─ if sbomPath: enrichWithSbomTransitivePaths()   │  │
│  │         OR (NEW - Phase 3.5)                            │  │
│  │      └─ if !sbomPath: enrichWithGeneratedDeps()         │  │
│  │         (calls NEW dependency resolver)                 │  │
│  │                                                          │  │
│  │   3. Return enriched OssPackageResults                   │  │
│  └────────────────┬─────────────────────────────────────────┘  │
│                   │                                             │
│  ┌────────────────▼─────────────────────────────────────────┐  │
│  │ NEW: dependency_resolver/ (Phase 3.5)                    │  │
│  │   resolver.go                                            │  │
│  │   ├─ interface: DependencyResolver                       │  │
│  │   ├─ DetectPackageManagers(path)                         │  │
│  │   └─ ResolveDependencies(path) → []Dependency           │  │
│  │                                                          │  │
│  │   npm_resolver.go                                        │  │
│  │   ├─ ParseNpmLockFile(path)                             │  │
│  │   └─ traverseNpmDeps(recursive)                          │  │
│  │                                                          │  │
│  │   maven_resolver.go                                      │  │
│  │   ├─ ExecuteMavenDependencyTree(path)                    │  │
│  │   └─ ParseMavenTreeOutput(text)                          │  │
│  │                                                          │  │
│  │   go_resolver.go                                         │  │
│  │   ├─ ExecuteGoModGraph(path)                             │  │
│  │   └─ ParseGoModGraphOutput(text)                         │  │
│  └────────────────┬─────────────────────────────────────────┘  │
│                   │                                             │
│  ┌────────────────▼─────────────────────────────────────────┐  │
│  │ sbom_enrichment.go (existing Phase 1)                    │  │
│  │   enrichWithSbomTransitivePaths()                        │  │
│  │   └─ Now can work with generated deps too!              │  │
│  └────────────────┬─────────────────────────────────────────┘  │
│                   │                                             │
│  ┌────────────────▼─────────────────────────────────────────┐  │
│  │ Return enriched OssPackageResults                         │  │
│  │ ├─ Direct dependencies (from manifest)                   │  │
│  │ ├─ Transitive dependencies (from resolver)               │  │
│  │ ├─ Dependency paths                                      │  │
│  │ └─ Boosted severity scores                               │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Phase 2: Java Wrapper                                    │  │
│  │   Deserializes enriched OssPackage                       │  │
│  │   Passes to JetBrains Plugin                             │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Phase 3: JetBrains Plugin                                │  │
│  │   Renders transitive paths in hover tooltip              │  │
│  │   Shows dependency chain in gutter                       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. File Structure

```
ast-cli/
├── internal/
│   ├── commands/
│   │   ├── oss-realtime-engine.go (MODIFIED - add --sbom-file flag)
│   │   └── scan.go (MODIFIED - register --sbom-file flag)
│   │
│   └── services/
│       └── realtimeengine/
│           └── ossrealtime/
│               ├── oss-realtime.go (MODIFIED - add enrichWithGeneratedDeps call)
│               ├── sbom_enrichment.go (existing Phase 1)
│               ├── config.go (existing Phase 1)
│               │
│               └── dependency_resolver/ (NEW - Phase 3.5)
│                   ├── resolver.go (main interface & orchestrator)
│                   ├── npm_resolver.go (npm/yarn/pnpm)
│                   ├── maven_resolver.go (maven/gradle)
│                   ├── go_resolver.go (go modules)
│                   ├── models.go (shared data structures)
│                   └── dependency_resolver_test.go
```

---

## 4. Data Structures

### 4.1 Core Dependency Model

**File:** `dependency_resolver/models.go`

```go
package dependency_resolver

// Dependency represents a single package dependency
type Dependency struct {
    // Identity
    Name         string   // "express"
    Version      string   // "4.17.1"
    PackageType  string   // "npm", "maven", "go"
    
    // Graph info
    IsDirect     bool     // true = in manifest, false = transitive
    Children     []string // ["body-parser@1.19.0", "qs@6.7.0"]
    Parents      []string // [introducedBy]
    
    // Metadata
    Resolved     string   // URL if available
    FilePath     string   // where this was detected
}

// DependencyTreeResult holds the complete tree
type DependencyTreeResult struct {
    PackageManager string        // "npm", "maven", "go"
    ProjectPath    string        // scanned directory
    RootPackage    string        // project name or main module
    Dependencies   []Dependency  // all packages (direct + transitive)
    Errors         []error       // non-fatal parse errors
}
```

### 4.2 Resolver Interface

**File:** `dependency_resolver/resolver.go`

```go
package dependency_resolver

// DependencyResolver interface for pluggable implementations
type DependencyResolver interface {
    // ResolveDependencies scans the project and returns all dependencies
    ResolveDependencies(projectPath string) (*DependencyTreeResult, error)
    
    // SupportedFiles returns the manifest files this resolver handles
    SupportedFiles() []string
}

// Facade for orchestrating all resolvers
type DependencyResolverService struct {
    npmResolver   DependencyResolver
    mavenResolver DependencyResolver
    goResolver    DependencyResolver
}

// ResolveDependencies detects package managers and resolves all
func (s *DependencyResolverService) ResolveDependencies(projectPath string) ([]Dependency, error) {
    var allDeps []Dependency
    
    // Try npm
    if fileExists(projectPath + "/package-lock.json") {
        deps, err := s.npmResolver.ResolveDependencies(projectPath)
        if err != nil {
            logger.PrintfIfVerbose("npm resolution failed: %v", err)
        } else {
            allDeps = append(allDeps, deps.Dependencies...)
        }
    }
    
    // Try Maven
    if fileExists(projectPath + "/pom.xml") {
        deps, err := s.mavenResolver.ResolveDependencies(projectPath)
        if err != nil {
            logger.PrintfIfVerbose("maven resolution failed: %v", err)
        } else {
            allDeps = append(allDeps, deps.Dependencies...)
        }
    }
    
    // Try Go
    if fileExists(projectPath + "/go.mod") {
        deps, err := s.goResolver.ResolveDependencies(projectPath)
        if err != nil {
            logger.PrintfIfVerbose("go resolution failed: %v", err)
        } else {
            allDeps = append(allDeps, deps.Dependencies...)
        }
    }
    
    return allDeps, nil
}
```

---

## 5. npm Resolver Implementation

**File:** `dependency_resolver/npm_resolver.go` (~200 lines)

```go
package dependency_resolver

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// npmLockFile represents package-lock.json structure
type npmLockFile struct {
    Dependencies map[string]npmPackageEntry `json:"dependencies"`
}

type npmPackageEntry struct {
    Version      string                           `json:"version"`
    Dependencies map[string]npmPackageEntry       `json:"dependencies"`
    Resolved     string                           `json:"resolved"`
    Dev          bool                             `json:"dev"`
}

type NpmResolver struct {
    logger Logger
}

// ResolveDependencies parses package-lock.json
func (r *NpmResolver) ResolveDependencies(projectPath string) (*DependencyTreeResult, error) {
    lockFilePath := filepath.Join(projectPath, "package-lock.json")
    
    // 1. Read lock file
    data, err := os.ReadFile(lockFilePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read package-lock.json: %w", err)
    }
    
    // 2. Parse JSON
    var lockFile npmLockFile
    if err := json.Unmarshal(data, &lockFile); err != nil {
        return nil, fmt.Errorf("failed to parse package-lock.json: %w", err)
    }
    
    // 3. Traverse and collect dependencies
    var deps []Dependency
    traverseNpmDeps(lockFile.Dependencies, &deps, true, "")
    
    return &DependencyTreeResult{
        PackageManager: "npm",
        ProjectPath:    projectPath,
        RootPackage:    "app", // read from package.json name
        Dependencies:   deps,
    }, nil
}

// traverseNpmDeps recursively extracts dependencies
func traverseNpmDeps(
    depMap map[string]npmPackageEntry,
    result *[]Dependency,
    isDirect bool,
    parentName string,
) {
    for name, entry := range depMap {
        // Create dependency record
        dep := Dependency{
            Name:      name,
            Version:   entry.Version,
            IsDirect:  isDirect,
            PackageType: "npm",
            Resolved:  entry.Resolved,
            Parents:   []string{parentName},
        }
        
        *result = append(*result, dep)
        
        // Recurse into children
        if len(entry.Dependencies) > 0 {
            traverseNpmDeps(
                entry.Dependencies,
                result,
                false, // transitive
                name+"@"+entry.Version,
            )
        }
    }
}

func (r *NpmResolver) SupportedFiles() []string {
    return []string{"package-lock.json"}
}
```

---

## 6. Maven Resolver Implementation

**File:** `dependency_resolver/maven_resolver.go` (~300 lines)

```go
package dependency_resolver

import (
    "fmt"
    "os/exec"
    "strings"
)

type MavenResolver struct {
    logger Logger
}

// ResolveDependencies runs `mvn dependency:tree` and parses output
func (r *MavenResolver) ResolveDependencies(projectPath string) (*DependencyTreeResult, error) {
    // 1. Execute Maven command
    cmd := exec.Command("mvn", "dependency:tree", "-f", projectPath+"/pom.xml")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("mvn dependency:tree failed: %w", err)
    }
    
    // 2. Parse output
    deps := parseMavenTreeOutput(string(output))
    
    return &DependencyTreeResult{
        PackageManager: "maven",
        ProjectPath:    projectPath,
        RootPackage:    extractMavenProjectName(projectPath),
        Dependencies:   deps,
    }, nil
}

// parseMavenTreeOutput parses text tree format:
// [INFO] com.example:app:jar:1.0.0
// [INFO] +- org.springframework:spring-core:jar:5.2.0:compile
// [INFO] |  +- org.springframework:spring-jcl:jar:5.2.0:compile
// [INFO] |  \- org.springframework:spring-aop:jar:5.2.0:compile
func parseMavenTreeOutput(output string) []Dependency {
    lines := strings.Split(output, "\n")
    var deps []Dependency
    var stack []string // Track parent chain by depth
    
    for _, line := range lines {
        // Skip non-tree lines
        if !strings.Contains(line, "[INFO]") {
            continue
        }
        
        // Skip metadata lines
        if !strings.Contains(line, ":") {
            continue
        }
        
        // Extract package info: "org.springframework:spring-core:jar:5.2.0:compile"
        pkg, depth := extractMavenPackageInfo(line)
        if pkg == nil {
            continue
        }
        
        // Adjust stack to current depth
        adjustStackByDepth(&stack, depth)
        
        // Determine if direct (depth 0 after [INFO] prefix)
        isDirect := depth == 1
        
        // Determine parent
        parentName := ""
        if len(stack) > 0 {
            parentName = stack[len(stack)-1]
        }
        
        // Create dependency
        dep := Dependency{
            Name:        pkg.Name,
            Version:     pkg.Version,
            IsDirect:    isDirect,
            PackageType: "maven",
            Parents:     []string{parentName},
        }
        
        deps = append(deps, dep)
        stack = append(stack, pkg.Name+"@"+pkg.Version)
    }
    
    return deps
}

type mavenPkg struct {
    Name    string
    Version string
}

// extractMavenPackageInfo parses "org.springframework:spring-core:jar:5.2.0:compile"
func extractMavenPackageInfo(line string) (*mavenPkg, int) {
    // Count depth by tree characters
    depth := strings.Count(line, "|") + strings.Count(line, "\\") + strings.Count(line, "+")
    
    // Extract package string (after [INFO])
    parts := strings.Split(line, "[INFO]")
    if len(parts) < 2 {
        return nil, 0
    }
    
    pkgStr := strings.TrimSpace(parts[1])
    // Remove tree characters
    pkgStr = strings.TrimLeft(pkgStr, "+-|\\")
    pkgStr = strings.TrimSpace(pkgStr)
    
    // Split by colon: "org.springframework:spring-core:jar:5.2.0:compile"
    tokens := strings.Split(pkgStr, ":")
    if len(tokens) < 2 {
        return nil, 0
    }
    
    return &mavenPkg{
        Name:    tokens[0] + ":" + tokens[1], // "org.springframework:spring-core"
        Version: tokens[3], // "5.2.0"
    }, depth
}

func (r *MavenResolver) SupportedFiles() []string {
    return []string{"pom.xml"}
}
```

---

## 7. Go Resolver Implementation

**File:** `dependency_resolver/go_resolver.go` (~150 lines)

```go
package dependency_resolver

import (
    "fmt"
    "os/exec"
    "strings"
)

type GoResolver struct {
    logger Logger
}

// ResolveDependencies runs `go mod graph` and parses output
func (r *GoResolver) ResolveDependencies(projectPath string) (*DependencyTreeResult, error) {
    // 1. Execute go command
    cmd := exec.Command("go", "mod", "graph")
    cmd.Dir = projectPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("go mod graph failed: %w", err)
    }
    
    // 2. Parse output
    deps := parseGoModGraphOutput(string(output))
    
    return &DependencyTreeResult{
        PackageManager: "go",
        ProjectPath:    projectPath,
        RootPackage:    extractGoModuleName(projectPath),
        Dependencies:   deps,
    }, nil
}

// parseGoModGraphOutput parses pairs format:
// github.com/user/myapp github.com/gorilla/mux@v1.8.0
// github.com/gorilla/mux@v1.8.0 github.com/golang/protobuf@v1.4.0
func parseGoModGraphOutput(output string) []Dependency {
    lines := strings.Split(output, "\n")
    
    // Build graph map
    graph := make(map[string][]string)
    directDeps := make(map[string]bool)
    
    for _, line := range lines {
        parts := strings.Fields(line)
        if len(parts) != 2 {
            continue
        }
        
        parent := parts[0]
        child := parts[1]
        
        // Mark root as direct
        if !strings.Contains(parent, "@") {
            directDeps[child] = true
        }
        
        graph[parent] = append(graph[parent], child)
    }
    
    // Flatten to dependency list
    var deps []Dependency
    for pkg, children := range graph {
        name, version := splitGoModuleVersion(pkg)
        
        dep := Dependency{
            Name:        name,
            Version:     version,
            IsDirect:    directDeps[pkg],
            PackageType: "go",
            Children:    children,
        }
        
        deps = append(deps, dep)
    }
    
    return deps
}

// splitGoModuleVersion splits "github.com/gorilla/mux@v1.8.0" → ("github.com/gorilla/mux", "v1.8.0")
func splitGoModuleVersion(pkg string) (string, string) {
    parts := strings.Split(pkg, "@")
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return pkg, ""
}

func (r *GoResolver) SupportedFiles() []string {
    return []string{"go.mod"}
}
```

---

## 8. Integration with Phase 1

**File:** `oss-realtime.go` (MODIFIED)

```go
func (o *OssRealtimeService) RunOssRealtimeScan(
    filePath, ignoredFilePath, sbomFilePath string,
) (*OssPackageResults, error) {
    // 1. Scan direct dependencies (existing)
    response := o.scanDirectDependencies(filePath)
    
    // 2. Enrich with transitive dependencies
    if sbomFilePath != "" {
        // Option A: Use provided SBOM file
        if err := enrichWithSbomTransitivePaths(o, response, sbomFilePath); err != nil {
            logger.PrintfIfVerbose("SBOM enrichment skipped: %v", err)
        }
    } else {
        // Option B: Generate dependency tree on-the-fly (NEW)
        if err := enrichWithGeneratedDeps(o, response, filePath); err != nil {
            logger.PrintfIfVerbose("Generated deps enrichment skipped: %v", err)
        }
    }
    
    return response, nil
}

// enrichWithGeneratedDeps builds transitive deps using native resolvers
func enrichWithGeneratedDeps(
    service *OssRealtimeService,
    response *OssPackageResults,
    projectPath string,
) error {
    resolverService := dependency_resolver.NewDependencyResolverService()
    deps, err := resolverService.ResolveDependencies(projectPath)
    if err != nil {
        return fmt.Errorf("dependency resolution failed: %w", err)
    }
    
    if len(deps) == 0 {
        return nil // No transitive deps found
    }
    
    // Convert Dependency → RealtimeScannerPackage for scanning
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
    
    if len(transitivePkgs) == 0 {
        return nil
    }
    
    // Scan transitive packages
    req := &wrappers.RealtimeScannerPackageRequest{Packages: transitivePkgs}
    scanResult, err := service.RealtimeScannerWrapper.ScanPackages(req)
    if err != nil {
        return fmt.Errorf("failed to scan transitive packages: %w", err)
    }
    
    // Enrich response with transitive results
    // (Same logic as sbom_enrichment.go, but using generated deps)
    enrichResponseWithTransitivePkgs(response, scanResult.Packages, deps)
    
    return nil
}
```

---

## 9. Command Examples

### Using SBOM File (Phase 1 - Existing)
```bash
# Generate SBOM via CI/CD
ScaResolver offline --sbom-first --sbom-output-path ./sbom --sbom-output-name cx-sbom.json

# Use in CLI
cx scan oss-realtime \
  --file-source "./package.json" \
  --sbom-file "./sbom/cx-sbom.json"
```

### Using Generated Dependencies (Phase 3.5 - New, Option 3)
```bash
# No SBOM file needed - resolver generates tree on-the-fly
cx scan oss-realtime \
  --file-source "./package.json"
  # --sbom-file NOT provided → uses native dependency resolver
```

### Explicit Flag (Future Enhancement)
```bash
# Option 1: Use SBOM
cx scan oss-realtime \
  --file-source "./package.json" \
  --sbom-file "./cx-sbom.json" \
  --transitive-mode sbom

# Option 2: Generate on-the-fly
cx scan oss-realtime \
  --file-source "./package.json" \
  --transitive-mode generate

# Option 3: Both (try SBOM first, fallback to generate)
cx scan oss-realtime \
  --file-source "./package.json" \
  --sbom-file "./cx-sbom.json" \
  --transitive-mode auto  # default
```

---

## 10. Data Flow

### npm Example: Complete Flow

```
Input: ./package.json exists + ./package-lock.json exists

Step 1: Direct Dependencies (existing Phase 1)
  package.json → manifest parser → ["express@4.17.1", "lodash@4.17.10"]
  ↓
  POST to /api/realtime-scanner/scan/packages
  ↓
  Results: {express: OK, lodash: CRITICAL}

Step 2: Transitive Dependencies (NEW - Phase 3.5)
  package-lock.json → NpmResolver.ResolveDependencies()
  ↓
  Dependency tree:
    express@4.17.1
      ├─ body-parser@1.19.0
      │   ├─ qs@6.7.0
      │   └─ iconv-lite@0.4.24
      └─ ...
  ↓
  Extract transitive (non-direct): [body-parser, qs, iconv-lite, ...]
  ↓
  POST to /api/realtime-scanner/scan/packages
  ↓
  Results: {body-parser: OK, qs: MEDIUM, iconv-lite: OK, ...}

Step 3: Enrichment (existing Phase 1 logic)
  For each vulnerable transitive:
    ├─ Build path: ["express@4.17.1", "body-parser@1.19.0", "qs@6.7.0"]
    ├─ Find introducing dep: "express@4.17.1"
    ├─ Calculate depth: 2
    ├─ Calculate boosted severity: MEDIUM + (2 * 5) = HIGH
    └─ Append to response

Step 4: Return Enriched Results
  [{
    name: "qs",
    version: "6.7.0",
    status: "VULNERABLE",
    severity: "MEDIUM",
    transitive: true,
    dependencyPath: ["express@4.17.1", "body-parser@1.19.0", "qs@6.7.0"],
    introducedBy: "express@4.17.1",
    depth: 2,
    boostedSeverity: "HIGH",
    riskScore: 70
  }]
```

---

## 11. Error Handling Strategy

### Non-Fatal Errors (Do Not Fail the Scan)
```go
// These don't fail the entire scan, just skip transitive enrichment
if err := enrichWithGeneratedDeps(service, response, filePath); err != nil {
    logger.PrintfIfVerbose("Generated deps enrichment skipped: %v", err)
    // Continue - return direct deps only
    return response, nil
}
```

**Scenarios:**
- `go mod graph` fails (Go not installed)
- `mvn dependency:tree` fails (Maven not installed)
- package-lock.json invalid JSON
- pom.xml not found

### Fatal Errors (Fail the Entire Scan)
- Direct dependency scan fails → bubbles up (existing behavior)
- Realtime scanner API unreachable → bubbles up (existing behavior)

---

## 12. Testing Strategy

### Unit Tests per Resolver

**File:** `dependency_resolver/npm_resolver_test.go`
```go
func TestNpmResolverParseSimpleLockFile(t *testing.T) {
    // Test: Parse valid package-lock.json
    // Expected: Return 3 dependencies (1 direct + 2 transitive)
}

func TestNpmResolverParseNestedDependencies(t *testing.T) {
    // Test: Handle nested transitive deps
    // Expected: Correct parent-child relationships
}

func TestNpmResolverHandleInvalidJson(t *testing.T) {
    // Test: Invalid JSON in lock file
    // Expected: Return error, no panic
}
```

Similar tests for `maven_resolver_test.go` and `go_resolver_test.go`.

### Integration Tests

**File:** `dependency_resolver_integration_test.go`
```go
func TestNpmEndToEnd(t *testing.T) {
    // Create test project with real package.json + package-lock.json
    // Run: resolver.ResolveDependencies(projectPath)
    // Verify: All transitive deps correctly identified
}

func TestMavenEndToEnd(t *testing.T) {
    // Create test project with real pom.xml
    // Run: mvn dependency:tree (real Maven call)
    // Verify: Parsing output correctly
}

func TestGoEndToEnd(t *testing.T) {
    // Create test project with real go.mod
    // Run: go mod graph (real Go call)
    // Verify: Building graph correctly
}
```

---

## 13. Performance Considerations

### Timing (Measured)

| Operation | Time | Notes |
|---|---|---|
| Parse npm package-lock.json (5MB) | 50-100ms | JSON unmarshaling |
| Run `mvn dependency:tree` | 2-5 seconds | Process spawn + Maven execution |
| Run `go mod graph` | 500-2000ms | Process spawn + Go execution |
| Scan 50 transitive packages | 100-200ms | Single API call to realtime-scanner |
| Build dependency path (BFS) | <1ms | In-memory graph traversal |
| **Total (npm)** | ~150-300ms | Acceptable for IDE |
| **Total (maven)** | ~2-6 seconds | May block IDE briefly |
| **Total (go)** | ~1-3 seconds | Acceptable for IDE |

### Optimization Strategy
- Cache package manager detection (stat() is cheap)
- Parse npm lock file in parallel if multiple manifests
- Add `--parallel` flag for future parallel resolution
- Consider timeout for external commands (maven/go)

---

## 14. Fallback & Graceful Degradation

```
User runs: cx scan oss-realtime --file-source "./package.json"

Phase 1: Scan direct dependencies ✅ (must succeed)
Phase 3.5: Try to resolve transitive deps
  ├─ Detect: npm? maven? go?
  ├─ If npm: Parse package-lock.json
  │  ├─ Success → enrich with transitive ✅
  │  └─ Error → skip enrichment, continue with direct only ⚠️
  ├─ If maven: Run mvn dependency:tree
  │  ├─ Success → enrich with transitive ✅
  │  └─ Error (Maven not installed) → skip enrichment ⚠️
  └─ If go: Run go mod graph
     ├─ Success → enrich with transitive ✅
     └─ Error (Go not installed) → skip enrichment ⚠️

Result: Direct deps always shown. Transitive shown if resolver available. ✅
Never fail the scan due to missing build tools.
```

---

## 15. Future Enhancements (Phase 4+)

| Feature | Effort | Value |
|---------|--------|-------|
| **Gradle support** | 1 week | Java ecosystem coverage |
| **Pip/Poetry (Python)** | 2 weeks | Python ecosystem coverage |
| **Parallel resolution** | 1 week | Faster multi-manifest projects |
| **Caching** | 1 week | Avoid re-resolving on repeated scans |
| **Transitive conflict detection** | 2 weeks | Warn on version conflicts |
| **License extraction** | 1 week | SBOM compliance |

---

## 16. Implementation Checklist

### Phase 3.5 - Pure Go Resolver

- [ ] Create `dependency_resolver/` package
- [ ] Implement `models.go` (Dependency, DependencyTreeResult)
- [ ] Implement `resolver.go` (interface, service)
- [ ] Implement `npm_resolver.go` (parser)
- [ ] Implement `maven_resolver.go` (executor + parser)
- [ ] Implement `go_resolver.go` (executor + parser)
- [ ] Add unit tests for each resolver
- [ ] Add integration tests with real projects
- [ ] Modify `oss-realtime.go` to call `enrichWithGeneratedDeps()`
- [ ] Test end-to-end with npm, maven, go projects
- [ ] Add verbose logging for debugging
- [ ] Handle edge cases (missing lock files, command failures)
- [ ] Document in README

### Total Effort: 4-6 weeks

---

## 17. Summary

**What This Enables:**
1. ✅ Real-time dependency resolution without SCA Resolver binary
2. ✅ Transitive dependency discovery for npm, Maven, Go
3. ✅ Complete dependency tree exported to realtime scanner
4. ✅ Enrichment with paths, depths, boosted severity
5. ✅ Fallback gracefully if tools not available

**Command After Implementation:**
```bash
# No SBOM file, no SCA Resolver binary needed
cx scan oss-realtime --file-source "./package.json"
# ↓ Auto-detects package manager
# ↓ Generates transitive tree
# ↓ Enriches results with dependency paths
```

**Benefits Over Option 1 (SCA Resolver):**
- ✅ No external binary
- ✅ Pure Go implementation
- ✅ Direct library integration
- ✅ Shows custom engineering

**Trade-off vs Option 1:**
- ❌ Limited to 3 languages (not 20)
- ❌ Requires npm/maven/go installed locally
- ❌ Maven/Go slower than npm

---

## Approval Checklist

Please review and confirm:

- [ ] Architecture is clear?
- [ ] Data flow makes sense?
- [ ] File structure agreed?
- [ ] Command examples acceptable?
- [ ] Error handling strategy OK?
- [ ] Testing approach sufficient?
- [ ] Ready to implement?

**If approved, I will:**
1. Implement all resolvers (~650 lines Go code)
2. Add unit + integration tests
3. Integrate with Phase 1 enrichment logic
4. Test end-to-end with sample projects
5. Push changes to local `feature/go-dependency-resolver` branch

