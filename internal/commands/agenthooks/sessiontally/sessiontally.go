// Package sessiontally accumulates per-engine scan counts across an AI-agent session so the
// session-end (Stop) hook can emit a per-engine telemetry summary.
//
// Why on disk: every `cx hooks <route>` invocation is a FRESH OS process (see
// internal/commands/agenthooks.go HookDispatchCommands). No in-memory state survives between the
// per-edit/per-tool detect events and the session-end idle event, so the running tally must live on
// disk — the same reason the sibling `.session-cx-findings` marker exists.
//
// Design guarantees (mirroring touchSessionFindingsMarker's contract): every function here is
// BEST-EFFORT and must NEVER return an error or panic into its caller — a tally write failure must
// never affect a scan verdict. Writes are append-only NDJSON so concurrent hook processes (parallel
// tool calls / multi-file edits) never lose records to a read-modify-write race without needing a
// lock file; Load folds the records into a per-engine map.
//
// Session scoping: files are keyed by a sanitized session id. Events whose session id could not be
// resolved (e.g. a non-Claude ToolCall, whose library event carries no SessionID) are written to a
// shared "default" bucket that Load also reads and Clear also removes. Two concurrent sessions can
// therefore both feed the default bucket and one session's Clear can discard the other's
// default-bucket counts — accepted best-effort noise, since the summary itself is best-effort.
package sessiontally

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Counts is the per-engine roll-up Load returns. JSON tags let it serialize cleanly into the local
// session-summary audit log.
type Counts struct {
	VulnerabilitiesFound int `json:"vulnerabilitiesFound"`
	RemediationsOffered  int `json:"remediationsOffered"`
}

const (
	filePrefix  = ".session-cx-tallies-"
	fileSuffix  = ".json"
	defaultKey  = "default"
	maxAgeHours = 24
	maxIDLen    = 128
	dirPerm     = 0o700
	filePerm    = 0o600
	// scanBufSize/scanMaxSize bound bufio.Scanner's growth when folding a tally file:
	// start at 64KiB, allow growth up to 1MiB per line before Scan reports an error.
	scanBufSize = 64 * 1024
	scanMaxSize = 1024 * 1024
)

// Only filename-safe characters survive; everything else (path separators, "..", spaces, control
// bytes) collapses to '_', so a hostile session id can never escape the ~/.checkmarx directory.
var sanitizeRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// record is one append-only NDJSON line.
type record struct {
	Engine     string `json:"engine"`
	Found      int    `json:"found"`
	RemOffered int    `json:"remOffered"`
}

func baseDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".checkmarx"), true
}

func sanitize(sessionID string) string {
	s := sanitizeRe.ReplaceAllString(sessionID, "_")
	if s == "" {
		s = defaultKey
	}
	if len(s) > maxIDLen {
		s = s[:maxIDLen]
	}
	return s
}

func tallyPath(sessionID string) (string, bool) {
	dir, ok := baseDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, filePrefix+sanitize(sessionID)+fileSuffix), true
}

// Add appends one best-effort NDJSON record for (sessionID, engine). Never returns or panics.
func Add(sessionID, engine string, foundDelta, remOfferedDelta int) {
	defer func() { _ = recover() }()
	path, ok := tallyPath(sessionID)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return
	}
	line, err := json.Marshal(record{Engine: engine, Found: foundDelta, RemOffered: remOfferedDelta})
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePerm)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(line, '\n'))
}

// Load folds the session's tally file AND the shared default file into a per-engine map. Best-effort:
// any read/parse error is skipped and whatever parsed so far is returned. Also opportunistically
// removes stale tally files (>24h) so a session whose Stop hook never fired cannot leak forever.
func Load(sessionID string) map[string]Counts {
	out := map[string]Counts{}
	defer func() { _ = recover() }()
	seen := map[string]bool{}
	for _, sid := range []string{sessionID, defaultKey} {
		path, ok := tallyPath(sid)
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		foldFile(path, out)
	}
	cleanupStale()
	return out
}

func foldFile(path string, out map[string]Counts) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, scanBufSize), scanMaxSize)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r record
		if json.Unmarshal([]byte(line), &r) != nil || r.Engine == "" {
			continue
		}
		c := out[r.Engine]
		c.VulnerabilitiesFound += r.Found
		c.RemediationsOffered += r.RemOffered
		out[r.Engine] = c
	}
}

// Clear removes the session's tally file AND the shared default file. Best-effort.
func Clear(sessionID string) {
	defer func() { _ = recover() }()
	seen := map[string]bool{}
	for _, sid := range []string{sessionID, defaultKey} {
		path, ok := tallyPath(sid)
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		_ = os.Remove(path)
	}
}

func cleanupStale() {
	dir, ok := baseDir()
	if !ok {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAgeHours * time.Hour)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, filePrefix) || !strings.HasSuffix(name, fileSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
