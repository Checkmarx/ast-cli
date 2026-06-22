package sca

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

func scannerWith(pkgs ...ossrealtime.OssPackage) *Scanner {
	return NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{Packages: pkgs}, nil
	})
}

func TestCheckBashInstall_NoInstallCommand(t *testing.T) {
	s := scannerWith(ossrealtime.OssPackage{PackageName: "anything", Status: "Malicious"})
	finding, _ := s.CheckBashInstall("ls -la", "")
	if finding != "" {
		t.Errorf("expected empty for non-install command, got %q", finding)
	}
}

func TestCheckBashInstall_CleanInstall(t *testing.T) {
	s := scannerWith(ossrealtime.OssPackage{PackageName: "lodash", Status: "OK"})
	finding, _ := s.CheckBashInstall("npm install lodash", "")
	if finding != "" {
		t.Errorf("expected empty for clean install, got %q", finding)
	}
}

func TestCheckBashInstall_MaliciousMentionsMCP(t *testing.T) {
	s := scannerWith(ossrealtime.OssPackage{PackageName: "lodash", PackageVersion: "4.17.21", Status: "Malicious"})
	finding, remediation := s.CheckBashInstall("npm install lodash@4.17.21", "")
	if !strings.Contains(finding, "MALICIOUS") {
		t.Errorf("expected finding to mention MALICIOUS, got %q", finding)
	}
	if !strings.Contains(remediation, "mcp__Checkmarx__packageRemediation") {
		t.Errorf("expected remediation to reference MCP tool, got %q", remediation)
	}
	if !strings.Contains(remediation, "install or enable") {
		t.Errorf("expected remediation to mention installing/enabling MCP, got %q", remediation)
	}
	if !strings.Contains(remediation, "Dev Assist") {
		t.Errorf("expected remediation to mention Dev Assist fallback, got %q", remediation)
	}
}

func TestCheckBashInstall_Vulnerable(t *testing.T) {
	s := scannerWith(ossrealtime.OssPackage{PackageName: "axios", PackageVersion: "0.21.0", Status: "Vulnerable"})
	finding, remediation := s.CheckBashInstall("npm install axios@0.21.0", "")
	if !strings.Contains(finding, "vulnerabilities") {
		t.Errorf("expected vulnerable finding, got %q", finding)
	}
	if !strings.Contains(remediation, "mcp__Checkmarx__packageRemediation") {
		t.Errorf("expected remediation to reference MCP tool, got %q", remediation)
	}
	if !strings.Contains(remediation, "ignore-vulnerability") {
		t.Errorf("expected remediation to include ignore command, got %q", remediation)
	}
	if !strings.Contains(remediation, "axios") {
		t.Errorf("expected remediation to include package name, got %q", remediation)
	}
}

func TestCheckBashInstall_MaliciousTakesPrecedence(t *testing.T) {
	s := scannerWith(
		ossrealtime.OssPackage{PackageName: "vuln", Status: "Vulnerable"},
		ossrealtime.OssPackage{PackageName: "bad", Status: "Malicious"},
	)
	finding, _ := s.CheckBashInstall("npm install vuln bad", "")
	if !strings.Contains(finding, "MALICIOUS") {
		t.Errorf("expected MALICIOUS message when both present, got %q", finding)
	}
}

func TestCheckBashInstall_CompoundWithCleanThenBad(t *testing.T) {
	// First call clean, second call malicious.
	call := 0
	s := NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		call++
		if call == 1 {
			return &ossrealtime.OssPackageResults{Packages: []ossrealtime.OssPackage{
				{PackageName: "lodash", Status: "OK"},
			}}, nil
		}
		return &ossrealtime.OssPackageResults{Packages: []ossrealtime.OssPackage{
			{PackageName: "evil", Status: "Malicious"},
		}}, nil
	})
	finding, _ := s.CheckBashInstall("npm install lodash && pip install evil", "")
	if !strings.Contains(finding, "MALICIOUS") {
		t.Errorf("expected deny on second segment, got %q", finding)
	}
	if call != 2 {
		t.Errorf("expected 2 scans, got %d", call)
	}
}

func TestCheckBashInstall_FailOpenOnScannerError(t *testing.T) {
	s := NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return nil, errBoom
	})
	finding, _ := s.CheckBashInstall("npm install lodash", "")
	if finding != "" {
		t.Errorf("expected fail-open empty on scanner error, got %q", finding)
	}
}

func TestCheckManifestEdit_NonManifestNoop(t *testing.T) {
	s := scannerWith(ossrealtime.OssPackage{PackageName: "x", Status: "Malicious"})
	finding, _ := s.CheckManifestEdit("/repo/main.go", []byte("anything"), "")
	if finding != "" {
		t.Errorf("non-manifest: expected empty, got %q", finding)
	}
}

func TestCheckManifestEdit_NewMaliciousAddition(t *testing.T) {
	dir, err := os.MkdirTemp("", "ck-edit-test-")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(dir)
	pkgJSON := filepath.Join(dir, "package.json")
	// Pre-existing state: lodash already installed.
	if err := os.WriteFile(pkgJSON, []byte(`{"name":"x","dependencies":{"lodash":"4.17.21"}}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Proposed edit: add evil-pkg.
	after := []byte(`{"name":"x","dependencies":{"lodash":"4.17.21","evil-pkg":"1.0.0"}}`)

	s := scannerWith(ossrealtime.OssPackage{PackageName: "evil-pkg", PackageVersion: "1.0.0", Status: "Malicious"})
	finding, remediation := s.CheckManifestEdit(pkgJSON, after, "")
	if !strings.Contains(finding, "MALICIOUS") {
		t.Errorf("expected MALICIOUS finding, got %q", finding)
	}
	if !strings.Contains(remediation, "mcp__Checkmarx__packageRemediation") {
		t.Errorf("expected remediation to reference MCP tool, got %q", remediation)
	}
}

func TestCheckManifestEdit_OnlyVersionBumpOfCleanPkg(t *testing.T) {
	dir, err := os.MkdirTemp("", "ck-edit-test-")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(dir)
	pkgJSON := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgJSON, []byte(`{"name":"x","dependencies":{"lodash":"4.17.0"}}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	after := []byte(`{"name":"x","dependencies":{"lodash":"4.17.21"}}`)

	// Even though it's only a bump, the new version is "new" and gets scanned.
	// If the scanner returns OK, the edit is accepted.
	s := scannerWith(ossrealtime.OssPackage{PackageName: "lodash", PackageVersion: "4.17.21", Status: "OK"})
	finding, _ := s.CheckManifestEdit(pkgJSON, after, "")
	if finding != "" {
		t.Errorf("expected accept for clean version bump, got %q", finding)
	}
}

func TestDenyVulnerable_IgnoreCommandIncludesPackageData(t *testing.T) {
	pkgs := []ossrealtime.OssPackage{
		{PackageManager: "pip", PackageName: "requests", PackageVersion: "2.19.0"},
	}
	_, remediation := DenyVulnerable(pkgs, "")
	if !strings.Contains(remediation, "ignore-vulnerability") {
		t.Errorf("expected ignore-vulnerability in remediation, got %q", remediation)
	}
	if !strings.Contains(remediation, "requests") {
		t.Errorf("expected package name in remediation, got %q", remediation)
	}
	if !strings.Contains(remediation, "2.19.0") {
		t.Errorf("expected package version in remediation, got %q", remediation)
	}
	if !strings.Contains(remediation, "pip") {
		t.Errorf("expected package manager in remediation, got %q", remediation)
	}
}

func TestDenyVulnerable_MultiplePackages_EachGetsIgnoreCommand(t *testing.T) {
	pkgs := []ossrealtime.OssPackage{
		{PackageManager: "npm", PackageName: "lodash", PackageVersion: "4.17.0"},
		{PackageManager: "npm", PackageName: "axios", PackageVersion: "0.21.0"},
	}
	_, remediation := DenyVulnerable(pkgs, "")
	if strings.Count(remediation, "ignore-vulnerability") != 2 {
		t.Errorf("expected 2 ignore commands for 2 packages, got %q", remediation)
	}
	if !strings.Contains(remediation, "lodash") {
		t.Errorf("expected lodash in remediation, got %q", remediation)
	}
	if !strings.Contains(remediation, "axios") {
		t.Errorf("expected axios in remediation, got %q", remediation)
	}
}

func TestDenyVulnerable_PinsIgnoredFilePathToWorkDir(t *testing.T) {
	pkgs := []ossrealtime.OssPackage{
		{PackageManager: "npm", PackageName: "axios", PackageVersion: "0.21.0"},
	}
	workDir := filepath.Join("repo", "ws")
	_, remediation := DenyVulnerable(pkgs, workDir)
	want := "--ignored-file-path '" + ignore.PathFor(workDir) + "'"
	if !strings.Contains(remediation, want) {
		t.Errorf("expected remediation to pin %q, got %q", want, remediation)
	}
}

func TestDenyVulnerable_EmptyWorkDirOmitsIgnoredFilePath(t *testing.T) {
	pkgs := []ossrealtime.OssPackage{
		{PackageManager: "npm", PackageName: "axios", PackageVersion: "0.21.0"},
	}
	_, remediation := DenyVulnerable(pkgs, "")
	if strings.Contains(remediation, "--ignored-file-path") {
		t.Errorf("expected no ignored-file-path flag for empty workDir, got %q", remediation)
	}
}

func TestDenyMalicious_StillMentionsDevAssist(t *testing.T) {
	pkgs := []ossrealtime.OssPackage{
		{PackageName: "evil-pkg", PackageVersion: "1.0.0"},
	}
	_, remediation := DenyMalicious(pkgs)
	if !strings.Contains(remediation, "Dev Assist") {
		t.Errorf("malicious remediation should still mention Dev Assist, got %q", remediation)
	}
}

func TestCheckManifestEdit_VulnerableContainsIgnoreCommand(t *testing.T) {
	dir, err := os.MkdirTemp("", "ck-vuln-ignore-test-")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(dir)
	pkgJSON := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgJSON, []byte(`{"name":"x","dependencies":{}}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	after := []byte(`{"name":"x","dependencies":{"axios":"0.21.0"}}`)

	s := scannerWith(ossrealtime.OssPackage{
		PackageManager: "npm", PackageName: "axios", PackageVersion: "0.21.0", Status: "Vulnerable",
	})
	finding, remediation := s.CheckManifestEdit(pkgJSON, after, "")
	if !strings.Contains(finding, "vulnerabilities") {
		t.Errorf("expected vulnerable finding, got %q", finding)
	}
	if !strings.Contains(remediation, "ignore-vulnerability") {
		t.Errorf("expected ignore command in remediation, got %q", remediation)
	}
	if !strings.Contains(remediation, "axios") {
		t.Errorf("expected package name in remediation, got %q", remediation)
	}
}

func TestExistingIgnoreFilePath_ResolvesUnderWorkDir(t *testing.T) {
	workDir := t.TempDir()
	dir := filepath.Join(workDir, ".checkmarx")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checkmarxIgnoredTempList.json"), []byte("[]"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := existingIgnoreFilePath(workDir)
	if want := ignore.PathFor(workDir); got != want {
		t.Errorf("expected ignore path %q under workDir, got %q", want, got)
	}
}

func TestExistingIgnoreFilePath_MissingReturnsEmpty(t *testing.T) {
	// Empty workspace: no .checkmarx file → no filtering path passed to the scanner.
	if got := existingIgnoreFilePath(t.TempDir()); got != "" {
		t.Errorf("expected empty for missing ignore file, got %q", got)
	}
}

var errBoom = stringError("boom")

type stringError string

func (e stringError) Error() string { return string(e) }
