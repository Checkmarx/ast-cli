package telemetry

import (
	"path/filepath"
	"strings"
	"time"
)

// ProcessSignature describes the process a shell tool call is expected to spawn.
// It is captured at PreToolUse time (when we still know the command) and matched
// against live process command lines by the watcher.
type ProcessSignature struct {
	Command   string    `json:"command"`   // the shell command as the agent submitted it
	WorkDir   string    `json:"workDir"`   // cwd at hook time, best-effort context only
	StartedAt time.Time `json:"startedAt"` // processes older than this are pre-existing
}

// procInfo is one row of the OS process table.
type procInfo struct {
	PID       int
	CmdLine   string
	CreatedAt time.Time // zero when the platform lister cannot supply it
}

// shellWrappers are the interpreters an agent uses to run a command. A match on
// the wrapper alone proves nothing — the command text has to match too.
var shellWrappers = map[string]bool{
	"bash": true, "sh": true, "zsh": true, "dash": true,
	"cmd": true, "powershell": true, "pwsh": true, "conhost": true,
	"git-bash": true, "busybox": true, "env": true, "sudo": true,
}

// ConfirmProcessRan watches the process table for a process matching sig.
//
// Returns Confirmed as soon as one is seen, NotObserved if the full window
// elapses without a match, and Inconclusive when the platform's process lister
// is unavailable (so the caller keeps the old heuristic rather than reporting a
// false block).
//
// Caveat by construction: sampling cannot see a process that starts and exits
// entirely between two polls. That skews only toward NotObserved, and callers
// treat NotObserved as "keep the block we already inferred" — so a miss costs
// nothing beyond losing the suppression this watcher exists to provide. It never
// invents a block that the old heuristic would not also have reported.
func ConfirmProcessRan(sig ProcessSignature, window time.Duration) Result {
	if strings.TrimSpace(sig.Command) == "" {
		return Inconclusive
	}
	deadline := time.Now().Add(window)
	sampled := false
	for {
		procs, err := listProcesses()
		if err != nil {
			if !sampled {
				return Inconclusive // no working process lister on this host
			}
			return NotObserved
		}
		sampled = true
		for i := range procs {
			if matchesSignature(procs[i], sig) {
				return Confirmed
			}
		}
		if !time.Now().Before(deadline) {
			return NotObserved
		}
		time.Sleep(pollInterval)
	}
}

// matchesSignature reports whether a live process is plausibly the one the
// shell tool call was going to spawn.
func matchesSignature(p procInfo, sig ProcessSignature) bool {
	if p.CmdLine == "" {
		return false
	}
	// Ignore processes that were already running before the tool call — they
	// cannot be the one this call spawned.
	if !p.CreatedAt.IsZero() && p.CreatedAt.Before(sig.StartedAt.Add(-2*time.Second)) {
		return false
	}
	cmdline := strings.ToLower(p.CmdLine)
	want := strings.ToLower(strings.TrimSpace(sig.Command))

	// Strongest signal: the agent passes the command verbatim to `bash -c`, so
	// the whole string shows up in the wrapper's command line.
	if strings.Contains(cmdline, want) {
		return true
	}
	// Otherwise fall back to the executable the command invokes, which survives
	// quoting differences between the agent and the OS.
	exe := commandExecutable(want)
	if exe == "" {
		return false
	}
	return strings.Contains(cmdline, exe)
}

// commandExecutable extracts the program a shell command runs — "npm" from
// `npm run build`. Returns "" for generic shell wrappers, whose presence in the
// process table carries no information about this particular command.
func commandExecutable(command string) string {
	fields := strings.Fields(command)
	for _, f := range fields {
		if strings.HasPrefix(f, "-") {
			continue // a flag on a wrapper we already skipped
		}
		// Skip leading VAR=value assignments, which precede the real program.
		if strings.Contains(f, "=") {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(f), ".exe")
		if name == "" || shellWrappers[name] {
			continue
		}
		if len(name) < 2 {
			return "" // too generic to match on
		}
		return name
	}
	return ""
}
