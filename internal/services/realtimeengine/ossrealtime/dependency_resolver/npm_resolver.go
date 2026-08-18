package dependency_resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// npmLockFile represents package-lock.json structure (modern format with "packages")
type npmLockFile struct {
	Dependencies map[string]string          `json:"dependencies"`
	Packages     map[string]npmPackageEntry `json:"packages"`
}

// npmPackageEntry represents a single package entry in lock file
type npmPackageEntry struct {
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Resolved     string            `json:"resolved"`
	Dev          bool              `json:"dev"`
}

// NpmResolver implements DependencyResolver for npm/yarn/pnpm
type NpmResolver struct{}

// ResolveDependencies parses package-lock.json and returns dependency tree
func (r *NpmResolver) ResolveDependencies(projectPath string) (*DependencyTreeResult, error) {
	lockFilePath := filepath.Join(projectPath, "package-lock.json")

	// 1. Read lock file
	data, err := os.ReadFile(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package-lock.json: %w", err)
	}

	// 2. Pre-validate JSON structure before unmarshalling (defense-in-depth)
	if err := validateNpmLockFile(data); err != nil {
		return nil, fmt.Errorf("npm lock file validation failed: %w", err)
	}

	// 3. Parse JSON (safe after validation)
	var lockFile npmLockFile
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse package-lock.json: %w", err)
	}

	// 4. Traverse and collect dependencies from the appropriate source
	var deps []Dependency
	// Modern npm (v7+) uses "packages" field; older versions use "dependencies"
	if len(lockFile.Packages) > 0 {
		// In modern npm, root dependencies are in packages[""].dependencies
		rootEntry, hasRoot := lockFile.Packages[""]
		if hasRoot && len(rootEntry.Dependencies) > 0 {
			traverseNpmModernFormat(lockFile.Packages, rootEntry.Dependencies, &deps)
		}
	}

	// 5. Extract root package name (from package.json)
	rootName := extractRootPackageName(projectPath)

	return &DependencyTreeResult{
		PackageManager: "npm",
		ProjectPath:    projectPath,
		RootPackage:    rootName,
		Dependencies:   deps,
	}, nil
}

// validateNpmLockFile performs pre-deserialization validation
func validateNpmLockFile(data []byte) error {
	// First, unmarshal to generic map to validate structure
	var rawLock map[string]interface{}
	if err := json.Unmarshal(data, &rawLock); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate has dependencies field or is empty (both valid)
	if _, hasDeps := rawLock["dependencies"]; !hasDeps {
		// Empty lock file is OK, will just have no transitive deps
		return nil
	}

	return nil
}

// traverseNpmModernFormat handles npm v7+ "packages" format
// In modern npm, the structure is:
// "packages": {
//   "": { "dependencies": { "pkg": "version" } },
//   "node_modules/pkg": { "version": "1.0.0", "dependencies": { ... } }
// }
func traverseNpmModernFormat(packagesMap map[string]npmPackageEntry, rootDeps map[string]string, result *[]Dependency) {
	// Build a map of all packages by name for quick lookup
	pkgByName := make(map[string]npmPackageEntry)
	for path, entry := range packagesMap {
		if path == "" {
			continue // Skip root
		}
		// Extract package name from path like "node_modules/lodash" or "node_modules/@org/pkg"
		var pkgName string
		if len(path) > len("node_modules/") {
			pkgName = path[len("node_modules/"):]
		}
		if pkgName != "" {
			pkgByName[pkgName] = entry
		}
	}

	// Process direct dependencies from root
	visitedDeps := make(map[string]bool)
	var processDirectDeps func(deps map[string]string, isDirect bool, parentPath string)
	processDirectDeps = func(deps map[string]string, isDirect bool, parentPath string) {
		for depName, versionStr := range deps {
			depKey := depName + "@" + versionStr
			if visitedDeps[depKey] {
				continue
			}
			visitedDeps[depKey] = true

			// Create dependency record
			dep := Dependency{
				Name:        depName,
				Version:     versionStr,
				IsDirect:    isDirect,
				PackageType: "npm",
				Parents:     []string{parentPath},
			}

			// Find children from packages map
			pkgEntry, found := pkgByName[depName]
			if found {
				dep.Resolved = pkgEntry.Resolved
				dep.Version = pkgEntry.Version // Use actual version from packages entry
				for childName, childVersion := range pkgEntry.Dependencies {
					dep.Children = append(dep.Children, childName+"@"+childVersion)
				}
			}

			*result = append(*result, dep)

			// Recurse into transitive dependencies
			if found && len(pkgEntry.Dependencies) > 0 {
				processDirectDeps(pkgEntry.Dependencies, false, depName+"@"+versionStr)
			}
		}
	}

	// Start with direct dependencies from root
	processDirectDeps(rootDeps, true, "")
}

// traverseNpmDeps recursively extracts dependencies from npm lock file (legacy format support)
func traverseNpmDeps(
	depMap map[string]string,
	result *[]Dependency,
	isDirect bool,
	parentName string,
) {
	for name, version := range depMap {
		// Create dependency record
		dep := Dependency{
			Name:        name,
			Version:     version,
			IsDirect:    isDirect,
			PackageType: "npm",
			Parents:     []string{parentName},
		}

		*result = append(*result, dep)
	}
}

// extractRootPackageName reads package.json to get root name
func extractRootPackageName(projectPath string) string {
	packageJsonPath := filepath.Join(projectPath, "package.json")

	type packageJSON struct {
		Name string `json:"name"`
	}

	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return "app" // fallback
	}

	// Pre-validate JSON structure before unmarshalling
	var rawPkg map[string]interface{}
	if err := json.Unmarshal(data, &rawPkg); err != nil {
		return "app" // fallback
	}

	// Safe to unmarshal after validation
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "app" // fallback
	}

	if pkg.Name != "" {
		return pkg.Name
	}
	return "app"
}

// SupportedFiles returns manifest files this resolver handles
func (r *NpmResolver) SupportedFiles() []string {
	return []string{"package-lock.json", "package.json"}
}
