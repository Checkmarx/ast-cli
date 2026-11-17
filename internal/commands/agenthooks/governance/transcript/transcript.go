// Package transcript reads hook outcomes out of the agent's own session
// transcript.
//
// Claude Code appends an attachment record for *every* hook it runs — ours and
// every third party's — carrying the hook's command, its decision, and the
// reason it gave. That makes the transcript the authoritative answer to the two
// questions telemetry cannot answer:
//
//	who blocked this?   → the hook's command line
//	why did they?       → the reason text the hook returned
//
// OS telemetry establishes *whether* an action took effect; this establishes
// *who* decided and *what they said*. The two are complementary, and together
// they replace the previous approach of guessing a vendor name from a hook
// inventory and inventing "cx-devassist" when that came up empty.
//
// Reading is strictly passive: the transcript is the agent's file, opened
// read-only and never modified.
package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// tailBytes caps how much of the transcript is read. Sessions run long and the
// file grows without bound, but hook records for the tool call being swept are
// always near the end. Reading the last slice keeps a hook invocation cheap.
const tailBytes = 4 << 20 // 4 MiB

// maxLineBytes guards bufio against a single pathological line.
const maxLineBytes = 8 << 20

// Decision is the verdict a hook returned for a tool call.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
	DecisionAsk   Decision = "ask"
	// DecisionError covers a hook that failed rather than deciding. Claude Code
	// treats a non-blocking failure as "carry on", so it is not a block — but it
	// is worth surfacing, because a security hook that silently fails open is a
	// governance gap.
	DecisionError Decision = "error"
	DecisionNone  Decision = ""
)

// HookRecord is one hook invocation as the agent recorded it.
type HookRecord struct {
	ToolUseID  string    // correlates to our .pre marker
	HookName   string    // "PreToolUse:Bash"
	HookEvent  string    // "PreToolUse" | "PostToolUse" | ...
	Command    string    // the hook's command line, or its configured description
	Decision   Decision  //
	Reason     string    // what the hook told the agent
	Blocking   bool      // true when the hook actually stopped the tool call
	ExitCode   int       //
	DurationMs int       //
	Timestamp  time.Time //
}

// DisplayName returns the most identifying human-readable name for the hook.
func (r HookRecord) DisplayName() string { return DisplayName(r.Command) }

// rawLine is the subset of a transcript line we care about.
type rawLine struct {
	Type       string         `json:"type"`
	Timestamp  time.Time      `json:"timestamp"`
	Attachment *rawAttachment `json:"attachment"`
}

// rawAttachment covers both record shapes Claude Code emits. A successful or
// non-blocking hook reports its command and stdout inline; a hook that actually
// blocked reports them nested under blockingError instead.
type rawAttachment struct {
	Type       string `json:"type"`
	HookName   string `json:"hookName"`
	ToolUseID  string `json:"toolUseID"`
	HookEvent  string `json:"hookEvent"`
	Command    string `json:"command"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exitCode"`
	DurationMs int    `json:"durationMs"`

	BlockingError *struct {
		BlockingError string `json:"blockingError"`
		Command       string `json:"command"`
	} `json:"blockingError"`
}

// hookStdout is the JSON a PreToolUse hook writes to stdout.
type hookStdout struct {
	HookSpecificOutput *struct {
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
		AdditionalContext        string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
	// Older/alternate form some hooks still emit.
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// RecordsFor returns every hook record in the transcript for one tool use.
// A missing or unreadable transcript yields no records rather than an error:
// this is an enrichment path, and governance must not fail because of it.
func RecordsFor(transcriptPath, toolUseID string) []HookRecord {
	if transcriptPath == "" || toolUseID == "" {
		return nil
	}
	all, err := Parse(transcriptPath)
	if err != nil {
		return nil
	}
	var out []HookRecord
	for _, r := range all {
		if r.ToolUseID == toolUseID {
			out = append(out, r)
		}
	}
	return out
}

// Parse reads the tail of a transcript and returns every hook record in it.
func Parse(transcriptPath string) ([]HookRecord, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var reader io.Reader = f
	if info.Size() > tailBytes {
		if _, err := f.Seek(info.Size()-tailBytes, io.SeekStart); err != nil {
			return nil, err
		}
		// The seek almost certainly landed mid-line; drop the partial remainder.
		br := bufio.NewReaderSize(f, 64<<10)
		if _, err := br.ReadString('\n'); err != nil && err != io.EOF {
			return nil, err
		}
		reader = br
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64<<10), maxLineBytes)

	var out []HookRecord
	for scanner.Scan() {
		line := scanner.Bytes()
		// Cheap reject before paying for a JSON parse — most lines are messages.
		if !strings.Contains(string(line), `"hookName"`) {
			continue
		}
		var raw rawLine
		if err := json.Unmarshal(line, &raw); err != nil {
			continue // a truncated or malformed line is not fatal
		}
		if raw.Attachment == nil || raw.Attachment.HookName == "" {
			continue
		}
		out = append(out, toRecord(raw))
	}
	// A scanner error still returns whatever parsed cleanly.
	return out, nil
}

// toRecord normalises the two attachment shapes into one HookRecord.
func toRecord(raw rawLine) HookRecord {
	a := raw.Attachment
	rec := HookRecord{
		ToolUseID:  a.ToolUseID,
		HookName:   a.HookName,
		HookEvent:  a.HookEvent,
		Command:    a.Command,
		ExitCode:   a.ExitCode,
		DurationMs: a.DurationMs,
		Timestamp:  raw.Timestamp,
	}

	// A hook that actually stopped the call: command and reason live nested.
	if a.BlockingError != nil {
		rec.Blocking = true
		rec.Decision = DecisionDeny
		rec.Reason = a.BlockingError.BlockingError
		if a.BlockingError.Command != "" {
			rec.Command = a.BlockingError.Command
		}
		return rec
	}

	// Otherwise the verdict is in the hook's stdout, when it produced any.
	if a.Stdout != "" {
		var so hookStdout
		if err := json.Unmarshal([]byte(a.Stdout), &so); err == nil {
			switch {
			case so.HookSpecificOutput != nil && so.HookSpecificOutput.PermissionDecision != "":
				rec.Decision = Decision(so.HookSpecificOutput.PermissionDecision)
				rec.Reason = so.HookSpecificOutput.PermissionDecisionReason
				if rec.Reason == "" {
					rec.Reason = so.HookSpecificOutput.AdditionalContext
				}
			case so.Decision != "":
				// "block" in the legacy form means the same as "deny".
				if so.Decision == "block" {
					rec.Decision = DecisionDeny
				} else {
					rec.Decision = Decision(so.Decision)
				}
				rec.Reason = so.Reason
			}
		}
	}

	// A hook that failed never rendered a verdict. Record that distinctly: the
	// tool call proceeded, but a hook that was supposed to vet it did not run.
	if rec.Decision == DecisionNone && a.ExitCode != 0 {
		rec.Decision = DecisionError
		rec.Reason = firstLine(a.Stderr)
	}
	rec.Blocking = rec.Decision == DecisionDeny
	return rec
}

// governanceMarkers identify our own hook routes, so we never attribute our
// decisions to a third party.
var governanceMarkers = []string{"gov-claude-", "gov-cursor-", "gov-telemetry-watch"}

// IsGovernanceHook reports whether a command is one of our own governance hooks.
func IsGovernanceHook(command string) bool {
	lower := strings.ToLower(command)
	for _, m := range governanceMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// ThirdParty filters a record set down to hooks that are not ours.
func ThirdParty(records []HookRecord) []HookRecord {
	var out []HookRecord
	for _, r := range records {
		if !IsGovernanceHook(r.Command) {
			out = append(out, r)
		}
	}
	return out
}

// FirstBlocking returns the first record that stopped the tool call, if any.
func FirstBlocking(records []HookRecord) (HookRecord, bool) {
	for _, r := range records {
		if r.Blocking {
			return r, true
		}
	}
	return HookRecord{}, false
}

// genericScriptNames are filenames that identify a hook's role but not its
// owner — the directory above them carries the actual identity.
var genericScriptNames = map[string]bool{
	"check": true, "hook": true, "run": true, "main": true,
	"index": true, "wrapper": true, "entry": true, "pre-tool-use": true,
}

// DisplayName derives a human-readable owner from a hook's command.
//
// Claude Code puts either a real command line here, or — when the hook config
// supplies one — a prose description like "Checking command against security
// policy...". Both identify the hook to a reader, so both are usable; they just
// need different handling.
func DisplayName(command string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return ""
	}
	// A prose description has no path separators and reads as a sentence.
	if !strings.ContainsAny(cmd, "/\\") {
		return strings.TrimSuffix(cmd, "...")
	}

	// Find the script or binary the command actually runs: the last token that
	// looks like a path, so flags and interpreter arguments are skipped.
	token := ""
	for _, f := range strings.Fields(cmd) {
		if strings.HasPrefix(f, "-") {
			continue
		}
		if strings.ContainsAny(f, "/\\") {
			token = f
		}
	}
	if token == "" {
		token = strings.Fields(cmd)[0]
	}
	token = strings.Trim(token, `"'`)

	base := filepath.Base(filepath.ToSlash(token))
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// "check.ps1" says nothing; ".../aegis-hook/check.ps1" says Aegis.
	if genericScriptNames[strings.ToLower(name)] {
		if parent := filepath.Base(filepath.Dir(filepath.ToSlash(token))); parent != "." && parent != "/" && parent != "" {
			return parent
		}
	}
	return name
}

// firstLine trims a multi-line message down to its first line, which is where
// a hook's actual complaint lives; the rest is usually a stack trace.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
