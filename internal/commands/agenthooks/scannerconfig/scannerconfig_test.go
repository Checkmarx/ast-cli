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
