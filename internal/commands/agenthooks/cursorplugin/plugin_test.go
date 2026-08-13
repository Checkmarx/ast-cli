package cursorplugin

import (
	"runtime"
	"strings"
	"testing"
)

func TestMCPTool(t *testing.T) {
	got := MCPTool("codeRemediation")
	want := "mcp__plugin-cx-devassist-Checkmarx__codeRemediation"
	if got != want {
		t.Errorf("MCPTool() = %q, want %q", got, want)
	}
}

func TestIgnoreVulnerabilityCommand_WindowsUsesStopParsing(t *testing.T) {
	if runtime.GOOS != goosWindows {
		t.Skip("windows-only")
	}
	data := []byte(`{"FileName":"Demo.java","Line":5,"RuleID":1027}`)
	cmd := IgnoreVulnerabilityCommand(`C:\cx\cx.exe`, "asca", data, ` --ignored-file-path "c:/proj/.checkmarx/ignored.json"`, "")
	if !strings.Contains(cmd, `--% ignore-vulnerability`) {
		t.Errorf("expected --%% stop-parsing, got %q", cmd)
	}
	want := `--data "{\"FileName\":\"Demo.java\",\"Line\":5,\"RuleID\":1027}"`
	if !strings.Contains(cmd, want) {
		t.Errorf("expected quoted backslash-escaped JSON, got %q", cmd)
	}
}

func TestIgnoreVulnerabilityCommand_UnixEscapesJSON(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("unix-only")
	}
	data := []byte(`{"FileName":"Demo.java"}`)
	cmd := IgnoreVulnerabilityCommand("cx", "asca", data, "", "")
	if !strings.Contains(cmd, `\"FileName\"`) {
		t.Errorf("expected backslash-escaped JSON on unix, got %q", cmd)
	}
}
