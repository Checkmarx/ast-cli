package dependency_resolver

// Dependency represents a single package dependency
type Dependency struct {
	// Identity
	Name        string // "express"
	Version     string // "4.17.1"
	PackageType string // "npm", "maven", "go"

	// Graph info
	IsDirect bool     // true = in manifest, false = transitive
	Children []string // ["body-parser@1.19.0", "qs@6.7.0"]
	Parents  []string // [introducedBy]

	// Metadata
	Resolved string // URL if available
	FilePath string // where this was detected
}

// DependencyTreeResult holds the complete tree
type DependencyTreeResult struct {
	PackageManager string       // "npm", "maven", "go"
	ProjectPath    string       // scanned directory
	RootPackage    string       // project name or main module
	Dependencies   []Dependency // all packages (direct + transitive)
	Errors         []error      // non-fatal parse errors
}

// ResolutionResult provides diagnostics
type ResolutionResult struct {
	Success      bool // true if at least one resolver found deps
	Dependencies []Dependency
	Warning      string // "Lock file not found" etc
	Error        string // "resolution failed" etc
}
