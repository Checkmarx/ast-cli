package sca

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Synthesize writes a minimal manifest of the given format containing pkgs
// into dir, and returns the absolute path. The synthesised file's basename
// matches what manifest-parser expects so its parser factory picks the right
// parser when the scanner reads the file back.
func Synthesize(format Format, pkgs []Package, dir string) (string, error) {
	name := format.SynthFileName()
	if name == "" {
		return "", fmt.Errorf("sca: unsupported synth format %d", format)
	}
	path := filepath.Join(dir, name)

	var content []byte
	var err error
	switch format {
	case FormatNpmPackageJson:
		content, err = synthNpm(pkgs)
	case FormatPypiRequirements:
		content = synthPypi(pkgs)
	case FormatGoMod:
		content = synthGoMod(pkgs)
	case FormatMavenPom:
		content = synthMavenPom(pkgs)
	case FormatDotnetCsproj:
		content = synthCsproj(pkgs)
	case FormatDotnetDirectoryPackagesProps:
		content = synthDirectoryPackagesProps(pkgs)
	case FormatDotnetPackagesConfig:
		content = synthPackagesConfig(pkgs)
	case FormatGradleBuild:
		content = synthGradleBuild(pkgs)
	case FormatGradleVersionCatalog:
		content = synthGradleVersionCatalog(pkgs)
	case FormatSbtBuild:
		content = synthSbt(pkgs)
	case FormatBower:
		content, err = synthBower(pkgs)
	case FormatComposerJson:
		content, err = synthComposerJson(pkgs)
	case FormatPubspecYaml:
		content = synthPubspecYaml(pkgs)
	case FormatGemfile:
		content = synthGemfile(pkgs)
	case FormatCocoaPodsPodfile:
		content = synthPodfile(pkgs)
	case FormatCocoaPodsPodspec:
		content = synthPodspec(pkgs)
	case FormatCarthage:
		content = synthCartfile(pkgs)
	case FormatSwiftPackageManager:
		content = synthSwiftPackage(pkgs)
	default:
		return "", fmt.Errorf("sca: unsupported synth format %d", format)
	}
	if err != nil {
		return "", err
	}
	if writeErr := os.WriteFile(path, content, 0600); writeErr != nil {
		return "", writeErr
	}
	return path, nil
}

func synthNpm(pkgs []Package) ([]byte, error) {
	deps := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "latest"
		}
		deps[p.Name] = v
	}
	manifest := map[string]any{
		"name":         "sca-scan-temp",
		"version":      "1.0.0",
		"dependencies": deps,
	}
	return json.MarshalIndent(manifest, "", "  ")
}

func synthPypi(pkgs []Package) []byte {
	var b strings.Builder
	for _, p := range pkgs {
		if p.Version != "" {
			fmt.Fprintf(&b, "%s==%s\n", p.Name, p.Version)
		} else {
			fmt.Fprintf(&b, "%s\n", p.Name)
		}
	}
	return []byte(b.String())
}

func synthGoMod(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("module sca-scan-temp\n\ngo 1.21\n\nrequire (\n")
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "latest"
		}
		fmt.Fprintf(&b, "\t%s %s\n", p.Name, v)
	}
	b.WriteString(")\n")
	return []byte(b.String())
}

func synthMavenPom(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>sca</groupId>
  <artifactId>sca-scan-temp</artifactId>
  <version>1.0.0</version>
  <dependencies>
`)
	for _, p := range pkgs {
		// Name is "groupId:artifactId".
		group, artifact := splitMavenName(p.Name)
		fmt.Fprintf(&b, "    <dependency>\n      <groupId>%s</groupId>\n      <artifactId>%s</artifactId>\n      <version>%s</version>\n    </dependency>\n", group, artifact, p.Version)
	}
	b.WriteString("  </dependencies>\n</project>\n")
	return []byte(b.String())
}

func splitMavenName(name string) (string, string) {
	idx := strings.Index(name, ":")
	if idx < 0 {
		return name, name
	}
	return name[:idx], name[idx+1:]
}

func synthCsproj(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("<Project Sdk=\"Microsoft.NET.Sdk\">\n  <ItemGroup>\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "    <PackageReference Include=\"%s\" Version=\"%s\" />\n", p.Name, p.Version)
	}
	b.WriteString("  </ItemGroup>\n</Project>\n")
	return []byte(b.String())
}

func synthDirectoryPackagesProps(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("<Project>\n  <ItemGroup>\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "    <PackageVersion Include=\"%s\" Version=\"%s\" />\n", p.Name, p.Version)
	}
	b.WriteString("  </ItemGroup>\n</Project>\n")
	return []byte(b.String())
}

func synthPackagesConfig(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<packages>\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "  <package id=\"%s\" version=\"%s\" />\n", p.Name, p.Version)
	}
	b.WriteString("</packages>\n")
	return []byte(b.String())
}

// synthGradleBuild writes one `implementation 'group:artifact:version'` line
// per package — the simplest form the Gradle parser's dependency regexes
// recognise. Name is "group:artifact" (mirrors Maven's groupId:artifactId).
func synthGradleBuild(pkgs []Package) []byte {
	var b strings.Builder
	for _, p := range pkgs {
		group, artifact := splitMavenName(p.Name)
		v := p.Version
		if v == "" {
			v = "latest"
		}
		fmt.Fprintf(&b, "implementation '%s:%s:%s'\n", group, artifact, v)
	}
	return []byte(b.String())
}

// synthGradleVersionCatalog writes a minimal libs.versions.toml [libraries]
// section — one inline "group:artifact:version" entry per package.
func synthGradleVersionCatalog(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("[libraries]\n")
	for i, p := range pkgs {
		group, artifact := splitMavenName(p.Name)
		v := p.Version
		if v == "" {
			v = "latest"
		}
		fmt.Fprintf(&b, "dep%d = \"%s:%s:%s\"\n", i, group, artifact, v)
	}
	return []byte(b.String())
}

// synthSbt writes one `libraryDependencies += "group" % "artifact" % "version"`
// line per package, the form the SBT parser's dependency regex recognises.
func synthSbt(pkgs []Package) []byte {
	var b strings.Builder
	for _, p := range pkgs {
		group, artifact := splitMavenName(p.Name)
		v := p.Version
		if v == "" {
			v = "latest"
		}
		fmt.Fprintf(&b, "libraryDependencies += \"%s\" %% \"%s\" %% \"%s\"\n", group, artifact, v)
	}
	return []byte(b.String())
}

func synthComposerJson(pkgs []Package) ([]byte, error) {
	require := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "latest"
		}
		require[p.Name] = v
	}
	manifest := map[string]any{
		"name":    "sca-scan-temp",
		"require": require,
	}
	return json.MarshalIndent(manifest, "", "  ")
}

func synthBower(pkgs []Package) ([]byte, error) {
	deps := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "latest"
		}
		deps[p.Name] = v
	}
	manifest := map[string]any{
		"name":         "sca-scan-temp",
		"dependencies": deps,
	}
	return json.MarshalIndent(manifest, "", "  ")
}

func synthPubspecYaml(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("name: sca_scan_temp\nenvironment:\n  sdk: '>=2.12.0 <4.0.0'\ndependencies:\n")
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "any"
		}
		fmt.Fprintf(&b, "  %s: %s\n", p.Name, v)
	}
	return []byte(b.String())
}

func synthGemfile(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("source 'https://rubygems.org'\n\n")
	for _, p := range pkgs {
		if p.Version != "" {
			fmt.Fprintf(&b, "gem '%s', '%s'\n", p.Name, p.Version)
		} else {
			fmt.Fprintf(&b, "gem '%s'\n", p.Name)
		}
	}
	return []byte(b.String())
}

func synthPodfile(pkgs []Package) []byte {
	var b strings.Builder
	for _, p := range pkgs {
		if p.Version != "" {
			fmt.Fprintf(&b, "pod '%s', '%s'\n", p.Name, p.Version)
		} else {
			fmt.Fprintf(&b, "pod '%s'\n", p.Name)
		}
	}
	return []byte(b.String())
}

func synthPodspec(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("Pod::Spec.new do |s|\n")
	b.WriteString("  s.name = 'sca-scan-temp'\n")
	b.WriteString("  s.version = '1.0.0'\n")
	for _, p := range pkgs {
		if p.Version != "" {
			fmt.Fprintf(&b, "  s.dependency '%s', '%s'\n", p.Name, p.Version)
		} else {
			fmt.Fprintf(&b, "  s.dependency '%s'\n", p.Name)
		}
	}
	b.WriteString("end\n")
	return []byte(b.String())
}

func synthCartfile(pkgs []Package) []byte {
	var b strings.Builder
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "latest"
		}
		fmt.Fprintf(&b, "github \"%s\" \"%s\"\n", p.Name, v)
	}
	return []byte(b.String())
}

func synthSwiftPackage(pkgs []Package) []byte {
	var b strings.Builder
	b.WriteString("// swift-tools-version:5.5\nimport PackageDescription\n\n")
	b.WriteString("let package = Package(\n")
	b.WriteString("    name: \"sca-scan-temp\",\n")
	b.WriteString("    dependencies: [\n")
	for _, p := range pkgs {
		v := p.Version
		if v == "" {
			v = "1.0.0"
		}
		fmt.Fprintf(&b, "        .package(url: \"https://github.com/sca-scan/%s.git\", exact: \"%s\"),\n", p.Name, v)
	}
	b.WriteString("    ]\n)\n")
	return []byte(b.String())
}
