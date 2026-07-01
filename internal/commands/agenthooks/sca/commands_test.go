//go:build !integration

package sca

import (
	"reflect"
	"sort"
	"testing"
)

func sortedPackages(pkgs []Package) []Package {
	out := append([]Package(nil), pkgs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Version < out[j].Version
	})
	return out
}

func wantPackages(t *testing.T, got, want []Package) {
	t.Helper()
	if !reflect.DeepEqual(sortedPackages(got), sortedPackages(want)) {
		t.Errorf("packages mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestParseInstall_SimpleNpm(t *testing.T) {
	tests := []struct {
		command string
		want    []Package
	}{
		{"npm install lodash", []Package{{Name: "lodash"}}},
		{"npm i lodash", []Package{{Name: "lodash"}}},
		{"npm add lodash@4.17.21", []Package{{Name: "lodash", Version: "4.17.21"}}},
		{"yarn add react", []Package{{Name: "react"}}},
		{"pnpm add react@18.0.0", []Package{{Name: "react", Version: "18.0.0"}}},
		{"pnpm install lodash", []Package{{Name: "lodash"}}},
		{"npm install @types/node@18.0.0", []Package{{Name: "@types/node", Version: "18.0.0"}}},
		{"npm install @types/node", []Package{{Name: "@types/node"}}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		if got[0].Manager != ManagerNpm {
			t.Errorf("%q: got manager %v, want npm", tt.command, got[0].Manager)
		}
		wantPackages(t, got[0].Packages, tt.want)
	}
}

func TestParseInstall_SimplePypi(t *testing.T) {
	tests := []struct {
		command string
		want    []Package
	}{
		{"pip install requests", []Package{{Name: "requests"}}},
		{"pip install requests==2.25.1", []Package{{Name: "requests", Version: "2.25.1"}}},
		{"pip install requests>=2.0", []Package{{Name: "requests"}}},
		{"pip3 install requests", []Package{{Name: "requests"}}},
		{"python -m pip install requests", []Package{{Name: "requests"}}},
		{"python3 -m pip install requests==2.25.1", []Package{{Name: "requests", Version: "2.25.1"}}},
		{"pipenv install requests", []Package{{Name: "requests"}}},
		{"poetry add requests", []Package{{Name: "requests"}}},
		{"uv add requests", []Package{{Name: "requests"}}},
		{"uv pip install requests", []Package{{Name: "requests"}}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		if got[0].Manager != ManagerPypi {
			t.Errorf("%q: got manager %v, want pypi", tt.command, got[0].Manager)
		}
		wantPackages(t, got[0].Packages, tt.want)
	}
}

func TestParseInstall_SimpleDotnet(t *testing.T) {
	tests := []struct {
		command string
		want    []Package
	}{
		{"dotnet add package Newtonsoft.Json", []Package{{Name: "Newtonsoft.Json"}}},
		{"dotnet add package Newtonsoft.Json -v 13.0.1", []Package{{Name: "Newtonsoft.Json", Version: "13.0.1"}}},
		{"dotnet add package Newtonsoft.Json --version 13.0.1", []Package{{Name: "Newtonsoft.Json", Version: "13.0.1"}}},
		{"nuget install Newtonsoft.Json", []Package{{Name: "Newtonsoft.Json"}}},
		{"nuget install Newtonsoft.Json -Version 13.0.1", []Package{{Name: "Newtonsoft.Json", Version: "13.0.1"}}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		if got[0].Manager != ManagerDotnet {
			t.Errorf("%q: got manager %v, want dotnet", tt.command, got[0].Manager)
		}
		wantPackages(t, got[0].Packages, tt.want)
	}
}

func TestParseInstall_SimpleGo(t *testing.T) {
	tests := []struct {
		command string
		want    []Package
	}{
		{"go get github.com/pkg/errors", []Package{{Name: "github.com/pkg/errors"}}},
		{"go get github.com/pkg/errors@v0.9.1", []Package{{Name: "github.com/pkg/errors", Version: "v0.9.1"}}},
		{"go install golang.org/x/tools/cmd/goimports@latest", []Package{{Name: "golang.org/x/tools/cmd/goimports", Version: "latest"}}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		if got[0].Manager != ManagerGo {
			t.Errorf("%q: got manager %v, want go", tt.command, got[0].Manager)
		}
		wantPackages(t, got[0].Packages, tt.want)
	}
}

func TestParseInstall_SimpleMaven(t *testing.T) {
	got := ParseInstall("mvn dependency:get -Dartifact=org.apache.commons:commons-lang3:3.12.0")
	if len(got) != 1 {
		t.Fatalf("got %d requests, want 1", len(got))
	}
	if got[0].Manager != ManagerMaven {
		t.Errorf("got manager %v, want maven", got[0].Manager)
	}
	want := []Package{{Name: "org.apache.commons:commons-lang3", Version: "3.12.0"}}
	wantPackages(t, got[0].Packages, want)
}

func TestParseInstall_MultiPackage(t *testing.T) {
	tests := []struct {
		command  string
		wantMgr  Manager
		wantPkgs []Package
	}{
		{
			"npm install lodash axios express",
			ManagerNpm,
			[]Package{{Name: "lodash"}, {Name: "axios"}, {Name: "express"}},
		},
		{
			"npm install lodash@4.0.0 axios@latest express",
			ManagerNpm,
			[]Package{{Name: "lodash", Version: "4.0.0"}, {Name: "axios", Version: "latest"}, {Name: "express"}},
		},
		{
			"yarn add a b c",
			ManagerNpm,
			[]Package{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		},
		{
			"pnpm add a b c",
			ManagerNpm,
			[]Package{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		},
		{
			"pip install pkg1==1.0 pkg2>=2.0 pkg3",
			ManagerPypi,
			[]Package{{Name: "pkg1", Version: "1.0"}, {Name: "pkg2"}, {Name: "pkg3"}},
		},
		{
			"poetry add a b",
			ManagerPypi,
			[]Package{{Name: "a"}, {Name: "b"}},
		},
		{
			"uv add a b",
			ManagerPypi,
			[]Package{{Name: "a"}, {Name: "b"}},
		},
		{
			"go get pkg1 pkg2@v1.0 pkg3",
			ManagerGo,
			[]Package{{Name: "pkg1"}, {Name: "pkg2", Version: "v1.0"}, {Name: "pkg3"}},
		},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		if got[0].Manager != tt.wantMgr {
			t.Errorf("%q: got manager %v, want %v", tt.command, got[0].Manager, tt.wantMgr)
		}
		wantPackages(t, got[0].Packages, tt.wantPkgs)
	}
}

func TestParseInstall_Compound(t *testing.T) {
	tests := []struct {
		command   string
		wantCount int
		wantMgrs  []Manager
	}{
		{"cd /repo && npm install lodash", 1, []Manager{ManagerNpm}},
		{"npm install lodash && npm test", 1, []Manager{ManagerNpm}},
		{"npm install lodash; npm install axios", 2, []Manager{ManagerNpm, ManagerNpm}},
		{"pip install lodash || echo failed", 1, []Manager{ManagerPypi}},
		{"echo \"starting\" && npm install lodash@4.0.0", 1, []Manager{ManagerNpm}},
		{"npm install lodash && yarn add axios", 2, []Manager{ManagerNpm, ManagerNpm}},
		{"npm install a b && pip install x y", 2, []Manager{ManagerNpm, ManagerPypi}},
		{"git pull && pip install requests", 1, []Manager{ManagerPypi}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != tt.wantCount {
			t.Errorf("%q: got %d requests, want %d (%v)", tt.command, len(got), tt.wantCount, got)
			continue
		}
		for i, m := range tt.wantMgrs {
			if got[i].Manager != m {
				t.Errorf("%q: request[%d] manager %v, want %v", tt.command, i, got[i].Manager, m)
			}
		}
	}
}

func TestParseInstall_PipRequirementRef(t *testing.T) {
	got := ParseInstall("pip install -r requirements.txt")
	if len(got) != 1 {
		t.Fatalf("got %d requests, want 1", len(got))
	}
	if got[0].ManifestRef != "requirements.txt" {
		t.Errorf("got ref %q, want %q", got[0].ManifestRef, "requirements.txt")
	}
	if len(got[0].Packages) != 0 {
		t.Errorf("got %d packages, want 0", len(got[0].Packages))
	}
}

func TestParseInstall_Negative(t *testing.T) {
	negatives := []string{
		"",
		"npm run build",
		"npm test",
		"pip uninstall pkg",
		"pip list",
		"git clone https://example.com/repo",
		"npm install", // bare
		"pip install", // bare
		"ls -la",
		"go build ./...",
		"docker run --rm img",
	}
	for _, cmd := range negatives {
		got := ParseInstall(cmd)
		if len(got) != 0 {
			t.Errorf("%q: got %d requests, want 0 (%v)", cmd, len(got), got)
		}
	}
}

func TestParseInstall_Flags(t *testing.T) {
	tests := []struct {
		command string
		want    []Package
	}{
		{"npm install --save-dev typescript", []Package{{Name: "typescript"}}},
		{"npm install -g pkg", []Package{{Name: "pkg"}}},
		{"npm install -D typescript prettier", []Package{{Name: "typescript"}, {Name: "prettier"}}},
		{"pip install --upgrade requests", []Package{{Name: "requests"}}},
	}
	for _, tt := range tests {
		got := ParseInstall(tt.command)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", tt.command, len(got))
			continue
		}
		wantPackages(t, got[0].Packages, tt.want)
	}
}

func TestParseInstall_LeadingNoOps(t *testing.T) {
	tests := []string{
		"sudo npm install lodash",
		"time npm install lodash",
		"NODE_ENV=production npm install lodash",
		"sudo NODE_ENV=production npm install lodash",
	}
	for _, cmd := range tests {
		got := ParseInstall(cmd)
		if len(got) != 1 {
			t.Errorf("%q: got %d requests, want 1", cmd, len(got))
			continue
		}
		if got[0].Manager != ManagerNpm {
			t.Errorf("%q: got manager %v, want npm", cmd, got[0].Manager)
		}
		wantPackages(t, got[0].Packages, []Package{{Name: "lodash"}})
	}
}

func TestParseInstall_ShellExpansionDropped(t *testing.T) {
	// $(cat req.txt) is opaque — we cannot statically know the packages, so
	// the segment should resolve to zero install requests rather than emit
	// garbage package names.
	got := ParseInstall("pip install $(cat req.txt)")
	if len(got) != 0 {
		t.Errorf("expected $() to drop, got %d requests (%v)", len(got), got)
	}

	got = ParseInstall("pip install `echo lodash`")
	if len(got) != 0 {
		t.Errorf("expected backtick to drop, got %d requests (%v)", len(got), got)
	}

	// Mixed: real package + shell expansion → keep the real one.
	got = ParseInstall("pip install requests $(cat extras)")
	if len(got) != 1 {
		t.Fatalf("expected 1 request, got %d (%v)", len(got), got)
	}
	if len(got[0].Packages) != 1 || got[0].Packages[0].Name != "requests" {
		t.Errorf("got %v, want [requests]", got[0].Packages)
	}
}

func TestParseInstall_QuotedStrings(t *testing.T) {
	// Strings that *contain* an install verb but aren't installs.
	got := ParseInstall(`echo "npm install lodash"`)
	if len(got) != 0 {
		t.Errorf("quoted install verb should not match, got %d requests", len(got))
	}

	// Subshell containing an install.
	got = ParseInstall(`bash -c "echo hello && npm install lodash"`)
	// We don't recursively parse `-c "..."` arg payloads — but $() and `` we do.
	// So this is a no-op (we don't dive into bash -c). Document the behaviour.
	if len(got) != 0 {
		t.Logf("bash -c payload: got %d requests (currently no-op by design)", len(got))
	}
}
