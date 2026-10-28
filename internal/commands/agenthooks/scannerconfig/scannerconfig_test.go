//go:build !integration

package scannerconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), fileName)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnabledDefaults(t *testing.T) {
	if !enabledIn("", ASCA) {
		t.Error("empty path must mean enabled")
	}
	if !enabledIn(filepath.Join(t.TempDir(), "missing"), SCA) {
		t.Error("missing file must mean enabled")
	}
	if !enabledIn(write(t, "# nothing set\n"), KICS) {
		t.Error("absent key must mean enabled")
	}
}

func TestEnabledFalseValues(t *testing.T) {
	for _, v := range []string{"false", "FALSE", "0", "no", "off", "Disabled", "  false  "} {
		if enabledIn(write(t, "cx_asca_scanner_enabled="+v+"\n"), ASCA) {
			t.Errorf("%q should disable", v)
		}
	}
}

func TestEnabledGarbageStaysOn(t *testing.T) {
	for _, v := range []string{"flase", "", "banana", "2", "true"} {
		if !enabledIn(write(t, "cx_kics_scanner_enabled="+v+"\n"), KICS) {
			t.Errorf("%q must stay enabled", v)
		}
	}
}

func TestKeysAreIndependent(t *testing.T) {
	p := write(t, "cx_sca_scanner_enabled=false\ncx_secret_scanner_enabled=true\n")
	if enabledIn(p, SCA) || !enabledIn(p, ASCA) || !enabledIn(p, KICS) || !enabledIn(p, Secrets) {
		t.Error("only SCA should be disabled")
	}
}

func TestCommentedLineIgnoredAndLastWins(t *testing.T) {
	if !enabledIn(write(t, "# cx_asca_scanner_enabled=false\n"), ASCA) {
		t.Error("commented key must be ignored")
	}
	if !enabledIn(write(t, "cx_asca_scanner_enabled=false\ncx_asca_scanner_enabled=true\n"), ASCA) {
		t.Error("last occurrence should win")
	}
}

func TestBOMAndCRLF(t *testing.T) {
	p := write(t, "\xef\xbb\xbfcx_secret_scanner_enabled=false\r\ncx_asca_scanner_enabled=true\r\n")
	if enabledIn(p, Secrets) || !enabledIn(p, ASCA) {
		t.Error("BOM/CRLF file misparsed")
	}
}

func TestOversizedFailsOpen(t *testing.T) {
	p := write(t, strings.Repeat("# pad\n", 5000)+"cx_asca_scanner_enabled=false\n")
	if !enabledIn(p, ASCA) {
		t.Error("oversized file must fail open to enabled")
	}
}

func TestPathUsesHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if got := Path(); got != filepath.Join(dir, ".checkmarx", fileName) {
		t.Errorf("unexpected path %q", got)
	}
	_ = os.MkdirAll(filepath.Join(dir, ".checkmarx"), 0o755)
	_ = os.WriteFile(Path(), []byte("cx_kics_scanner_enabled=off\n"), 0o600)
	if Enabled(KICS) || !Enabled(ASCA) {
		t.Error("Enabled should read the file under the user's home")
	}
}

func TestThresholdIn(t *testing.T) {
	cases := map[string]int{
		"":                                 1, // no file content
		"cx_severity_threshold=\n":         1, // blank
		"cx_severity_threshold=low\n":      1,
		"cx_severity_threshold=Medium\n":   2,
		"cx_severity_threshold= HIGH \n":   3,
		"cx_severity_threshold=critical\n": 4,
		"cx_severity_threshold=hihg\n":     1, // invalid
		"# cx_severity_threshold=high\n":   1, // commented
		"cx_severity_threshold=high\ncx_severity_threshold=low\n": 1, // last wins
	}
	for body, want := range cases {
		if got := thresholdIn(write(t, body)); got != want {
			t.Errorf("%q: got %d, want %d", body, got, want)
		}
	}
	if thresholdIn(filepath.Join(t.TempDir(), "missing")) != 1 || thresholdIn("") != 1 {
		t.Error("missing file / path must mean low")
	}
}

func TestBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".checkmarx"), 0o755)
	set := func(body string) {
		if err := os.WriteFile(Path(), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	set("cx_severity_threshold=high\n")
	for sev, want := range map[string]bool{
		"Low": true, "MEDIUM": true, "High": false, "Critical": false,
		"": false, "Malicious": false, "weird": false, // unranked is never relaxed
	} {
		if got := BelowThreshold(sev); got != want {
			t.Errorf("high threshold, %q: got %v, want %v", sev, got, want)
		}
	}
	set("cx_severity_threshold=\n")
	if BelowThreshold("Low") {
		t.Error("blank threshold must filter nothing")
	}
}

func writeIDE(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ideFileName)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSelfHealingIn(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"key missing is on", "cx_schema_version=1\n", true},
		{"explicit true is on", "cx_self_healing_enabled=true\n", true},
		{"garbage is on", "cx_self_healing_enabled=flase\n", true},
		{"empty value is on", "cx_self_healing_enabled=\n", true},
		{"false is off", "cx_schema_version=1\ncx_self_healing_enabled=false\n", false},
		{"no schema key, off", "cx_self_healing_enabled=false\n", false},
		{"caps / spaces / CRLF / BOM, off", "\xef\xbb\xbfcx_schema_version= 1 \r\ncx_self_healing_enabled= Disabled \r\n", false},
		{"0 / no / off are off", "cx_self_healing_enabled=0\ncx_self_healing_enabled=no\ncx_self_healing_enabled=off\n", false},
		{"last occurrence wins", "cx_self_healing_enabled=false\ncx_self_healing_enabled=true\n", true},
		{"commented key ignored", "# cx_self_healing_enabled=false\n", true},
		{"unknown schema is on", "cx_schema_version=2\ncx_self_healing_enabled=false\n", true},
		{"blank schema is on", "cx_schema_version=\ncx_self_healing_enabled=false\n", true},
		{"oversized is on", strings.Repeat("# pad\n", 5000) + "cx_self_healing_enabled=false\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selfHealingIn(writeIDE(t, tc.body)); got != tc.want {
				t.Fatalf("selfHealingIn(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
	if !selfHealingIn(filepath.Join(t.TempDir(), "missing")) || !selfHealingIn("") {
		t.Error("missing file / path must mean on")
	}
	if !selfHealingIn(t.TempDir()) {
		t.Error("unreadable (directory) must mean on")
	}
}

func TestIDEEnabledIn(t *testing.T) {
	cli := write(t, "cx_asca_scanner_enabled=false\n") // terminal: ASCA off, others on
	cases := []struct {
		name string
		ide  string // "" = no IDE file
		s    Scanner
		want bool
	}{
		{"no IDE file falls back to CLI (off)", "", ASCA, false},
		{"no IDE file falls back to CLI (on)", "", KICS, true},
		{"valid IDE file, key missing is on despite CLI off", "cx_schema_version=1\n", ASCA, true},
		{"valid IDE file, key off", "cx_kics_scanner_enabled=false\n", KICS, false},
		{"valid IDE file, key on", "cx_asca_scanner_enabled=true\n", ASCA, true},
		{"valid IDE file, garbage is on", "cx_sca_scanner_enabled=banana\n", SCA, true},
		{"valid IDE file, secrets off", "cx_secret_scanner_enabled=OFF\n", Secrets, false},
		{"unknown schema falls back to CLI", "cx_schema_version=9\ncx_asca_scanner_enabled=true\n", ASCA, false},
		{"oversized IDE file falls back to CLI", strings.Repeat("# pad\n", 5000), ASCA, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ide := filepath.Join(t.TempDir(), "missing")
			if tc.ide != "" {
				ide = writeIDE(t, tc.ide)
			}
			if got := ideEnabledIn(ide, cli, tc.s); got != tc.want {
				t.Fatalf("ideEnabledIn(%s) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestIDEPathUsesHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if got := IDEPath(); got != filepath.Join(dir, ".checkmarx", "cx-onboarding-ide.properties") {
		t.Errorf("unexpected path %q", got)
	}
	if !SelfHealingEnabled() || !IDEEnabled(ASCA) {
		t.Error("no files under home must mean on")
	}
	_ = os.MkdirAll(filepath.Join(dir, ".checkmarx"), 0o755)
	_ = os.WriteFile(IDEPath(), []byte("cx_self_healing_enabled=false\ncx_kics_scanner_enabled=false\n"), 0o600)
	if SelfHealingEnabled() || IDEEnabled(KICS) || !IDEEnabled(ASCA) {
		t.Error("SelfHealingEnabled / IDEEnabled should read the IDE file under the user's home")
	}
	if !Enabled(KICS) {
		t.Error("Enabled (terminal) must never read the IDE file")
	}
}

func TestIDEThresholdIn(t *testing.T) {
	cases := map[string]int{
		"cx_schema_version=1\n":                             1, // VS Code writes no threshold: low
		"cx_severity_threshold=high\n":                      3,
		"cx_schema_version=1\ncx_severity_threshold=\n":     1, // blank
		"cx_severity_threshold=hihg\n":                      1, // invalid
		"cx_schema_version=2\ncx_severity_threshold=high\n": 1, // unknown schema: file ignored
	}
	for body, want := range cases {
		if got := ideThresholdIn(writeIDE(t, body)); got != want {
			t.Errorf("%q: got %d, want %d", body, got, want)
		}
	}
	if ideThresholdIn(filepath.Join(t.TempDir(), "missing")) != 1 || ideThresholdIn("") != 1 {
		t.Error("missing IDE file / path must mean low")
	}
}

func TestIDEBelowThresholdIgnoresTerminalThreshold(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".checkmarx"), 0o755)
	_ = os.WriteFile(Path(), []byte("cx_severity_threshold=critical\n"), 0o600)
	_ = os.WriteFile(IDEPath(), []byte("cx_schema_version=1\ncx_self_healing_enabled=true\n"), 0o600)
	if !BelowThreshold("High") {
		t.Error("terminal: a critical threshold must relax High")
	}
	if IDEBelowThreshold("Low") || IDEBelowThreshold("High") {
		t.Error("VS Code without a threshold must block at every severity (low)")
	}
}
