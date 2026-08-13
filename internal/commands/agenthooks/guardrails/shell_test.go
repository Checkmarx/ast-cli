//go:build !integration

package guardrails

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func shellTestOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func setOSPaths(pp *PathPolicy, paths []string) {
	pp.Enabled = true
	switch runtime.GOOS {
	case "darwin":
		pp.Mac = paths
	case "windows":
		pp.Windows = paths
	default:
		pp.Linux = paths
	}
}

// --------------------------------------------------------------------------
// CheckShellCommand — end-to-end scenarios for shell.go
// --------------------------------------------------------------------------

func TestCheckShellCommand_EmptyCommand_Allows(t *testing.T) {
	defer writePolicyHelper(t, &HooksPolicy{})()
	blocked, needsConfirm, reason := CheckShellCommand("", "")
	if blocked || needsConfirm || reason != "" {
		t.Fatalf("empty command should allow, got blocked=%v confirm=%v reason=%q", blocked, needsConfirm, reason)
	}
}

func TestCheckShellCommand_NoPolicy_Allows(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		orig, had := os.LookupEnv("USERPROFILE")
		os.Setenv("USERPROFILE", dir)
		defer func() {
			if had {
				os.Setenv("USERPROFILE", orig)
			} else {
				os.Unsetenv("USERPROFILE")
			}
		}()
	} else {
		orig, had := os.LookupEnv("HOME")
		os.Setenv("HOME", dir)
		defer func() {
			if had {
				os.Setenv("HOME", orig)
			} else {
				os.Unsetenv("HOME")
			}
		}()
	}
	blocked, _, _ := CheckShellCommand("ls -la", dir)
	if blocked {
		t.Fatal("missing policy should fail-open")
	}
}

func TestCheckShellCommand_Blacklist_HardBlock(t *testing.T) {
	policy := HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []BlacklistedTool{
		{Name: "rm -rf", OS: []string{shellTestOS()}, Category: "destructive", Risk: "wipes files"},
	}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("sudo rm -rf /tmp/x", "")
	if !blocked || needsConfirm {
		t.Fatalf("blacklist should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "rm -rf") || !strings.Contains(reason, "destructive") {
		t.Errorf("reason should cite blacklist entry, got %q", reason)
	}
	if !strings.Contains(reason, DenyMessage) {
		t.Errorf("reason should append DenyMessage, got %q", reason)
	}
}

func TestCheckShellCommand_Blacklist_CaseInsensitive(t *testing.T) {
	policy := HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []BlacklistedTool{
		{Name: "FORMAT", OS: []string{shellTestOS()}, Category: "destructive", Risk: "wipe disk"},
	}
	defer writePolicyHelper(t, &policy)()

	blocked, _, _ := CheckShellCommand("format C:", "")
	if !blocked {
		t.Fatal("blacklist match should be case-insensitive")
	}
}

func TestCheckShellCommand_ArgsExclude_HardBlock(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []ToolRule{{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		ArgsExclude: []string{"deploy"},
	}}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("mvn clean deploy", "/proj")
	if !blocked || needsConfirm {
		t.Fatalf("args_exclude should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "deploy") {
		t.Errorf("reason should cite excluded arg, got %q", reason)
	}
}

func TestCheckShellCommand_ArgsInclude_UnknownAsks(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []ToolRule{{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		ArgsInclude: []string{"compile", "test"},
	}}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("mvn package", "")
	if !blocked || !needsConfirm {
		t.Fatalf("unknown arg should ask, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "package") {
		t.Errorf("reason should cite unknown arg, got %q", reason)
	}
}

func TestCheckShellCommand_ArgsInclude_CommandNameOnly_Allows(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []ToolRule{{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		ArgsInclude: []string{"compile"},
	}}
	defer writePolicyHelper(t, &policy)()

	// tokens[1:] empty — include whitelist is skipped.
	blocked, needsConfirm, _ := CheckShellCommand("mvn", "")
	if blocked || needsConfirm {
		t.Fatal("command with no args should not trip args_include")
	}
}

func TestCheckShellCommand_ArgsInclude_Allowed_Passes(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []ToolRule{{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		ArgsInclude: []string{"compile", "-D*"},
	}}
	defer writePolicyHelper(t, &policy)()

	blocked, _, _ := CheckShellCommand("mvn compile -DskipTests", "")
	if blocked {
		t.Fatal("allowed args (exact + glob) should pass")
	}
}

func TestCheckShellCommand_ExcludeBeatsInclude(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []ToolRule{{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		ArgsInclude: []string{"deploy"},
		ArgsExclude: []string{"deploy"},
	}}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, _ := CheckShellCommand("mvn deploy", "")
	if !blocked || needsConfirm {
		t.Fatal("exclude must hard-block even when also in include")
	}
}

func TestCheckShellCommand_GlobalRestrictedDir_NoToolRule(t *testing.T) {
	restricted := filepath.Join(t.TempDir(), "secrets")
	policy := HooksPolicy{}
	setOSPaths(&policy.DefaultPolicy.RestrictedDirectories, []string{restricted})
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("ls", restricted)
	if !blocked || needsConfirm {
		t.Fatalf("global restricted dir should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "restricted by policy") {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckShellCommand_GlobalRestrictedFile_PathShaped(t *testing.T) {
	policy := HooksPolicy{}
	setOSPaths(&policy.DefaultPolicy.RestrictedFiles, []string{"**/*.pem"})
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("cat /tmp/secrets/foo.pem", "")
	if !blocked || needsConfirm {
		t.Fatalf("restricted glob file should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "foo.pem") {
		t.Errorf("reason should cite file token, got %q", reason)
	}
}

func TestCheckShellCommand_GlobalRestrictedFile_BareWordLiteral(t *testing.T) {
	policy := HooksPolicy{}
	setOSPaths(&policy.DefaultPolicy.RestrictedFiles, []string{"kubeconfig"})
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("cat kubeconfig", "")
	if !blocked || needsConfirm {
		t.Fatalf("bare-word restricted file should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "kubeconfig") {
		t.Errorf("reason should cite kubeconfig, got %q", reason)
	}
}

func TestCheckShellCommand_ToolRestrictedDir_HardBlock(t *testing.T) {
	restricted := filepath.Join(t.TempDir(), "prod")
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{RestrictedDirectories: "override"},
	}
	setOSPaths(&rule.RestrictedDirectories, []string{restricted})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("mvn compile", restricted)
	if !blocked || needsConfirm {
		t.Fatalf("tool restricted dir should hard-block, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "not permitted for this tool") {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckShellCommand_ToolRestrictedFile_HardBlock(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "cat", Tool: []string{"cat"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{RestrictedFiles: "override"},
	}
	setOSPaths(&rule.RestrictedFiles, []string{"*.key"})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, _ := CheckShellCommand("cat ./secret.key", "")
	if !blocked || needsConfirm {
		t.Fatal("tool restricted file should hard-block")
	}
}

func TestCheckShellCommand_AllowedDir_OutsideAsks(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "ok")
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{AllowedDirectories: "override"},
	}
	setOSPaths(&rule.AllowedDirectories, []string{allowed})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("mvn compile", filepath.Join(t.TempDir(), "other"))
	if !blocked || !needsConfirm {
		t.Fatalf("workdir outside allowed dirs should ask, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "not in the allowed list") {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckShellCommand_AllowedDir_InsidePasses(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "ok")
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{AllowedDirectories: "override"},
	}
	setOSPaths(&rule.AllowedDirectories, []string{allowed})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, _, _ := CheckShellCommand("mvn compile", allowed)
	if blocked {
		t.Fatal("workdir inside allowed dirs should pass")
	}
}

func TestCheckShellCommand_AllowedFiles_UnknownAsks(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{AllowedFiles: "override"},
	}
	setOSPaths(&rule.AllowedFiles, []string{"*.java", "**/pom.xml"})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, needsConfirm, reason := CheckShellCommand("mvn compile script.sh", "")
	if !blocked || !needsConfirm {
		t.Fatalf("disallowed file arg should ask, blocked=%v confirm=%v", blocked, needsConfirm)
	}
	if !strings.Contains(reason, "script.sh") {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckShellCommand_AllowedFiles_NonFileTokenSkipped(t *testing.T) {
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{AllowedFiles: "override"},
	}
	setOSPaths(&rule.AllowedFiles, []string{"*.java"})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	// "compile" has no ./\\ so allowed-files check skips it.
	blocked, _, _ := CheckShellCommand("mvn compile", "")
	if blocked {
		t.Fatal("non-file tokens should be ignored by allowed_files")
	}
}

func TestCheckShellCommand_EmptyWorkDir_SkipsDirChecks(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "ok")
	policy := HooksPolicy{}
	policy.Tools.Enabled = true
	rule := ToolRule{
		ID: "mvn", Tool: []string{"mvn"}, OS: []string{shellTestOS()},
		MergeStrategy: MergeStrategy{AllowedDirectories: "override"},
	}
	setOSPaths(&rule.AllowedDirectories, []string{allowed})
	policy.Tools.Rules = []ToolRule{rule}
	defer writePolicyHelper(t, &policy)()

	blocked, _, _ := CheckShellCommand("mvn compile", "")
	if blocked {
		t.Fatal("empty workDir should skip allowed/restricted dir checks")
	}
}

// --------------------------------------------------------------------------
// findRestrictedFileInCommand / argMatchesAny / PathUnderAny
// --------------------------------------------------------------------------

func TestFindRestrictedFileInCommand(t *testing.T) {
	t.Run("no_args", func(t *testing.T) {
		if hit := findRestrictedFileInCommand("cat", []string{"kubeconfig"}); hit != "" {
			t.Fatalf("got %q", hit)
		}
	})
	t.Run("path_shaped_glob", func(t *testing.T) {
		if hit := findRestrictedFileInCommand("cat ./a/b.pem", []string{"**/*.pem"}); hit != "./a/b.pem" {
			t.Fatalf("got %q", hit)
		}
	})
	t.Run("bare_word_literal", func(t *testing.T) {
		if hit := findRestrictedFileInCommand("cat KubeConfig", []string{"kubeconfig"}); !strings.EqualFold(hit, "KubeConfig") {
			t.Fatalf("got %q", hit)
		}
	})
	t.Run("bare_word_ignores_glob_only_policy", func(t *testing.T) {
		// "*.pem" reduces to ".pem" via extractLiteralAnchors; bare "pem" alone
		// should not match unless the token equals the anchor.
		if hit := findRestrictedFileInCommand("echo hello", []string{"**/*.pem"}); hit != "" {
			t.Fatalf("unexpected hit %q", hit)
		}
	})
	t.Run("empty_restricted_list", func(t *testing.T) {
		if hit := findRestrictedFileInCommand("cat ./x.pem", nil); hit != "" {
			t.Fatalf("got %q", hit)
		}
	})
}

func TestArgMatchesAny(t *testing.T) {
	if !argMatchesAny("compile", []string{"compile", "test"}) {
		t.Fatal("exact match")
	}
	if !argMatchesAny("-DskipTests", []string{"-D*"}) {
		t.Fatal("glob match")
	}
	if argMatchesAny("deploy", []string{"compile", "-D*"}) {
		t.Fatal("should not match")
	}
	if !argMatchesAny("COMPILE", []string{"compile"}) {
		t.Fatal("case-insensitive exact")
	}
}

func TestPathUnderAny_LiteralAndNested(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proj")
	nested := filepath.Join(root, "src")
	if !PathUnderAny(nested, []string{root}) {
		t.Fatal("nested path should be under root")
	}
	if PathUnderAny(filepath.Join(t.TempDir(), "other"), []string{root}) {
		t.Fatal("unrelated path should not match")
	}
	if PathUnderAny(root, nil) {
		t.Fatal("empty dirs should not match")
	}
}
