//go:build !integration

package sca

import "testing"

func TestIsManifest(t *testing.T) {
	tests := []struct {
		path     string
		wantOK   bool
		wantFmt  Format
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

