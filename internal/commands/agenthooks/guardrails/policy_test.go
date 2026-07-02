//go:build !integration

package guardrails_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
)

// setHomeDir redirects os.UserHomeDir() to dir and returns a cleanup function.
func setHomeDir(dir string) func() {
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

// writePolicy writes a HooksPolicy to a temp file and sets the home dir so
// LoadPolicy picks it up. Returns a cleanup function.
func writePolicy(t *testing.T, policy guardrails.HooksPolicy) func() {
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
	return setHomeDir(dir)
}

// currentOS returns the policy OS label for the current platform.
func currentOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// --------------------------------------------------------------------------
// LoadPolicy
// --------------------------------------------------------------------------

func TestLoadPolicy_MissingFile(t *testing.T) {
	dir := t.TempDir()
	defer setHomeDir(dir)()

	if got := guardrails.LoadPolicy(); got != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestLoadPolicy_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	cxDir := filepath.Join(dir, ".checkmarx")
	os.MkdirAll(cxDir, 0o755)
	os.WriteFile(filepath.Join(cxDir, "policyhooks.json"), []byte("not-json{{{"), 0o644)
	defer setHomeDir(dir)()

	if got := guardrails.LoadPolicy(); got != nil {
		t.Fatal("expected nil for malformed JSON")
	}
}

func TestLoadPolicy_UTF8BOMPrefix(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	body, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}

	dir := t.TempDir()
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, body...)
	if err := os.WriteFile(filepath.Join(cxDir, "policyhooks.json"), withBOM, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	defer setHomeDir(dir)()

	got := guardrails.LoadPolicy()
	if got == nil {
		t.Fatal("expected non-nil policy when file is BOM-prefixed; LoadPolicy should strip the UTF-8 BOM")
	}
	if !got.DefaultPolicy.BlacklistTools.Enabled {
		t.Fatal("BlacklistTools.Enabled should be true after BOM strip")
	}
}

func TestLoadPolicy_ValidJSON(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	cleanup := writePolicy(t, policy)
	defer cleanup()

	got := guardrails.LoadPolicy()
	if got == nil {
		t.Fatal("expected non-nil policy")
	}
	if !got.DefaultPolicy.BlacklistTools.Enabled {
		t.Fatal("BlacklistTools.Enabled should be true")
	}
}

// --------------------------------------------------------------------------
// LoadBlacklistedCommands
// --------------------------------------------------------------------------

func TestLoadBlacklistedCommands_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = false
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "rm -rf", OS: []string{currentOS()}, Category: "destructive", Risk: "bad"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	got := guardrails.LoadBlacklistedCommands()
	if len(got) != 0 {
		t.Fatalf("expected empty map when disabled, got %d entries", len(got))
	}
}

func TestLoadBlacklistedCommands_OSMatch(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "danger-cmd", OS: []string{currentOS()}, Category: "test", Risk: "none"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	got := guardrails.LoadBlacklistedCommands()
	if _, ok := got["danger-cmd"]; !ok {
		t.Fatal("expected danger-cmd in blacklist for current OS")
	}
}

func TestLoadBlacklistedCommands_OSNoMatch(t *testing.T) {
	wrongOS := "linux"
	if runtime.GOOS == "linux" {
		wrongOS = "windows"
	}
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "other-os-cmd", OS: []string{wrongOS}, Category: "test", Risk: "none"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	got := guardrails.LoadBlacklistedCommands()
	if _, ok := got["other-os-cmd"]; ok {
		t.Fatal("should not include tool for wrong OS")
	}
}

func TestLoadBlacklistedCommands_CaseInsensitive(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "RM -RF", OS: []string{currentOS()}, Category: "destructive", Risk: "bad"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	got := guardrails.LoadBlacklistedCommands()
	if _, ok := got["rm -rf"]; !ok {
		t.Fatal("expected lowercased key in blacklist map")
	}
}

// --------------------------------------------------------------------------
// CheckShellCommand — blacklist
// --------------------------------------------------------------------------

func TestCheckShellCommand_Blacklisted(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "rm -rf", OS: []string{currentOS()}, Category: "destructive", Risk: "wipes files"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, needsConfirm, reason := guardrails.CheckShellCommand("rm -rf /tmp/foo", "")
	if !blocked {
		t.Fatal("expected blocked=true for blacklisted command")
	}
	if needsConfirm {
		t.Fatal("expected needsConfirm=false for blacklisted command")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckShellCommand_Clean(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "rm -rf", OS: []string{currentOS()}, Category: "destructive", Risk: "bad"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, _, _ := guardrails.CheckShellCommand("ls -la", "")
	if blocked {
		t.Fatal("expected clean command to pass")
	}
}

// --------------------------------------------------------------------------
// CheckShellCommand — tool rules
// --------------------------------------------------------------------------

func makeToolRulePolicy(rule guardrails.ToolRule) guardrails.HooksPolicy {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{rule}
	return policy
}

func TestCheckShellCommand_ToolRule_ExcludedArg(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:          "t1",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsExclude: []string{"deploy"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	blocked, needsConfirm, reason := guardrails.CheckShellCommand("mvn deploy", "/project")
	if !blocked {
		t.Fatal("expected blocked for excluded arg")
	}
	if needsConfirm {
		t.Fatal("excluded arg should hard-deny, not ask")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestCheckShellCommand_ToolRule_UnknownArg_AlwaysAsks(t *testing.T) {
	// Unmatched args always produce needsConfirm=true regardless of rule.Action.
	for _, action := range []string{"ask", "block", "allow", ""} {
		rule := guardrails.ToolRule{
			ID:          "t2",
			Tool:        []string{"mvn"},
			OS:          []string{currentOS()},
			ArgsInclude: []string{"compile", "test"},
		}
		cleanup := writePolicy(t, makeToolRulePolicy(rule))

		blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn unknown-goal", "")
		if !blocked {
			t.Fatalf("action=%q: expected blocked for arg not in whitelist", action)
		}
		if !needsConfirm {
			t.Fatalf("action=%q: unmatched arg must always ask (needsConfirm=true), not hard-block", action)
		}
		cleanup()
	}
}

func TestCheckShellCommand_ToolRule_AllowedArg(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:          "t4",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsInclude: []string{"compile", "test"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	blocked, _, _ := guardrails.CheckShellCommand("mvn compile", "")
	if blocked {
		t.Fatal("expected allowed for whitelisted arg")
	}
}

func TestCheckShellCommand_ToolRule_GlobMatch_Allowed(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:          "tg1",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsInclude: []string{"compile", "-D*", "--*"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	// "-Dmaven.test.skip=true" matches glob "-D*" → allowed
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile -Dmaven.test.skip=true", ""); blocked {
		t.Fatal("expected -D* glob to allow -Dmaven.test.skip=true")
	}
	// "--offline" matches glob "--*" → allowed
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile --offline", ""); blocked {
		t.Fatal("expected --* glob to allow --offline")
	}
}

func TestCheckShellCommand_ToolRule_GlobMatch_MissAsks(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:          "tg2",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsInclude: []string{"compile", "-D*"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	// "-Pfoo" does not match any pattern → ask (not hard block)
	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn compile -Pfoo", "")
	if !blocked {
		t.Fatal("expected blocked for arg not matching any glob")
	}
	if !needsConfirm {
		t.Fatal("unmatched glob should ask, not hard-block (even when action=block)")
	}
}

func TestCheckShellCommand_ToolRule_ExcludeBeatsInclude(t *testing.T) {
	// Even if an arg is in args_include, args_exclude takes precedence.
	rule := guardrails.ToolRule{
		ID:          "t5",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsInclude: []string{"deploy"},
		ArgsExclude: []string{"deploy"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn deploy", "")
	if !blocked || needsConfirm {
		t.Fatal("args_exclude must hard-deny before args_include whitelist is checked")
	}
}

// --------------------------------------------------------------------------
// CheckShellCommand — allowed_directories with merge strategy
// --------------------------------------------------------------------------

func TestCheckShellCommand_AllowedDirs_Merge(t *testing.T) {
	globalDir := "/global/allowed"
	ruleDir := "/rule/allowed"

	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.AllowedDirectories.Enabled = true
	policy.DefaultPolicy.AllowedDirectories.Linux = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Mac = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Windows = []string{globalDir}

	rule := guardrails.ToolRule{
		ID:   "t6",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		AllowedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{ruleDir},
			Mac:     []string{ruleDir},
			Windows: []string{ruleDir},
		},

		MergeStrategy: guardrails.MergeStrategy{AllowedDirectories: "merge"},
	}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{rule}

	cleanup := writePolicy(t, policy)
	defer cleanup()

	// Both global and rule dirs should be allowed.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", globalDir); blocked {
		t.Fatal("global dir should be allowed (merge)")
	}
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", ruleDir); blocked {
		t.Fatal("rule dir should be allowed (merge)")
	}
	// A dir outside both should trigger ask.
	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn compile", "/other")
	if !blocked || !needsConfirm {
		t.Fatal("dir outside merge set should trigger ask")
	}
}

func TestCheckShellCommand_AllowedDirs_Override(t *testing.T) {
	globalDir := "/global/allowed"
	ruleDir := "/rule/only"

	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.AllowedDirectories.Enabled = true
	policy.DefaultPolicy.AllowedDirectories.Linux = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Mac = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Windows = []string{globalDir}

	rule := guardrails.ToolRule{
		ID:   "t7",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		AllowedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{ruleDir},
			Mac:     []string{ruleDir},
			Windows: []string{ruleDir},
		},

		MergeStrategy: guardrails.MergeStrategy{AllowedDirectories: "override"},
	}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{rule}

	cleanup := writePolicy(t, policy)
	defer cleanup()

	// Global dir is no longer in the effective set (override).
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", ruleDir); blocked {
		t.Fatal("rule dir should be allowed (override)")
	}
	blocked, _, _ := guardrails.CheckShellCommand("mvn compile", globalDir)
	if !blocked {
		t.Fatal("global dir should be blocked (override replaces global list)")
	}
}

func TestCheckShellCommand_AllowedDirs_Default(t *testing.T) {
	globalDir := "/global/allowed"
	ruleDir := "/rule/ignored"

	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.AllowedDirectories.Enabled = true
	policy.DefaultPolicy.AllowedDirectories.Linux = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Mac = []string{globalDir}
	policy.DefaultPolicy.AllowedDirectories.Windows = []string{globalDir}

	rule := guardrails.ToolRule{
		ID:   "t8",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		AllowedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{ruleDir},
			Mac:     []string{ruleDir},
			Windows: []string{ruleDir},
		},

		MergeStrategy: guardrails.MergeStrategy{AllowedDirectories: "default"},
	}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{rule}

	cleanup := writePolicy(t, policy)
	defer cleanup()

	// Only global dir allowed.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", globalDir); blocked {
		t.Fatal("global dir should be allowed (default strategy)")
	}
	// Rule dir is ignored.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", ruleDir); !blocked {
		t.Fatal("rule dir should not be allowed when strategy=default")
	}
}

// --------------------------------------------------------------------------
// LoadRestrictedPaths
// --------------------------------------------------------------------------

func TestLoadRestrictedPaths_Nil(t *testing.T) {
	dir := t.TempDir()
	defer setHomeDir(dir)()

	files, dirs := guardrails.LoadRestrictedPaths()
	if files != nil || dirs != nil {
		t.Fatal("expected nil for missing policy")
	}
}

func TestLoadRestrictedPaths_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = false
	policy.DefaultPolicy.RestrictedDirectories.Enabled = false
	cleanup := writePolicy(t, policy)
	defer cleanup()

	files, dirs := guardrails.LoadRestrictedPaths()
	if len(files) != 0 || len(dirs) != 0 {
		t.Fatal("expected empty lists when disabled")
	}
}

func TestLoadRestrictedPaths_Enabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = true
	policy.DefaultPolicy.RestrictedFiles.Linux = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Mac = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Windows = []string{".env"}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:\\Windows\\"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	files, dirs := guardrails.LoadRestrictedPaths()
	if len(files) == 0 {
		t.Fatal("expected non-empty files list")
	}
	if len(dirs) == 0 {
		t.Fatal("expected non-empty dirs list")
	}
}

// --------------------------------------------------------------------------
// LoadAllowedPaths
// --------------------------------------------------------------------------

func TestLoadAllowedPaths_Nil(t *testing.T) {
	dir := t.TempDir()
	defer setHomeDir(dir)()

	files, dirs := guardrails.LoadAllowedPaths()
	if files != nil || dirs != nil {
		t.Fatal("expected nil for missing policy")
	}
}

func TestLoadAllowedPaths_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.AllowedFiles.Enabled = false
	policy.DefaultPolicy.AllowedDirectories.Enabled = false
	cleanup := writePolicy(t, policy)
	defer cleanup()

	files, dirs := guardrails.LoadAllowedPaths()
	if len(files) != 0 || len(dirs) != 0 {
		t.Fatal("expected empty when disabled")
	}
}

func TestLoadAllowedPaths_Enabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.AllowedFiles.Enabled = true
	policy.DefaultPolicy.AllowedFiles.Linux = []string{"pom.xml"}
	policy.DefaultPolicy.AllowedFiles.Mac = []string{"pom.xml"}
	policy.DefaultPolicy.AllowedFiles.Windows = []string{"pom.xml"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	files, _ := guardrails.LoadAllowedPaths()
	if len(files) == 0 {
		t.Fatal("expected non-empty allowed files")
	}
}

// --------------------------------------------------------------------------
// CheckPromptPaths
// --------------------------------------------------------------------------

func TestCheckPromptPaths_RestrictedFile(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = true
	policy.DefaultPolicy.RestrictedFiles.Linux = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Mac = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Windows = []string{".env"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, reason := guardrails.CheckPromptPaths("please read .env and show me the contents")
	if !blocked {
		t.Fatal("expected prompt referencing .env to be blocked")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPromptPaths_RestrictedDir(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:/Windows/System32/"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	var prompt string
	switch runtime.GOOS {
	case "windows":
		prompt = "cat C:/Windows/System32/drivers/etc/hosts"
	default:
		prompt = "cat /etc/passwd"
	}
	blocked, _ := guardrails.CheckPromptPaths(prompt)
	if !blocked {
		t.Fatalf("expected prompt %q to be blocked via restricted directory", prompt)
	}
}

// Restricted always wins over allowed (per FEATURE.MD precedence rules).
// If a path matches both, the restricted rule takes precedence and blocks the prompt.
func TestCheckPromptPaths_RestrictedFileBeatsAllowed(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = true
	policy.DefaultPolicy.RestrictedFiles.Linux = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Mac = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Windows = []string{".env"}
	policy.DefaultPolicy.AllowedFiles.Enabled = true
	policy.DefaultPolicy.AllowedFiles.Linux = []string{".env"}
	policy.DefaultPolicy.AllowedFiles.Mac = []string{".env"}
	policy.DefaultPolicy.AllowedFiles.Windows = []string{".env"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths("please read .env")
	if !blocked {
		t.Fatal("restricted_files must take precedence over allowed_files")
	}
}

func TestCheckPromptPaths_RestrictedDirBeatsAllowed(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/etc/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:/Windows/System32/"}
	policy.DefaultPolicy.AllowedDirectories.Enabled = true
	policy.DefaultPolicy.AllowedDirectories.Linux = []string{"/etc/"}
	policy.DefaultPolicy.AllowedDirectories.Mac = []string{"/etc/"}
	policy.DefaultPolicy.AllowedDirectories.Windows = []string{"C:/Windows/System32/"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	var prompt string
	switch runtime.GOOS {
	case "windows":
		prompt = "cat C:/Windows/System32/drivers/etc/hosts"
	default:
		prompt = "cat /etc/passwd"
	}
	blocked, _ := guardrails.CheckPromptPaths(prompt)
	if !blocked {
		t.Fatal("restricted_directories must take precedence over allowed_directories")
	}
}

func TestCheckPromptPaths_CleanPrompt(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = true
	policy.DefaultPolicy.RestrictedFiles.Linux = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Mac = []string{".env"}
	policy.DefaultPolicy.RestrictedFiles.Windows = []string{".env"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths("show me the main.go file")
	if blocked {
		t.Fatal("clean prompt should not be blocked")
	}
}

// --------------------------------------------------------------------------
// Glob pattern support (doublestar)
// --------------------------------------------------------------------------

// restrictedFilesPolicy is a helper building a HooksPolicy whose restricted_files list
// applies on every OS — avoids repeating the boilerplate across glob tests.
func restrictedFilesPolicy(patterns []string) guardrails.HooksPolicy {
	p := guardrails.HooksPolicy{}
	p.DefaultPolicy.RestrictedFiles.Enabled = true
	p.DefaultPolicy.RestrictedFiles.Linux = patterns
	p.DefaultPolicy.RestrictedFiles.Mac = patterns
	p.DefaultPolicy.RestrictedFiles.Windows = patterns
	return p
}

// restrictedDirsPolicy mirrors restrictedFilesPolicy for restricted_directories.
func restrictedDirsPolicy(patterns []string) guardrails.HooksPolicy {
	p := guardrails.HooksPolicy{}
	p.DefaultPolicy.RestrictedDirectories.Enabled = true
	p.DefaultPolicy.RestrictedDirectories.Linux = patterns
	p.DefaultPolicy.RestrictedDirectories.Mac = patterns
	p.DefaultPolicy.RestrictedDirectories.Windows = patterns
	return p
}

func TestCheckPromptPaths_GlobBasename_StarDotPem(t *testing.T) {
	cleanup := writePolicy(t, restrictedFilesPolicy([]string{"*.pem"}))
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths("please read cert.pem for the deploy")
	if !blocked {
		t.Fatal("expected *.pem to match cert.pem via basename glob")
	}
}

func TestCheckPromptPaths_DoubleStar_AnywherePem(t *testing.T) {
	cleanup := writePolicy(t, restrictedFilesPolicy([]string{"**/*.pem"}))
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths("please inspect /srv/keys/cert.pem now")
	if !blocked {
		t.Fatal("expected **/*.pem to match /srv/keys/cert.pem")
	}
}

func TestCheckPromptPaths_GlobDir_PerUserSSH(t *testing.T) {
	var patterns []string
	var prompt string
	switch runtime.GOOS {
	case "windows":
		patterns = []string{"C:/Users/*/.ssh"}
		prompt = "grab C:/Users/alice/.ssh/id_rsa"
	default:
		patterns = []string{"/home/*/.ssh"}
		prompt = "grab /home/alice/.ssh/id_rsa"
	}
	cleanup := writePolicy(t, restrictedDirsPolicy(patterns))
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths(prompt)
	if !blocked {
		t.Fatalf("expected per-user .ssh glob to block %q", prompt)
	}
}

func TestCheckPromptPaths_DoubleStar_SecretsAnywhere(t *testing.T) {
	cleanup := writePolicy(t, restrictedDirsPolicy([]string{"**/secrets/**"}))
	defer cleanup()

	blocked, _ := guardrails.CheckPromptPaths("look at /repo/services/secrets/db.yaml please")
	if !blocked {
		t.Fatal("expected **/secrets/** to match a file inside any secrets dir")
	}
}

func TestCheckPromptPaths_LiteralBasename_StillWorks(t *testing.T) {
	cleanup := writePolicy(t, restrictedFilesPolicy([]string{"kubeconfig", "terraform.tfstate"}))
	defer cleanup()

	if blocked, _ := guardrails.CheckPromptPaths("merge /etc/kubeconfig please"); !blocked {
		t.Fatal("literal basename kubeconfig should still block")
	}
	if blocked, _ := guardrails.CheckPromptPaths("read terraform.tfstate for audit"); !blocked {
		t.Fatal("literal basename terraform.tfstate should still block")
	}
}

func TestPathUnderAny_GlobDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		if !guardrails.PathUnderAny("C:/Users/alice/.ssh/id_rsa", []string{"C:/Users/*/.ssh"}) {
			t.Fatal("expected glob dir to match nested path")
		}
		if guardrails.PathUnderAny("C:/Users/alice/Documents/report.txt", []string{"C:/Users/*/.ssh"}) {
			t.Fatal("glob dir must not match unrelated path")
		}
	default:
		if !guardrails.PathUnderAny("/home/alice/.ssh/id_rsa", []string{"/home/*/.ssh"}) {
			t.Fatal("expected glob dir to match nested path")
		}
		if guardrails.PathUnderAny("/home/alice/Documents/report.txt", []string{"/home/*/.ssh"}) {
			t.Fatal("glob dir must not match unrelated path")
		}
	}
}

func TestCheckShellCommand_RestrictedFilesGlob(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedFiles.Enabled = true
	policy.DefaultPolicy.RestrictedFiles.Linux = []string{"**/*.pem"}
	policy.DefaultPolicy.RestrictedFiles.Mac = []string{"**/*.pem"}
	policy.DefaultPolicy.RestrictedFiles.Windows = []string{"**/*.pem"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand("cat /tmp/secrets/foo.pem", "")
	if !blocked {
		t.Fatal("expected **/*.pem to block a command referencing the file")
	}
	if needsConfirm {
		t.Fatal("restricted-files match should be a hard block, not an ask")
	}
}

func TestCheckShellCommand_AllowedFilesGlob(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:   "glob-allow",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		AllowedFiles: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{"**/pom.xml", "*.java"},
			Mac:     []string{"**/pom.xml", "*.java"},
			Windows: []string{"**/pom.xml", "*.java"},
		},
		MergeStrategy: guardrails.MergeStrategy{AllowedFiles: "override"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	// Foo.java matches "*.java" via basename glob — should be allowed.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile Foo.java", ""); blocked {
		t.Fatal("expected *.java glob to allow Foo.java")
	}
	// ./sub/pom.xml matches "**/pom.xml" via full-path glob — should be allowed.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn -f ./sub/pom.xml compile", ""); blocked {
		t.Fatal("expected **/pom.xml glob to allow ./sub/pom.xml")
	}
	// script.sh matches nothing — should ask.
	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn compile script.sh", "")
	if !blocked || !needsConfirm {
		t.Fatal("unknown file must trigger ask, not allow or hard-block")
	}
}

func TestResolveRestrictedPaths_MergeWithEmptyRule(t *testing.T) {
	// Empty rule list + merge strategy must safely fall back to the global list.
	got := guardrails.ResolveRestrictedPaths([]string{"/a", "/b"}, nil, "merge")
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("empty rule + merge should return global list verbatim, got %v", got)
	}
}

// --------------------------------------------------------------------------
// CheckWorkspaceRoots
// --------------------------------------------------------------------------

func TestCheckWorkspaceRoots_Blocked(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:\\Cx-Flow\\"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	var roots []string
	switch runtime.GOOS {
	case "windows":
		// Cursor reports Windows roots with a leading slash before the drive letter.
		roots = []string{"/c:/Cx-Flow/Test/JavaVulnerabilityLabE"}
	default:
		roots = []string{"/restricted/project"}
	}

	blocked, reason := guardrails.CheckWorkspaceRoots(roots)
	if !blocked {
		t.Fatalf("expected workspace %v to be blocked", roots)
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckWorkspaceRoots_Allowed(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:\\Cx-Flow\\"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	var roots []string
	switch runtime.GOOS {
	case "windows":
		roots = []string{"/d:/Projects/safe"}
	default:
		roots = []string{"/home/user/safe"}
	}

	blocked, _ := guardrails.CheckWorkspaceRoots(roots)
	if blocked {
		t.Fatalf("expected workspace %v to be allowed", roots)
	}
}

func TestCheckWorkspaceRoots_EmptyList(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{"/restricted/"}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{"C:\\Cx-Flow\\"}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, _ := guardrails.CheckWorkspaceRoots(nil)
	if blocked {
		t.Fatal("empty workspace root list must not block")
	}
}

// --------------------------------------------------------------------------
// NormalizeWorkspaceRoot
// --------------------------------------------------------------------------

func TestNormalizeWorkspaceRoot(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"cursor-windows-leading-slash", "/c:/Cx-Flow/Test", "c:/Cx-Flow/Test"},
		{"already-normalized-windows", "C:/Cx-Flow/Test", "C:/Cx-Flow/Test"},
		{"windows-backslashes", "C:\\Cx-Flow\\Test", "C:/Cx-Flow/Test"},
		{"unix-absolute", "/etc/secrets", "/etc/secrets"},
		{"empty", "", ""},
		{"slash-only", "/", "/"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardrails.NormalizeWorkspaceRoot(tc.in); got != tc.want {
				t.Fatalf("NormalizeWorkspaceRoot(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// FindMatchingToolRule
// --------------------------------------------------------------------------

func TestFindMatchingToolRule_Match(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r1", Tool: []string{"mvn", "mvnw"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	rule := guardrails.FindMatchingToolRule("mvn clean")
	if rule == nil {
		t.Fatal("expected matching rule for mvn")
	}
	if rule.ID != "r1" {
		t.Fatalf("expected rule r1, got %q", rule.ID)
	}
}

func TestFindMatchingToolRule_AltName(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r2", Tool: []string{"mvn", "mvnw"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	rule := guardrails.FindMatchingToolRule("mvnw compile")
	if rule == nil || rule.ID != "r2" {
		t.Fatal("expected match on alternate tool name mvnw")
	}
}

func TestFindMatchingToolRule_OSMismatch(t *testing.T) {
	wrongOS := "linux"
	if runtime.GOOS == "linux" {
		wrongOS = "windows"
	}
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r3", Tool: []string{"mvn"}, OS: []string{wrongOS}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn compile"); rule != nil {
		t.Fatal("should not match rule for wrong OS")
	}
}

func TestFindMatchingToolRule_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = false
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r4", Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn compile"); rule != nil {
		t.Fatal("should not match rule when tools section is disabled")
	}
}

func TestFindMatchingToolRule_NoPolicy(t *testing.T) {
	dir := t.TempDir()
	defer setHomeDir(dir)()

	if rule := guardrails.FindMatchingToolRule("mvn compile"); rule != nil {
		t.Fatal("should return nil when no policy file exists")
	}
}

// --------------------------------------------------------------------------
// ResolveAllowedPaths
// --------------------------------------------------------------------------

func TestResolveAllowedPaths_Merge(t *testing.T) {
	global := []string{"/a", "/b"}
	rule := []string{"/c"}
	got := guardrails.ResolveAllowedPaths(global, rule, "merge")
	want := map[string]bool{"/a": true, "/b": true, "/c": true}
	if len(got) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(got), got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected path %q in merge result", p)
		}
	}
}

func TestResolveAllowedPaths_Merge_Deduplicates(t *testing.T) {
	global := []string{"/a", "/b"}
	rule := []string{"/b", "/c"}
	got := guardrails.ResolveAllowedPaths(global, rule, "merge")
	if len(got) != 3 {
		t.Fatalf("expected 3 deduplicated paths, got %d: %v", len(got), got)
	}
}

func TestResolveAllowedPaths_Override(t *testing.T) {
	global := []string{"/a", "/b"}
	rule := []string{"/c"}
	got := guardrails.ResolveAllowedPaths(global, rule, "override")
	if len(got) != 1 || got[0] != "/c" {
		t.Fatalf("expected only rule paths on override, got %v", got)
	}
}

func TestResolveAllowedPaths_Default(t *testing.T) {
	global := []string{"/a", "/b"}
	rule := []string{"/c"}
	got := guardrails.ResolveAllowedPaths(global, rule, "default")
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("expected only global paths on default, got %v", got)
	}
}

func TestResolveAllowedPaths_UnknownStrategyActsAsDefault(t *testing.T) {
	global := []string{"/a"}
	rule := []string{"/b"}
	got := guardrails.ResolveAllowedPaths(global, rule, "unknown-strategy")
	if len(got) != 1 || got[0] != "/a" {
		t.Fatalf("unknown strategy should act as default, got %v", got)
	}
}

// --------------------------------------------------------------------------
// ResolveRestrictedPaths (delegates to same logic as allowed)
// --------------------------------------------------------------------------

func TestResolveRestrictedPaths_Merge(t *testing.T) {
	got := guardrails.ResolveRestrictedPaths([]string{"/a"}, []string{"/b"}, "merge")
	if len(got) != 2 {
		t.Fatalf("expected 2 merged paths, got %v", got)
	}
}

func TestResolveRestrictedPaths_Override(t *testing.T) {
	got := guardrails.ResolveRestrictedPaths([]string{"/a"}, []string{"/b"}, "override")
	if len(got) != 1 || got[0] != "/b" {
		t.Fatalf("expected override to yield rule paths only, got %v", got)
	}
}

func TestResolveRestrictedPaths_Default(t *testing.T) {
	got := guardrails.ResolveRestrictedPaths([]string{"/a"}, []string{"/b"}, "default")
	if len(got) != 1 || got[0] != "/a" {
		t.Fatalf("expected default to yield global paths only, got %v", got)
	}
}

// --------------------------------------------------------------------------
// Tool-level restricted paths (shell.go)
// --------------------------------------------------------------------------

// When ask_on_restricted=false (default), tool-level restricted paths hard-block.
func TestCheckShellCommand_ToolRestrictedDir_HardBlock(t *testing.T) {
	restricted := "/prod"
	rule := guardrails.ToolRule{
		ID:   "trd1",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		RestrictedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{restricted},
			Mac:     []string{restricted},
			Windows: []string{restricted},
		},
		MergeStrategy: guardrails.MergeStrategy{RestrictedDirectories: "override"},
	}
	policy := makeToolRulePolicy(rule)
	cleanup := writePolicy(t, policy)
	defer cleanup()

	blocked, needsConfirm, reason := guardrails.CheckShellCommand("mvn compile", restricted)
	if !blocked {
		t.Fatal("expected blocked for restricted workDir")
	}
	if needsConfirm {
		t.Fatal("expected hard block when ask_on_restricted=false")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

// Restricted paths always hard-block regardless of any flag — needsConfirm is never true.
func TestCheckShellCommand_ToolRestrictedDir_AlwaysHardBlock(t *testing.T) {
	restricted := "/prod"
	rule := guardrails.ToolRule{
		ID:   "trd2",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		RestrictedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{restricted},
			Mac:     []string{restricted},
			Windows: []string{restricted},
		},
		MergeStrategy: guardrails.MergeStrategy{RestrictedDirectories: "override"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn compile", restricted)
	if !blocked {
		t.Fatal("expected blocked=true for restricted workDir")
	}
	if needsConfirm {
		t.Fatal("restricted paths must always hard-block (needsConfirm=false)")
	}
}

func TestCheckShellCommand_ToolRestrictedFile_HardBlock(t *testing.T) {
	rule := guardrails.ToolRule{
		ID:   "trf1",
		Tool: []string{"cat"},
		OS:   []string{currentOS()},
		RestrictedFiles: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{"secret.key"},
			Mac:     []string{"secret.key"},
			Windows: []string{"secret.key"},
		},
		MergeStrategy: guardrails.MergeStrategy{RestrictedFiles: "override"},
	}
	cleanup := writePolicy(t, makeToolRulePolicy(rule))
	defer cleanup()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand("cat ./secret.key", "")
	if !blocked || needsConfirm {
		t.Fatal("expected hard block for restricted file arg")
	}
}

// Tool-level restricted paths merge with the global list when strategy=merge.
func TestCheckShellCommand_ToolRestrictedDir_MergeStrategy(t *testing.T) {
	globalRestricted := "/global-prod"
	ruleRestricted := "/rule-prod"

	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.RestrictedDirectories.Enabled = true
	policy.DefaultPolicy.RestrictedDirectories.Linux = []string{globalRestricted}
	policy.DefaultPolicy.RestrictedDirectories.Mac = []string{globalRestricted}
	policy.DefaultPolicy.RestrictedDirectories.Windows = []string{globalRestricted}

	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{{
		ID:   "merge-r",
		Tool: []string{"mvn"},
		OS:   []string{currentOS()},
		RestrictedDirectories: guardrails.PathPolicy{
			Enabled: true,
			Linux:   []string{ruleRestricted},
			Mac:     []string{ruleRestricted},
			Windows: []string{ruleRestricted},
		},
		MergeStrategy: guardrails.MergeStrategy{RestrictedDirectories: "merge"},
	}}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	// Both global and rule restricted dirs should block.
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", globalRestricted); !blocked {
		t.Fatal("global restricted dir should block under merge strategy")
	}
	if blocked, _, _ := guardrails.CheckShellCommand("mvn compile", ruleRestricted); !blocked {
		t.Fatal("rule restricted dir should block under merge strategy")
	}
}

// --------------------------------------------------------------------------
// ToolRule.Enabled (pointer-based opt-out)
// --------------------------------------------------------------------------

func TestFindMatchingToolRule_ExplicitlyDisabled(t *testing.T) {
	disabled := false
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "off", Enabled: &disabled, Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn compile"); rule != nil {
		t.Fatal("rule with enabled=false should be skipped")
	}
}

func TestFindMatchingToolRule_ExplicitlyEnabled(t *testing.T) {
	enabled := true
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "on", Enabled: &enabled, Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn compile"); rule == nil || rule.ID != "on" {
		t.Fatal("rule with enabled=true should match")
	}
}

// --------------------------------------------------------------------------
// BlastRadiusLimit
// --------------------------------------------------------------------------

func TestBlastRadiusLimit_BlocksAfterThreshold(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlastRadiusLimit = guardrails.BlastRadiusLimit{
		Enabled:   true,
		Threshold: 2,
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	guardrails.ResetBlastRadiusCount()

	// First two writes are allowed.
	if blocked, _ := guardrails.CheckAndIncrementBlastRadius(); blocked {
		t.Fatal("first write should be allowed")
	}
	if blocked, _ := guardrails.CheckAndIncrementBlastRadius(); blocked {
		t.Fatal("second write should be allowed")
	}
	// Third write exceeds the threshold.
	blocked, reason := guardrails.CheckAndIncrementBlastRadius()
	if !blocked {
		t.Fatal("third write should be blocked by blast radius limit")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestBlastRadiusLimit_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlastRadiusLimit = guardrails.BlastRadiusLimit{
		Enabled:   false,
		Threshold: 1,
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	guardrails.ResetBlastRadiusCount()
	for i := 0; i < 10; i++ {
		if blocked, _ := guardrails.CheckAndIncrementBlastRadius(); blocked {
			t.Fatalf("write %d should be allowed when disabled", i+1)
		}
	}
}

// --------------------------------------------------------------------------
// MaxTotalFileSizeKB
// --------------------------------------------------------------------------

func TestMaxTotalFileSizeKB_BlocksAfterThreshold(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = guardrails.FilesLimits{
		Enabled:            true,
		MaxTotalFileSizeKB: 1, // 1 KB = 1024 bytes
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	guardrails.ResetTotalFileSizeCount()

	// First 512 bytes allowed.
	if blocked, _ := guardrails.CheckAndIncrementTotalFileSize(512); blocked {
		t.Fatal("first write (512 B) should be allowed")
	}
	// Second 512 bytes brings total to exactly 1024 — still within limit.
	if blocked, _ := guardrails.CheckAndIncrementTotalFileSize(512); blocked {
		t.Fatal("second write (512 B, total 1024 B) should be allowed")
	}
	// One more byte pushes total over 1024.
	blocked, reason := guardrails.CheckAndIncrementTotalFileSize(1)
	if !blocked {
		t.Fatal("write exceeding max_total_file_size_kb should be blocked")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestMaxTotalFileSizeKB_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = guardrails.FilesLimits{
		Enabled:            false,
		MaxTotalFileSizeKB: 1,
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	guardrails.ResetTotalFileSizeCount()
	for i := 0; i < 10; i++ {
		if blocked, _ := guardrails.CheckAndIncrementTotalFileSize(1024 * 1024); blocked {
			t.Fatalf("write %d should be allowed when disabled", i+1)
		}
	}
}

// --------------------------------------------------------------------------
// BlockedExtensions
// --------------------------------------------------------------------------

func TestCheckBlockedExtensions_Blocks(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.BlockedExtensions = guardrails.BlockedExtensions{
		Enabled:    true,
		Extensions: []string{".env", ".pem"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.CheckBlockedExtensions("please read secrets.pem"); reason == "" {
		t.Fatal("expected .pem reference to be blocked")
	}
}

func TestCheckBlockedExtensions_Clean(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.BlockedExtensions = guardrails.BlockedExtensions{
		Enabled:    true,
		Extensions: []string{".env"},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.CheckBlockedExtensions("please read main.go"); reason != "" {
		t.Fatal("clean prompt should not be blocked")
	}
}

func TestCheckBlockedExtensions_DisabledNoPolicy(t *testing.T) {
	dir := t.TempDir()
	defer setHomeDir(dir)()

	if reason := guardrails.CheckBlockedExtensions("please read secrets.pem"); reason != "" {
		t.Fatal("should not block when no policy is configured")
	}
}

// --------------------------------------------------------------------------
// FilesLimits
// --------------------------------------------------------------------------

func TestCheckFilesLimits_Blocks(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = guardrails.FilesLimits{
		Enabled:      true,
		MaxFileCount: 2,
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	prompt := "read a.go, b.go, and c.go"
	if reason := guardrails.CheckFilesLimits(prompt); reason == "" {
		t.Fatal("expected prompt referencing 3 files to exceed max_file_count=2")
	}
}

func TestCheckFilesLimits_UnderLimit(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = guardrails.FilesLimits{
		Enabled:      true,
		MaxFileCount: 5,
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.CheckFilesLimits("read a.go and b.go"); reason != "" {
		t.Fatal("prompt under the limit should not be blocked")
	}
}

// --------------------------------------------------------------------------
// ScanForPolicyPatterns
// --------------------------------------------------------------------------

func TestScanForPolicyPatterns_Matches(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.ContentScanning = guardrails.ContentScanning{
		Enabled: true,
		Patterns: []guardrails.ContentScanPattern{
			{ID: "no-prod", Pattern: `prod\.example\.com`, Description: "Production URLs"},
		},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.ScanForPolicyPatterns("see https://prod.example.com/api"); reason == "" {
		t.Fatal("expected prompt matching policy pattern to be rejected")
	}
}

func TestScanForPolicyPatterns_NoMatch(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.ContentScanning = guardrails.ContentScanning{
		Enabled: true,
		Patterns: []guardrails.ContentScanPattern{
			{ID: "no-prod", Pattern: `prod\.example\.com`, Description: "Production URLs"},
		},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.ScanForPolicyPatterns("see https://staging.example.com"); reason != "" {
		t.Fatal("clean prompt should not match")
	}
}

func TestScanForPolicyPatterns_Disabled(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.ContentScanning = guardrails.ContentScanning{
		Enabled: false,
		Patterns: []guardrails.ContentScanPattern{
			{ID: "no-prod", Pattern: `prod`, Description: "Production"},
		},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if reason := guardrails.ScanForPolicyPatterns("prod is a keyword"); reason != "" {
		t.Fatal("disabled scanner should not produce findings")
	}
}

// --------------------------------------------------------------------------
// FindMatchingToolRule — compound command / token-boundary behavior (Fix B)
// --------------------------------------------------------------------------

func TestFindMatchingToolRule_TokenInChain(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r-chain", Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	rule := guardrails.FindMatchingToolRule(`cd "c:\foo" && mvn deploy`)
	if rule == nil || rule.ID != "r-chain" {
		t.Fatalf("expected r-chain match for chained mvn, got %+v", rule)
	}
}

func TestFindMatchingToolRule_TokenAtStart(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r-start", Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn package"); rule == nil || rule.ID != "r-start" {
		t.Fatalf("regression: expected r-start match for plain `mvn package`, got %+v", rule)
	}
}

func TestFindMatchingToolRule_TokenNotSubstringMatch(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r-nosub", Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("cd /opt/foo-mvn-bar/ls"); rule != nil {
		t.Fatalf("expected nil for `mvn` substring inside path token, got %+v", rule)
	}
}

func TestFindMatchingToolRule_TokenBoundaryUnderscore(t *testing.T) {
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{ID: "r-underscore", Tool: []string{"mvn"}, OS: []string{currentOS()}},
	}
	cleanup := writePolicy(t, policy)
	defer cleanup()

	if rule := guardrails.FindMatchingToolRule("mvn_helper run"); rule != nil {
		t.Fatalf("expected nil for `mvn_helper` (underscore is a word byte), got %+v", rule)
	}
}

// --------------------------------------------------------------------------
// CheckShellCommand — compound + parent-command args_exclude (Fix B)
// --------------------------------------------------------------------------

func mvnDeployExcludeFixture(t *testing.T) func() {
	t.Helper()
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{
		{
			ID:          "mvn-deploy-excluded",
			Tool:        []string{"mvn"},
			OS:          []string{currentOS()},
			ArgsExclude: []string{"deploy"},
		},
	}
	return writePolicy(t, policy)
}

func TestCheckShellCommand_Compound_ArgsExcludeAfterCd(t *testing.T) {
	defer mvnDeployExcludeFixture(t)()

	blocked, needsConfirm, reason := guardrails.CheckShellCommand(`cd "c:\foo" && mvn deploy`, "")
	if !blocked || needsConfirm {
		t.Fatalf("expected hard deny on chained `mvn deploy`, got blocked=%v needsConfirm=%v reason=%q",
			blocked, needsConfirm, reason)
	}
	if !strings.Contains(reason, "deploy") {
		t.Fatalf("expected reason to cite `deploy`, got %q", reason)
	}
}

func TestCheckShellCommand_Compound_ArgsExcludeWithExtraFlags(t *testing.T) {
	defer mvnDeployExcludeFixture(t)()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand(`cd "c:\foo" && mvn deploy -Djava=11`, "")
	if !blocked || needsConfirm {
		t.Fatalf("expected hard deny on chained `mvn deploy -Djava=11`, got blocked=%v needsConfirm=%v",
			blocked, needsConfirm)
	}
}

func TestCheckShellCommand_SingleCmd_ArgsExcludeWithExtraFlags(t *testing.T) {
	defer mvnDeployExcludeFixture(t)()

	blocked, needsConfirm, _ := guardrails.CheckShellCommand("mvn deploy -Djava=11", "")
	if !blocked || needsConfirm {
		t.Fatalf("expected hard deny on `mvn deploy -Djava=11`, got blocked=%v needsConfirm=%v",
			blocked, needsConfirm)
	}
}

func TestCheckShellCommand_DenyMessageAppended(t *testing.T) {
	defer mvnDeployExcludeFixture(t)()

	_, _, reason := guardrails.CheckShellCommand("mvn deploy", "")
	if !strings.Contains(reason, "Do NOT attempt alternative commands") {
		t.Fatalf("expected DenyMessage no-workaround text in reason, got %q", reason)
	}
}
