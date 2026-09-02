package dependency_resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MavenResolver implements DependencyResolver for maven/gradle
type MavenResolver struct{}

// ResolveDependencies runs `mvn dependency:tree` and parses output
func (r *MavenResolver) ResolveDependencies(projectPath string) (*DependencyTreeResult, error) {
	// 1. Execute Maven command
	cmd := exec.Command("mvn", "dependency:tree", "-f", filepath.Join(projectPath, "pom.xml"))
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mvn dependency:tree failed: %w", err)
	}

	// 2. Parse output
	deps := parseMavenTreeOutput(string(output))

	// 3. Extract root package name
	rootName := extractMavenProjectName(projectPath)

	return &DependencyTreeResult{
		PackageManager: "mvn",
		ProjectPath:    projectPath,
		RootPackage:    rootName,
		Dependencies:   deps,
	}, nil
}

// parseMavenTreeOutput parses text tree format from mvn dependency:tree
// Example format:
// [INFO] com.example:app:jar:1.0.0
// [INFO] +- org.springframework:spring-core:jar:5.2.0:compile
// [INFO] |  +- org.springframework:spring-jcl:jar:5.2.0:compile
// [INFO] |  \- org.springframework:spring-aop:jar:5.2.0:compile
func parseMavenTreeOutput(output string) []Dependency {
	lines := strings.Split(output, "\n")
	var deps []Dependency
	var stack []string                   // Track parent chain by depth
	visitedDeps := make(map[string]bool) // Deduplicate same package@version

	for _, line := range lines {
		// Skip non-tree lines
		if !strings.Contains(line, "[INFO]") {
			continue
		}

		// Skip metadata lines - must contain actual dependency tree patterns like "+-", "\-"
		// This filters out lines that only have + in timestamps
		if !strings.Contains(line, "+-") && !strings.Contains(line, "\\-") && !strings.Contains(line, "|") {
			continue
		}

		// Extract package info
		pkg, depth := extractMavenPackageInfo(line)
		if pkg == nil {
			continue
		}

		// Skip packages with empty names or versions
		if pkg.Name == "" || pkg.Version == "" {
			continue
		}

		// Adjust stack to current depth
		adjustStackByDepth(&stack, depth)

		// Determine if direct (depth 2 is direct - first level with tree chars, others are transitive)
		isDirect := depth == 2

		// Deduplicate by name@version to avoid sending same package multiple times
		depKey := pkg.Name + "@" + pkg.Version
		if visitedDeps[depKey] {
			continue
		}
		visitedDeps[depKey] = true

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
			PackageType: "mvn",
			Parents:     []string{parentName},
		}

		deps = append(deps, dep)
		stack = append(stack, pkg.Name+":"+pkg.Version)
	}

	// Post-processing: link each dependency into its parent's Children list, using
	// an index instead of an O(n) scan per parent (name:version -> position in deps).
	byNameVersion := make(map[string]int, len(deps))
	for i, dep := range deps {
		byNameVersion[dep.Name+":"+dep.Version] = i
	}
	for i := range deps {
		for _, parentKey := range deps[i].Parents {
			if j, found := byNameVersion[parentKey]; found {
				deps[j].Children = append(deps[j].Children, deps[i].Name+"@"+deps[i].Version)
			}
		}
	}

	return deps
}

// mavenPkg represents parsed maven package info
type mavenPkg struct {
	Name    string
	Version string
}

// extractMavenPackageInfo parses "org.springframework:spring-core:jar:5.2.0:compile"
func extractMavenPackageInfo(line string) (*mavenPkg, int) {
	// Count depth by tree characters
	depth := strings.Count(line, "|") + strings.Count(line, "\\") + strings.Count(line, "+")
	// Each level adds 1 to depth, so real depth is count + 1
	depth = depth + 1

	// Extract package string (after [INFO])
	parts := strings.Split(line, "[INFO]")
	if len(parts) < 2 {
		return nil, 0
	}

	pkgStr := strings.TrimSpace(parts[1])
	// Remove tree characters and whitespace (space must be included in the set)
	pkgStr = strings.TrimLeft(pkgStr, " +-|\\")
	pkgStr = strings.TrimSpace(pkgStr)

	// Split by colon: "org.springframework:spring-core:jar:5.2.0:compile"
	tokens := strings.Split(pkgStr, ":")
	if len(tokens) < 4 {
		return nil, 0
	}

	// Skip test scope dependencies (scope is last token: tokens[4])
	if len(tokens) > 4 && tokens[4] == "test" {
		return nil, 0
	}

	return &mavenPkg{
		Name:    tokens[0] + ":" + tokens[1], // "org.springframework:spring-core"
		Version: tokens[3],                   // "5.2.0"
	}, depth
}

// adjustStackByDepth adjusts the parent stack based on current depth
func adjustStackByDepth(stack *[]string, currentDepth int) {
	for len(*stack) >= currentDepth {
		*stack = (*stack)[:len(*stack)-1]
	}
}

// extractMavenProjectName reads pom.xml to get project name
func extractMavenProjectName(projectPath string) string {
	pomPath := filepath.Join(projectPath, "pom.xml")

	data, err := os.ReadFile(pomPath)
	if err != nil {
		return "app" // fallback
	}

	// Simple extraction of artifact ID from pom.xml
	content := string(data)
	idx := strings.Index(content, "<artifactId>")
	if idx == -1 {
		return "app"
	}

	content = content[idx+12:]
	endIdx := strings.Index(content, "</artifactId>")
	if endIdx == -1 {
		return "app"
	}

	return strings.TrimSpace(content[:endIdx])
}

// SupportedFiles returns manifest files this resolver handles
func (r *MavenResolver) SupportedFiles() []string {
	return []string{"pom.xml"}
}
