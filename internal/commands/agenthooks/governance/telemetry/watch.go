package telemetry

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Process telemetry cannot be gathered after the fact: a command that already
// finished leaves no trace in the process table. The hook itself must return
// within milliseconds, so it instead hands the signature to a detached watcher
// that observes the window and leaves a verdict file behind. The correlation
// sweeper — running in a later hook process — reads that verdict.
//
// Layout under the telemetry directory, keyed by tool-use id:
//
//	<id>.sig.json      signature written by PreToolUse, consumed by the watcher
//	<id>.verdict.json  result written by the watcher, consumed by the sweeper

// Verdict is the persisted outcome of a watcher run.
type Verdict struct {
	ToolUseID string    `json:"toolUseId"`
	Outcome   string    `json:"outcome"` // Result.String()
	Signal    string    `json:"signal"`  // "process" | "file"
	WrittenAt time.Time `json:"writtenAt"`
}

// Result converts the persisted outcome back into a Result.
func (v Verdict) Result() Result {
	switch v.Outcome {
	case Confirmed.String():
		return Confirmed
	case NotObserved.String():
		return NotObserved
	default:
		return Inconclusive
	}
}

// WriteSignature persists the expected process signature for a tool use so the
// detached watcher can pick it up without it appearing in the watcher's own
// command line (which would otherwise match itself).
func WriteSignature(dir, toolUseID string, sig ProcessSignature) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(sig)
	if err != nil {
		return err
	}
	return os.WriteFile(sigPath(dir, toolUseID), data, 0o600)
}

// ReadSignature loads a previously written signature.
func ReadSignature(dir, toolUseID string) (ProcessSignature, bool) {
	data, err := os.ReadFile(sigPath(dir, toolUseID))
	if err != nil {
		return ProcessSignature{}, false
	}
	var sig ProcessSignature
	if err := json.Unmarshal(data, &sig); err != nil {
		return ProcessSignature{}, false
	}
	return sig, true
}

// WriteVerdict records a watcher outcome for the sweeper to read later.
func WriteVerdict(dir, toolUseID string, r Result, signal string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(Verdict{
		ToolUseID: toolUseID,
		Outcome:   r.String(),
		Signal:    signal,
		WrittenAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	// Write-then-rename so the sweeper never reads a half-written verdict.
	tmp := verdictPath(dir, toolUseID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, verdictPath(dir, toolUseID))
}

// ReadVerdict returns the watcher's verdict for a tool use, if it has landed.
func ReadVerdict(dir, toolUseID string) (Verdict, bool) {
	data, err := os.ReadFile(verdictPath(dir, toolUseID))
	if err != nil {
		return Verdict{}, false
	}
	var v Verdict
	if err := json.Unmarshal(data, &v); err != nil {
		return Verdict{}, false
	}
	return v, true
}

// Cleanup removes both the signature and verdict files for a tool use.
func Cleanup(dir, toolUseID string) {
	_ = os.Remove(sigPath(dir, toolUseID))
	_ = os.Remove(verdictPath(dir, toolUseID))
}

// PurgeStale deletes telemetry artifacts older than maxAge. Watchers can die
// before writing a verdict (machine sleep, process kill), so nothing here is
// guaranteed to be cleaned up by the normal path.
func PurgeStale(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < maxAge {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// SpawnProcessWatcher starts a detached watcher for a shell tool call and
// returns immediately — the hook must not block the agent.
//
// route is the hidden CLI route that runs the watcher body (see RunProcessWatcher).
func SpawnProcessWatcher(exePath, dir, toolUseID string, sig ProcessSignature, window time.Duration, route ...string) error {
	if err := WriteSignature(dir, toolUseID, sig); err != nil {
		return err
	}
	args := append(append([]string{}, route...),
		"--telemetry-dir", dir,
		"--tool-use-id", toolUseID,
		"--window", window.String(),
	)
	cmd := exec.Command(exePath, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	hideConsole(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the child asynchronously so it does not linger as a zombie on POSIX.
	go func() { _ = cmd.Wait() }()
	return nil
}

// RunProcessWatcher is the watcher body: load the signature, observe the process
// table for the window, and persist the verdict. Runs in the detached child.
func RunProcessWatcher(dir, toolUseID string, window time.Duration) error {
	sig, ok := ReadSignature(dir, toolUseID)
	if !ok {
		return WriteVerdict(dir, toolUseID, Inconclusive, "process")
	}
	result := ConfirmProcessRan(sig, window)
	_ = os.Remove(sigPath(dir, toolUseID))
	return WriteVerdict(dir, toolUseID, result, "process")
}

func sigPath(dir, id string) string     { return filepath.Join(dir, safeID(id)+".sig.json") }
func verdictPath(dir, id string) string { return filepath.Join(dir, safeID(id)+".verdict.json") }

// safeID strips path separators and other unsafe characters from a tool-use id
// so it can be used as a filename.
func safeID(id string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, id)
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}
