package dependency_resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GoResolver implements DependencyResolver for go modules
type GoResolver struct{}

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

	// 3. Extract root package name
	rootName := extractGoModuleName(projectPath)

	return &DependencyTreeResult{
		PackageManager: "go",
		ProjectPath:    projectPath,
		RootPackage:    rootName,
		Dependencies:   deps,
	}, nil
}

// parseGoModGraphOutput parses pairs format from `go mod graph`
// Example format:
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

		// Mark root as direct (modules without @ are root)
		if !strings.Contains(parent, "@") {
			directDeps[child] = true
		}

		graph[parent] = append(graph[parent], child)
	}

	// Flatten to dependency list
	var deps []Dependency
	for pkg, children := range graph {
		name, version := splitGoModuleVersion(pkg)

		// Skip root module (has no @version) and packages with empty versions
		if version == "" {
			continue
		}

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

// extractGoModuleName reads go.mod to get module name
func extractGoModuleName(projectPath string) string {
	modPath := filepath.Join(projectPath, "go.mod")

	data, err := os.ReadFile(modPath)
	if err != nil {
		return "app" // fallback
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}

	return "app"
}

// SupportedFiles returns manifest files this resolver handles
func (r *GoResolver) SupportedFiles() []string {
	return []string{"go.mod", "go.sum"}
}
