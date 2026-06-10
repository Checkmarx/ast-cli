// Package sca implements Open Source Software (OSS) realtime guardrails for
// AI coding agents: install-command interception and manifest-edit gating.
package sca

import (
	"path/filepath"
	"strings"
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
)

// IsManifest reports whether path names a manifest file the OSS realtime
// scanner can analyse. The rules mirror manifest-parser's selectManifestFile:
//
//   - *.csproj                        → Dotnet csproj
//   - requirements*.txt, packages*.txt → Pypi requirements
//   - pom.xml                          → Maven
//   - package.json                     → Npm
//   - Directory.Packages.props         → Dotnet central package management
//   - packages.config                  → Dotnet legacy
//   - go.mod                           → Go modules
func IsManifest(path string) (Format, bool) {
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	switch {
	case ext == ".csproj":
		return FormatDotnetCsproj, true
	case ext == ".txt" && (strings.HasPrefix(base, "requirement") || strings.HasPrefix(base, "packages")):
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
	}
	return ""
}
