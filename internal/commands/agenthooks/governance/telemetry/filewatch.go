package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"
)

// hashSizeLimit caps how much of a file we digest. Content hashing catches
// in-place edits that leave size and mtime untouched (some editors preserve
// mtime); beyond this size the stat fields alone are relied upon.
const hashSizeLimit = 2 << 20 // 2 MiB

// FileSnapshot is a cheap fingerprint of a file taken before a write is allowed.
// It is persisted in the correlation .pre.json so a later hook process — a
// different OS process entirely — can diff against the current state.
type FileSnapshot struct {
	Path    string    `json:"path"`
	Exists  bool      `json:"exists"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Hash    string    `json:"hash,omitempty"` // sha256 of content, empty for large/unreadable files
}

// Snapshot fingerprints path. A missing file yields Exists=false, which is
// itself a meaningful baseline: a later-existing file proves the write ran.
func Snapshot(path string) FileSnapshot {
	snap := FileSnapshot{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		return snap
	}
	snap.Exists = true
	snap.Size = info.Size()
	snap.ModTime = info.ModTime()
	if info.Size() <= hashSizeLimit {
		snap.Hash = hashFile(path)
	}
	return snap
}

// SnapshotAll fingerprints every path a tool call declared it would touch.
func SnapshotAll(paths []string) []FileSnapshot {
	if len(paths) == 0 {
		return nil
	}
	out := make([]FileSnapshot, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		out = append(out, Snapshot(p))
	}
	return out
}

// ConfirmFileWritten reports whether the file changed relative to before.
//
// It polls until window elapses so a slow write still registers, but returns as
// soon as a change is seen. Unlike the process watcher this is genuinely
// deterministic: filesystem state persists, so a late call is just as valid as
// an immediate one — no signal can be "missed".
func ConfirmFileWritten(before FileSnapshot, window time.Duration) Result {
	if before.Path == "" {
		return Inconclusive
	}
	deadline := time.Now().Add(window)
	for {
		if changed(before, Snapshot(before.Path)) {
			return Confirmed
		}
		if !time.Now().Before(deadline) {
			return NotObserved
		}
		time.Sleep(pollInterval)
	}
}

// ConfirmAnyFileWritten returns Confirmed if any of the snapshotted paths
// changed. A write tool touching several files only needs one to have landed to
// prove it was not blocked.
func ConfirmAnyFileWritten(before []FileSnapshot, window time.Duration) Result {
	if len(before) == 0 {
		return Inconclusive
	}
	// Fast path first: check every path once before spending the window.
	for _, b := range before {
		if changed(b, Snapshot(b.Path)) {
			return Confirmed
		}
	}
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)
		for _, b := range before {
			if changed(b, Snapshot(b.Path)) {
				return Confirmed
			}
		}
	}
	return NotObserved
}

// changed reports whether now differs from before in any observable way.
func changed(before, now FileSnapshot) bool {
	if before.Exists != now.Exists {
		return true
	}
	if !now.Exists {
		return false
	}
	if before.Size != now.Size || !before.ModTime.Equal(now.ModTime) {
		return true
	}
	// Same size and mtime: only a content hash can tell them apart.
	if before.Hash != "" && now.Hash != "" {
		return before.Hash != now.Hash
	}
	return false
}

// hashFile returns the hex sha256 of the file, or "" when it cannot be read.
func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
