//go:build !integration

package guardrails

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// sampleJWT is a well-known test JWT (no real value) used to give 2ms a
// concrete secret to detect when we want to assert a block.
const sampleJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
	"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." +
	"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

// resolveReferencedFile is the resolver behind ScanReferencedFiles. We exercise
// it directly because the scanner integration is unchanged — only the resolver
// logic shifted from "literal stat" to "literal stat + glob fallback".

func TestResolveReferencedFile_LiteralAbsoluteHit(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yml")
	mustWrite(t, target, "k: v")

	got := resolveReferencedFile(target, nil)
	if len(got) != 1 || got[0] != target {
		t.Fatalf("expected [%q], got %v", target, got)
	}
}

func TestResolveReferencedFile_GlobFallbackFindsSibling(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	typed := filepath.Join(dir, "application-jira") // no extension
	got := resolveReferencedFile(typed, nil)

	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback to find application-jira.yml, got %v", got)
	}
}

func TestResolveReferencedFile_GlobFallbackNoSibling(t *testing.T) {
	dir := t.TempDir()
	// parent exists but nothing matches the prefix
	typed := filepath.Join(dir, "application-jira")
	if got := resolveReferencedFile(typed, nil); got != nil {
		t.Fatalf("expected nil when nothing matches, got %v", got)
	}
}

func TestResolveReferencedFile_GlobFallbackBailsOnTooManyMatches(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= maxGlobFallbackMatches; i++ {
		mustWrite(t, filepath.Join(dir, "common-prefix-"+itoa(i)+".log"), "x")
	}

	typed := filepath.Join(dir, "common-prefix")
	if got := resolveReferencedFile(typed, nil); got != nil {
		t.Fatalf("expected nil when match count exceeds cap, got %d entries", len(got))
	}
}

func TestResolveReferencedFile_TypedPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "secrets")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mustWrite(t, filepath.Join(subdir, "creds.yml"), "k: v")

	if got := resolveReferencedFile(subdir, nil); got != nil {
		t.Fatalf("expected nil for directory reference, got %v", got)
	}
}

func TestResolveReferencedFile_RelativePathResolvesAgainstWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	got := resolveReferencedFile("application-jira", []string{dir})
	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback under workspace root to find application-jira.yml, got %v", got)
	}
}

func TestResolveReferencedFile_RelativeStopsAtFirstMatchingRoot(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	mustWrite(t, filepath.Join(rootA, "config.yml"), "a")
	mustWrite(t, filepath.Join(rootB, "config.yml"), "b")

	got := resolveReferencedFile("config.yml", []string{rootA, rootB})
	if len(got) != 1 || filepath.Dir(got[0]) != rootA {
		t.Fatalf("expected resolution to stop at rootA, got %v", got)
	}
}

func TestResolveReferencedFile_CursorStyleWindowsRootNormalised(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Cursor /c:/ root form is Windows-specific")
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "application-jira.yml"), "k: v")

	// Cursor reports Windows roots as "/c:/foo"; NormalizeWorkspaceRoot strips
	// the leading slash. Confirm the resolver still finds the file via glob.
	cursorRoot := "/" + filepath.ToSlash(dir)
	got := resolveReferencedFile("application-jira", []string{cursorRoot})
	if len(got) != 1 || filepath.Base(got[0]) != "application-jira.yml" {
		t.Fatalf("expected glob fallback under Cursor-style root, got %v", got)
	}
}

func TestResolveReferencedFile_GlobMatchesMixedRegularAndDir(t *testing.T) {
	// A directory whose name shares the prefix must not be returned as a file.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "app.yml"), "k: v")
	if err := os.Mkdir(filepath.Join(dir, "app-data"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	typed := filepath.Join(dir, "app")
	got := resolveReferencedFile(typed, nil)
	sort.Strings(got)
	if len(got) != 1 || filepath.Base(got[0]) != "app.yml" {
		t.Fatalf("expected only the regular file, got %v", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

// --------------------------------------------------------------------------
// ScanWorkspaceFilesByPromptName — bare-word filename matching (Fix C)
// --------------------------------------------------------------------------

// writePolicyHelper writes a HooksPolicy to a temp ~/.checkmarx/policyhooks.json
// and redirects the home dir so LoadPolicy() picks it up. Returns a cleanup
// function that must be invoked (typically via defer) to restore the env.
func writePolicyHelper(t *testing.T, policy HooksPolicy) func() {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	dir := t.TempDir()
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cxDir, "policyhooks.json"), data, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	if runtime.GOOS == "windows" {
		orig, had := os.LookupEnv("USERPROFILE")
		os.Setenv("USERPROFILE", dir)
		return func() {
			if had {
				os.Setenv("USERPROFILE", orig)
			} else {
				os.Unsetenv("USERPROFILE")
			}
		}
	}
	orig, had := os.LookupEnv("HOME")
	os.Setenv("HOME", dir)
	return func() {
		if had {
			os.Setenv("HOME", orig)
		} else {
			os.Unsetenv("HOME")
		}
	}
}

// makeWorkspace writes a workspace directory containing the given files
// (path → contents) and returns the workspace root. Parent directories are
// created automatically. Use to set up ScanWorkspaceFilesByPromptName tests.
func makeWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "workspace")
	for rel, content := range files {
		full := filepath.Join(ws, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		mustWrite(t, full, content)
	}
	return ws
}

func TestScanWorkspaceFilesByPromptName_BasenameMatch_BlocksOnJWT(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "token = " + sampleJWT,
	})
	reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws})
	if reason == "" {
		t.Fatal("expected block: workspace file Kedar contains a JWT and the prompt names it")
	}
	if !strings.Contains(strings.ToLower(reason), "kedar") {
		t.Fatalf("reason should cite the offending file path, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_CaseInsensitive(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "secret = " + sampleJWT,
	})
	for _, prompt := range []string{
		"check kedar file",
		"Check Kedar File",
		"please review the KEDAR doc",
	} {
		if reason := ScanWorkspaceFilesByPromptName(prompt, []string{ws}); reason == "" {
			t.Fatalf("expected block for prompt %q (case-insensitive match)", prompt)
		}
	}
}

func TestScanWorkspaceFilesByPromptName_NoAtSymbolRequired(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"kedar.json": `{"jwt":"` + sampleJWT + `"}`,
	})
	if reason := ScanWorkspaceFilesByPromptName("explain kedar to me", []string{ws}); reason == "" {
		t.Fatal("expected block on a plain word `kedar` matching kedar.json by stem")
	}
}

func TestScanWorkspaceFilesByPromptName_StemMatchWithExtension(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"kedar.yaml": "token: " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("check kedar configs", []string{ws}); reason == "" {
		t.Fatal("expected block: prompt `kedar` should match `kedar.yaml` via stem")
	}
}

func TestScanWorkspaceFilesByPromptName_CleanFile_DoesNotBlock(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "just notes, nothing sensitive here",
	})
	if reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws}); reason != "" {
		t.Fatalf("expected no block when matched file has no secrets, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_NoMatch_DoesNotBlock(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "token = " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("show me the latest tests", []string{ws}); reason != "" {
		t.Fatalf("expected no block when prompt does not name any workspace file, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_SubstringInsideWord_NotMatched(t *testing.T) {
	// File "mvn" inside a basename like "foo-mvn-bar" must not be matched
	// when the prompt contains those joined word characters.
	ws := makeWorkspace(t, map[string]string{
		"foo-mvn-bar/secret.txt": "token = " + sampleJWT,
	})
	// The prompt mentions "mvn" but the only file with secrets is named
	// "secret.txt"; "mvn" appears only inside a parent dir name and is not a
	// basename token, so no scan should match.
	if reason := ScanWorkspaceFilesByPromptName("run mvn build", []string{ws}); reason != "" {
		t.Fatalf("expected no block: `mvn` is inside a directory name, not a basename, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_ShortFilenameStillScanned(t *testing.T) {
	// A 1-char filename should still be detected when it appears as a standalone
	// token in the prompt — the token-boundary check is what prevents over-block.
	ws := makeWorkspace(t, map[string]string{
		"a": "token = " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("review file a please", []string{ws}); reason == "" {
		t.Fatal("expected block: 1-char filename `a` appears as a standalone token in the prompt")
	}
}

func TestScanWorkspaceFilesByPromptName_ShortFilenameInsideWord_NotMatched(t *testing.T) {
	// File "a" must NOT match `a` inside "apple", "have", etc. — token boundary protects.
	ws := makeWorkspace(t, map[string]string{
		"a": "token = " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("there are apples here, have one", []string{ws}); reason != "" {
		t.Fatalf("expected no block: `a` only appears inside word characters, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_BothBasenameAndStem_BothBlock(t *testing.T) {
	// Workspace has BOTH `kedar` (no extension) and `Kedar.json`. The prompt
	// names `kedar`; both files match (one by basename, one by stem) and both
	// contain secrets — the rejection must cite both.
	ws := makeWorkspace(t, map[string]string{
		"kedar":      "token1 = " + sampleJWT,
		"Kedar.json": `{"jwt":"` + sampleJWT + `"}`,
	})
	reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws})
	if reason == "" {
		t.Fatal("expected block: both `kedar` and `Kedar.json` should be detected")
	}
	if !strings.Contains(reason, "kedar") || !strings.Contains(reason, "Kedar.json") {
		t.Fatalf("rejection should cite BOTH files, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_SizePolicyViolation_BlocksWithoutSecrets(t *testing.T) {
	// Policy max_file_size_kb=3. A 5 KB file with no secrets should be blocked
	// purely on the size violation: policy says it cannot enter AI context.
	policy := HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = FilesLimits{Enabled: true, MaxFileSizeKB: 3}
	defer writePolicyHelper(t, policy)()

	ws := makeWorkspace(t, map[string]string{
		"Kedar.txt": strings.Repeat("a", 5*1024), // 5 KB, no secrets
	})
	reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws})
	if reason == "" {
		t.Fatal("expected block: 5 KB file exceeds 3 KB policy cap")
	}
	if !strings.Contains(reason, "exceeds policy limit") {
		t.Fatalf("reason should cite size policy violation, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_SizePolicyAtCap_NotBlocked(t *testing.T) {
	policy := HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = FilesLimits{Enabled: true, MaxFileSizeKB: 3}
	defer writePolicyHelper(t, policy)()

	ws := makeWorkspace(t, map[string]string{
		"Kedar.txt": strings.Repeat("a", 3*1024), // exactly at cap
	})
	if reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws}); reason != "" {
		t.Fatalf("expected no block at exactly the policy cap, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_SkipsIgnoredDirs(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"node_modules/kedar.json": `{"jwt":"` + sampleJWT + `"}`,
		".git/kedar":              "token = " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("look at kedar", []string{ws}); reason != "" {
		t.Fatalf("expected no block: files only inside node_modules/.git should be pruned, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_NoWorkspaceRoots_NoOp(t *testing.T) {
	if reason := ScanWorkspaceFilesByPromptName("check kedar file", nil); reason != "" {
		t.Fatalf("expected no-op with empty workspace roots, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_CursorStyleWindowsRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Cursor /c:/foo root form is Windows-specific")
	}
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "token = " + sampleJWT,
	})
	// Convert "C:\path\workspace" -> "/c:/path/workspace" (Cursor's form).
	slashy := filepath.ToSlash(ws)
	cursorRoot := "/" + strings.ToLower(slashy[:2]) + slashy[2:]
	if reason := ScanWorkspaceFilesByPromptName("check kedar", []string{cursorRoot}); reason == "" {
		t.Fatalf("expected block: Cursor-style root %q should normalize", cursorRoot)
	}
}

func TestScanWorkspaceFilesByPromptName_RecursiveSubdirMatch(t *testing.T) {
	// File is nested several levels deep under the workspace root and not in
	// any skipped directory. The recursive walk should still find it.
	ws := makeWorkspace(t, map[string]string{
		"src/auth/internal/Kedar.txt": "token = " + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws}); reason == "" {
		t.Fatal("expected block: nested file should be found by recursive walk")
	}
}

func TestScanWorkspaceFilesByPromptName_MultiDotFilename(t *testing.T) {
	// "config.local.json": name parts ["config","local"], either token
	// in the prompt is enough to flag the file.
	ws := makeWorkspace(t, map[string]string{
		"config.local.json": `{"jwt":"` + sampleJWT + `"}`,
	})
	if reason := ScanWorkspaceFilesByPromptName("show me the local override", []string{ws}); reason == "" {
		t.Fatal("expected block: `local` is a name part of config.local.json")
	}
}

func TestScanWorkspaceFilesByPromptName_DotfileLeadingDotStripped(t *testing.T) {
	// ".env" → name part "env"; prompt mentioning "env" as a token matches.
	ws := makeWorkspace(t, map[string]string{
		".env": "TOKEN=" + sampleJWT,
	})
	if reason := ScanWorkspaceFilesByPromptName("review env settings", []string{ws}); reason == "" {
		t.Fatal("expected block: leading dot in .env should not prevent matching `env`")
	}
}

func TestScanWorkspaceFilesByPromptName_ExtensionAloneNotMatched(t *testing.T) {
	// Generic extensions like "json" must not flag every json file in the repo
	// — the trailing extension piece is dropped from filenameNameParts.
	ws := makeWorkspace(t, map[string]string{
		"kedar.json": `{"jwt":"` + sampleJWT + `"}`,
	})
	if reason := ScanWorkspaceFilesByPromptName("what is a json document", []string{ws}); reason != "" {
		t.Fatalf("expected no block: extension `json` should not match by itself, got %q", reason)
	}
}

func TestExtractPromptTokens(t *testing.T) {
	got := extractPromptTokens("check Kedar.json and id_rsa, also @secret-config!")
	want := []string{"check", "kedar", "json", "and", "id_rsa", "also", "secret-config"}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			t.Errorf("missing token %q in %v", w, got)
		}
	}
}

func TestFilenameNameParts(t *testing.T) {
	cases := map[string][]string{
		"Kedar":             {"kedar"},
		"kedar.json":        {"kedar"},
		".env":              {"env"},
		".env.local":        {"env"},
		"config.local.json": {"config", "local"},
		"Makefile":          {"makefile"},
		"id_rsa":            {"id_rsa"},
		"":                  nil,
		".":                 nil,
	}
	for in, want := range cases {
		got := filenameNameParts(in)
		if len(got) != len(want) {
			t.Errorf("filenameNameParts(%q) = %v; want %v", in, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("filenameNameParts(%q)[%d] = %q; want %q", in, i, got[i], want[i])
			}
		}
	}
}

// --------------------------------------------------------------------------
// ScanFileForSecrets — Cursor beforeReadFile content gate (Fix D)
// --------------------------------------------------------------------------

func TestScanFileForSecrets_BlocksOnJWT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Kedar.txt")
	mustWrite(t, path, "token = "+sampleJWT)

	reason := ScanFileForSecrets(path)
	if reason == "" {
		t.Fatal("expected block: file contains a JWT")
	}
	if !strings.Contains(reason, "Kedar.txt") {
		t.Fatalf("reason should cite the file path, got %q", reason)
	}
	if !strings.Contains(reason, "Do NOT attempt alternative commands") {
		t.Fatalf("reason should include DenyMessage, got %q", reason)
	}
}

func TestScanFileForSecrets_CleanFile_NoBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	mustWrite(t, path, "no secrets here, just notes")

	if reason := ScanFileForSecrets(path); reason != "" {
		t.Fatalf("expected no block for clean file, got %q", reason)
	}
}

func TestScanFileForSecrets_MissingFile_FailOpen(t *testing.T) {
	if reason := ScanFileForSecrets(filepath.Join(t.TempDir(), "does-not-exist")); reason != "" {
		t.Fatalf("expected fail-open for missing file, got %q", reason)
	}
}

func TestScanFileForSecrets_EmptyPath_NoOp(t *testing.T) {
	if reason := ScanFileForSecrets(""); reason != "" {
		t.Fatalf("expected no-op on empty path, got %q", reason)
	}
}

func TestScanFileForSecrets_OverPolicyCap_BlocksOnSize(t *testing.T) {
	policy := HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = FilesLimits{Enabled: true, MaxFileSizeKB: 3}
	defer writePolicyHelper(t, policy)()

	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	mustWrite(t, path, strings.Repeat("a", 5*1024)) // 5 KB, no secrets

	reason := ScanFileForSecrets(path)
	if reason == "" {
		t.Fatal("expected block: 5 KB file exceeds 3 KB policy cap")
	}
	if !strings.Contains(reason, "exceeds the policy size limit") {
		t.Fatalf("reason should cite size violation, got %q", reason)
	}
}

func TestScanFileForSecrets_AtPolicyCap_Allowed(t *testing.T) {
	policy := HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = FilesLimits{Enabled: true, MaxFileSizeKB: 3}
	defer writePolicyHelper(t, policy)()

	dir := t.TempDir()
	path := filepath.Join(dir, "exact.txt")
	mustWrite(t, path, strings.Repeat("a", 3*1024)) // exactly at cap, no secrets

	if reason := ScanFileForSecrets(path); reason != "" {
		t.Fatalf("expected no block at exactly the policy cap, got %q", reason)
	}
}

func TestScanWorkspaceFilesByPromptName_DenyMessageAppended(t *testing.T) {
	ws := makeWorkspace(t, map[string]string{
		"Kedar": "token = " + sampleJWT,
	})
	reason := ScanWorkspaceFilesByPromptName("check kedar file", []string{ws})
	if !strings.Contains(reason, "Do NOT attempt alternative commands") {
		t.Fatalf("expected DenyMessage no-workaround text in reason, got %q", reason)
	}
}
