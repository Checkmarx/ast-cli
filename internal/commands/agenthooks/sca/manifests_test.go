//go:build !integration

package sca

import "testing"

func TestIsManifest(t *testing.T) {
	tests := []struct {
		path    string
		wantOK  bool
		wantFmt Format
	}{
		{"package.json", true, FormatNpmPackageJson},
		{"/repo/package.json", true, FormatNpmPackageJson},
		{"requirements.txt", true, FormatPypiRequirements},
		{"requirements-dev.txt", true, FormatPypiRequirements},
		{"packages.txt", true, FormatPypiRequirements},
		{"go.mod", true, FormatGoMod},
		{"pom.xml", true, FormatMavenPom},
		{"app.csproj", true, FormatDotnetCsproj},
		{"Project.csproj", true, FormatDotnetCsproj},
		{"Directory.Packages.props", true, FormatDotnetDirectoryPackagesProps},
		{"packages.config", true, FormatDotnetPackagesConfig},
		{"build.gradle", true, FormatGradleBuild},
		{"build.gradle.kts", true, FormatGradleBuild},
		{"/repo/app/build.gradle.kts", true, FormatGradleBuild},
		{"libs.versions.toml", true, FormatGradleVersionCatalog},
		{"build.sbt", true, FormatSbtBuild},
		{"plugins.sbt", true, FormatSbtBuild},
		{"constraints.txt", true, FormatPypiRequirements},
		{"setup.cfg", true, FormatPypiRequirements},
		{"setup.py", true, FormatPypiRequirements},
		{"pyproject.toml", true, FormatPypiRequirements},
		{"Podfile", true, FormatCocoaPodsPodfile},
		{"synth.podspec", true, FormatCocoaPodsPodspec},
		{"lib.podspec", true, FormatCocoaPodsPodspec},
		{"lib.podspec.json", true, FormatCocoaPodsPodspec},
		{"Cartfile", true, FormatCarthage},
		{"Cartfile.private", true, FormatCarthage},
		{"Package.swift", true, FormatSwiftPackageManager},
		{"Package@swift-5.5.swift", true, FormatSwiftPackageManager},
		{"bower.json", true, FormatBower},
		{"composer.json", true, FormatComposerJson},
		{"pubspec.yaml", true, FormatPubspecYaml},
		{"Gemfile", true, FormatGemfile},

		// Negatives.
		{"main.go", false, FormatUnknown},
		{"Dockerfile", false, FormatUnknown},
		{"README.md", false, FormatUnknown},
		{"random.txt", false, FormatUnknown},
		{"", false, FormatUnknown},
	}
	for _, tt := range tests {
		gotFmt, gotOK := IsManifest(tt.path)
		if gotOK != tt.wantOK || gotFmt != tt.wantFmt {
			t.Errorf("IsManifest(%q) = (%v, %v), want (%v, %v)",
				tt.path, gotFmt, gotOK, tt.wantFmt, tt.wantOK)
		}
	}
}

// Test ManagerName returns the correct package manager name for each format
func TestFormatManagerName(t *testing.T) {
	tests := []struct {
		format   Format
		wantName string
	}{
		{FormatNpmPackageJson, "npm"},
		{FormatPypiRequirements, "pypi"},
		{FormatGoMod, "go"},
		{FormatMavenPom, "maven"},
		{FormatDotnetCsproj, "nuget"},
		{FormatDotnetDirectoryPackagesProps, "nuget"},
		{FormatDotnetPackagesConfig, "nuget"},
		{FormatGradleBuild, "gradle"},
		{FormatGradleVersionCatalog, "gradle"},
		{FormatSbtBuild, "sbt"},
		{FormatCocoaPodsPodfile, "cocoapods"},
		{FormatCocoaPodsPodspec, "cocoapods"},
		{FormatCarthage, "carthage"},
		{FormatSwiftPackageManager, "swift"},
		{FormatBower, "npm"},
		{FormatComposerJson, "packagist"},
		{FormatPubspecYaml, "pub"},
		{FormatGemfile, "rubygems"},
		{FormatUnknown, ""},
	}
	for _, tt := range tests {
		gotName := tt.format.ManagerName()
		if gotName != tt.wantName {
			t.Errorf("Format(%d).ManagerName() = %q, want %q", tt.format, gotName, tt.wantName)
		}
	}
}

// Test SynthFileName returns the correct filename for each format
func TestFormatSynthFileName(t *testing.T) {
	tests := []struct {
		format   Format
		wantName string
	}{
		{FormatNpmPackageJson, "package.json"},
		{FormatPypiRequirements, "requirements.txt"},
		{FormatGoMod, "go.mod"},
		{FormatMavenPom, "pom.xml"},
		{FormatDotnetCsproj, "synth.csproj"},
		{FormatDotnetDirectoryPackagesProps, "Directory.Packages.props"},
		{FormatDotnetPackagesConfig, "packages.config"},
		{FormatGradleBuild, "build.gradle"},
		{FormatGradleVersionCatalog, "libs.versions.toml"},
		{FormatSbtBuild, "synth.sbt"},
		{FormatCocoaPodsPodfile, "Podfile"},
		{FormatCocoaPodsPodspec, "synth.podspec"},
		{FormatCarthage, "Cartfile"},
		{FormatSwiftPackageManager, "Package.swift"},
		{FormatBower, "bower.json"},
		{FormatComposerJson, "composer.json"},
		{FormatPubspecYaml, "pubspec.yaml"},
		{FormatGemfile, "Gemfile"},
		{FormatUnknown, ""},
	}
	for _, tt := range tests {
		gotName := tt.format.SynthFileName()
		if gotName != tt.wantName {
			t.Errorf("Format(%d).SynthFileName() = %q, want %q", tt.format, gotName, tt.wantName)
		}
	}
}
