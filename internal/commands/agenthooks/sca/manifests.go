// Package sca implements Open Source Software (OSS) realtime guardrails for
// AI coding agents: install-command interception and manifest-edit gating.
package sca

import (
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

// Format identifies one of the manifest shapes that the SCA guardrails care
// about. Each format lines up 1:1 with a format the Checkmarx manifest-parser
// library recognises.
type Format int

const (
	FormatUnknown Format = iota
	FormatNpmPackageJson
	FormatPypiRequirements
	FormatGoMod
	FormatMavenPom
	FormatDotnetCsproj
	FormatDotnetDirectoryPackagesProps
	FormatDotnetPackagesConfig
	FormatGradleBuild
	FormatGradleVersionCatalog
	FormatSbtBuild
)

// gradleBuildFileName and gradleVersionCatalogFileName are the canonical basenames for the Gradle
// manifest formats, shared between the classifier switch below and SynthFileName.
const (
	gradleBuildFileName          = "build.gradle"
	gradleVersionCatalogFileName = "libs.versions.toml"
)

// IsManifest reports whether path names a manifest file the OSS realtime
// scanner can analyse. ossrealtime.IsSupportedManifestFile is the single
// source of truth for which filenames the scanner accepts — it is also used
// to gate the actual scan, so hooks and the scanner can never drift apart on
// what counts as a supported manifest. The switch below only classifies an
// already-accepted path into the Format the guardrails need to re-synthesise
// a minimal manifest for the added packages (see Synthesize).
//
//   - *.csproj                                      → Dotnet csproj
//   - *.sbt                                          → Sbt build
//   - requirements*.txt, packages*.txt, constraint*.txt → Pypi requirements
//   - setup.cfg, setup.py, pyproject.toml            → Pypi (alt. formats)
//   - pom.xml                                        → Maven
//   - package.json                                   → Npm
//   - Directory.Packages.props                       → Dotnet central package management
//   - packages.config                                → Dotnet legacy
//   - go.mod                                          → Go modules
//   - build.gradle, build.gradle.kts                 → Gradle build
//   - libs.versions.toml                             → Gradle version catalog
func IsManifest(path string) (Format, bool) {
	if !ossrealtime.IsSupportedManifestFile(path) {
		return FormatUnknown, false
	}

	base := filepath.Base(path)
	ext := filepath.Ext(base)

	switch {
	case ext == ".csproj":
		return FormatDotnetCsproj, true
	case ext == ".sbt":
		return FormatSbtBuild, true
	case ext == ".txt" && (strings.HasPrefix(base, "requirement") || strings.HasPrefix(base, "packages") || strings.HasPrefix(base, "constraint")):
		return FormatPypiRequirements, true
	case base == "pom.xml":
		return FormatMavenPom, true
	case base == "package.json":
		return FormatNpmPackageJson, true
	case base == "Directory.Packages.props":
		return FormatDotnetDirectoryPackagesProps, true
	case base == "packages.config":
		return FormatDotnetPackagesConfig, true
	case base == "go.mod":
		return FormatGoMod, true
	case base == gradleBuildFileName, base == gradleBuildFileName+".kts":
		return FormatGradleBuild, true
	case base == gradleVersionCatalogFileName:
		return FormatGradleVersionCatalog, true
	case base == "setup.cfg", base == "setup.py", base == "pyproject.toml":
		return FormatPypiRequirements, true
	}
	return FormatUnknown, false
}

// ManagerName returns the package-manager string oss-realtime uses for the
// given format (matching the PackageManager field on OssPackage).
func (f Format) ManagerName() string {
	switch f {
	case FormatNpmPackageJson:
		return "npm"
	case FormatPypiRequirements:
		return "pypi"
	case FormatGoMod:
		return "go"
	case FormatMavenPom:
		return "maven"
	case FormatDotnetCsproj, FormatDotnetDirectoryPackagesProps, FormatDotnetPackagesConfig:
		return "nuget"
	case FormatGradleBuild, FormatGradleVersionCatalog:
		return "gradle"
	case FormatSbtBuild:
		return "sbt"
	}
	return ""
}

// SynthFileName returns a filename to use when writing a temp manifest in the
// given format. The name matters because manifest-parser's factory selects a
// parser by basename/extension.
func (f Format) SynthFileName() string {
	switch f {
	case FormatNpmPackageJson:
		return "package.json"
	case FormatPypiRequirements:
		return "requirements.txt"
	case FormatGoMod:
		return "go.mod"
	case FormatMavenPom:
		return "pom.xml"
	case FormatDotnetCsproj:
		return "synth.csproj"
	case FormatDotnetDirectoryPackagesProps:
		return "Directory.Packages.props"
	case FormatDotnetPackagesConfig:
		return "packages.config"
	case FormatGradleBuild:
		return gradleBuildFileName
	case FormatGradleVersionCatalog:
		return gradleVersionCatalogFileName
	case FormatSbtBuild:
		return "synth.sbt"
	}
	return ""
}
