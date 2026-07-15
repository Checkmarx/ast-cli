//go:build !integration

package sca

import (
	"os"
	"testing"

	"github.com/Checkmarx/manifest-parser/pkg/parser"
)

// roundTrip runs Synthesize, then re-parses the file via manifest-parser, and
// asserts that the set of name+version pairs matches the input.
func roundTrip(t *testing.T, format Format, pkgs []Package) {
	t.Helper()
	dir, err := os.MkdirTemp("", "synth-test-")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(dir)

	path, err := Synthesize(format, pkgs, dir)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	p := parser.ParsersFactory(path)
	if p == nil {
		t.Fatalf("manifest-parser has no parser for %s", path)
	}
	parsed, err := p.Parse(path)
	if err != nil {
		t.Fatalf("Parse(%s): %v", path, err)
	}

	want := make(map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		want[pkg.Name] = pkg.Version
	}
	for _, parsedPkg := range parsed {
		v, ok := want[parsedPkg.PackageName]
		if !ok {
			t.Errorf("unexpected package after parse: %s@%s", parsedPkg.PackageName, parsedPkg.Version)
			continue
		}
		if v != "" && parsedPkg.Version != v {
			t.Errorf("%s: version %q after parse, want %q", parsedPkg.PackageName, parsedPkg.Version, v)
		}
		delete(want, parsedPkg.PackageName)
	}
	for n := range want {
		t.Errorf("package %s missing after parse", n)
	}
}

func TestSynthesize_Npm(t *testing.T) {
	roundTrip(t, FormatNpmPackageJson, []Package{
		{Name: "lodash", Version: "4.17.21"},
		{Name: "axios", Version: "1.0.0"},
		{Name: "@types/node", Version: "18.0.0"},
	})
}

func TestSynthesize_Pypi(t *testing.T) {
	roundTrip(t, FormatPypiRequirements, []Package{
		{Name: "requests", Version: "2.25.1"},
		{Name: "flask", Version: "2.0.0"},
	})
}

func TestSynthesize_GoMod(t *testing.T) {
	roundTrip(t, FormatGoMod, []Package{
		{Name: "github.com/pkg/errors", Version: "v0.9.1"},
	})
}

func TestSynthesize_Csproj(t *testing.T) {
	roundTrip(t, FormatDotnetCsproj, []Package{
		{Name: "Newtonsoft.Json", Version: "13.0.1"},
	})
}

func TestSynthesize_PackagesConfig(t *testing.T) {
	roundTrip(t, FormatDotnetPackagesConfig, []Package{
		{Name: "Newtonsoft.Json", Version: "13.0.1"},
	})
}

func TestSynthesize_GradleBuild(t *testing.T) {
	roundTrip(t, FormatGradleBuild, []Package{
		{Name: "com.example:foo", Version: "1.0.0"},
		{Name: "com.example:bar", Version: "2.0.0"},
	})
}

func TestSynthesize_GradleVersionCatalog(t *testing.T) {
	roundTrip(t, FormatGradleVersionCatalog, []Package{
		{Name: "com.example:foo", Version: "1.0.0"},
		{Name: "com.example:bar", Version: "2.0.0"},
	})
}

func TestSynthesize_Sbt(t *testing.T) {
	roundTrip(t, FormatSbtBuild, []Package{
		{Name: "com.example:foo", Version: "1.0.0"},
		{Name: "com.example:bar", Version: "2.0.0"},
	})
}

func TestSynthesize_UnsupportedFormat(t *testing.T) {
	dir, _ := os.MkdirTemp("", "synth-test-")
	defer os.RemoveAll(dir)
	_, err := Synthesize(FormatUnknown, nil, dir)
	if err == nil {
		t.Errorf("Synthesize(FormatUnknown) returned nil error, want non-nil")
	}
}
