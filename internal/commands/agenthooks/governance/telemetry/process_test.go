package telemetry

import (
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestCommandExecutable(t *testing.T) {
	cases := []struct {
		command string
		want    string
	}{
		{"npm run build", "npm"},
		{"/usr/bin/python3 script.py", "python3"},
		{`C:\tools\ripgrep.exe --files`, "ripgrep"},
		{"NODE_ENV=production webpack", "webpack"},
		{"bash -c 'go test ./...'", "'go"}, // quoting survives; substring match still works
		{"sh", ""},                         // a bare shell identifies nothing
		{"", ""},
	}
	for _, tc := range cases {
		if got := commandExecutable(tc.command); got != tc.want {
			t.Errorf("commandExecutable(%q) = %q, want %q", tc.command, got, tc.want)
		}
	}
}

func TestMatchesSignature_IgnoresPreexistingProcesses(t *testing.T) {
	now := time.Now()
	sig := ProcessSignature{Command: "npm run build", StartedAt: now}

	old := procInfo{PID: 1, CmdLine: "npm run build", CreatedAt: now.Add(-time.Hour)}
	if matchesSignature(old, sig) {
		t.Error("a process started an hour before the tool call must not match")
	}

	fresh := procInfo{PID: 2, CmdLine: "npm run build", CreatedAt: now}
	if !matchesSignature(fresh, sig) {
		t.Error("a process started at tool-call time must match")
	}
}

func TestMatchesSignature_MatchesWrappedCommandLine(t *testing.T) {
	now := time.Now()
	sig := ProcessSignature{Command: "go test ./...", StartedAt: now}
	p := procInfo{PID: 3, CmdLine: `C:\Program Files\Git\bin\bash.exe -c "go test ./..."`, CreatedAt: now}
	if !matchesSignature(p, sig) {
		t.Error("the command passed verbatim to a shell wrapper must match")
	}
}

func TestMatchesSignature_RejectsUnrelatedProcess(t *testing.T) {
	now := time.Now()
	sig := ProcessSignature{Command: "npm run build", StartedAt: now}
	p := procInfo{PID: 4, CmdLine: "chrome.exe --type=renderer", CreatedAt: now}
	if matchesSignature(p, sig) {
		t.Error("an unrelated process must not match")
	}
}

func TestConfirmProcessRan_InconclusiveWithoutCommand(t *testing.T) {
	if got := ConfirmProcessRan(ProcessSignature{}, time.Millisecond); got != Inconclusive {
		t.Fatalf("ConfirmProcessRan = %v, want Inconclusive", got)
	}
}

func TestConfirmProcessRan_ObservesRealProcess(t *testing.T) {
	// Spawn something long-lived enough to be sampled, then confirm the watcher
	// finds it in the real process table.
	var command string
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		command = "governancetelemetryprobe"
		cmd = exec.Command("powershell.exe", "-NoProfile", "-Command",
			"$governancetelemetryprobe=1; Start-Sleep -Seconds 6")
	} else {
		command = "sleep 6"
		cmd = exec.Command("sleep", "6")
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn probe process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	sig := ProcessSignature{Command: command, StartedAt: time.Now().UTC()}
	got := ConfirmProcessRan(sig, 5*time.Second)
	if got == Inconclusive {
		t.Skip("no process lister available on this host")
	}
	if got != Confirmed {
		t.Fatalf("ConfirmProcessRan = %v, want Confirmed", got)
	}
}

func TestConfirmProcessRan_NotObservedForCommandThatNeverRuns(t *testing.T) {
	sig := ProcessSignature{
		Command:   "cx-governance-command-that-never-runs --nope",
		StartedAt: time.Now().UTC(),
	}
	got := ConfirmProcessRan(sig, 600*time.Millisecond)
	if got == Inconclusive {
		t.Skip("no process lister available on this host")
	}
	if got != NotObserved {
		t.Fatalf("ConfirmProcessRan = %v, want NotObserved", got)
	}
}
