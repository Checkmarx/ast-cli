package governance

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/governance/telemetry"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/governance/transcript"
)

// Correlation tracks third-party hook enforcement via file-based state.
//
// Flow:
//   PreToolUse (our verdict=ALLOW) → writes .correlation/<id>.pre.json
//                                  → snapshots OS state (file fingerprints) and,
//                                    for shell calls, spawns a detached process watcher
//   PostToolUse                    → writes .correlation/<id>.post (empty marker)
//   Next hook call                 → sweepOrphanedPre() finds .pre.json with no .post after 5s
//                                  → asks telemetry whether the action actually happened
//                                  → emits tool.third_party_enforcement only when it did not
//
// The marker files remain the fast path: a .post means the tool ran, decided in
// microseconds with no telemetry needed. Telemetry exists purely to adjudicate
// the ambiguous case — a missing .post, which used to be assumed to mean
// "blocked" and is the sole source of this detector's false positives.
//
// Also handles Mechanism B: cx-guardrails WARN delivered via
// .hook-verdicts/<tool_use_id>.json written by the guardrail hook.

// defaultCorrelationWindow is how long a .pre marker is left alone before the
// sweeper will consider it orphaned.
//
// This was 5 seconds, which quietly assumed every tool call finishes within 5
// seconds. Plenty do not — a build, a test run, or any batch of parallel calls
// routinely runs longer. When that happened the sweeper sailed past an
// in-flight call, reported it as a third-party block, and PostToolUse then
// landed a .post with no .pre left to pair with. Those stray .post files are
// the fingerprint of the bug: this machine had 29 of them going back weeks,
// one per false block.
//
// Two minutes covers the overwhelming majority of tool calls. Detection latency
// grows to match, which is the right trade for a dashboard: a block reported a
// minute late is useful, a block that never happened is not.
const defaultCorrelationWindow = 2 * time.Minute

// correlationWindowEnv overrides the window, mainly so tests need not sleep for
// minutes. Accepts any Go duration string, e.g. "10s".
const correlationWindowEnv = "CX_GOVERNANCE_CORRELATION_WINDOW"

// correlationWindow returns the effective orphan grace period.
func correlationWindow() time.Duration {
	if v := os.Getenv(correlationWindowEnv); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
		log.Printf("governance: ignoring invalid %s=%q", correlationWindowEnv, v)
	}
	return defaultCorrelationWindow
}

// telemetrySweepWindow bounds how long the sweeper itself waits on filesystem
// telemetry. It is short because the write, if it happened, happened seconds
// ago — this is a confirmation re-check, not a live watch.
const telemetrySweepWindow = 750 * time.Millisecond

// telemetryMaxAge is how long orphaned signature/verdict files survive before
// the sweeper purges them.
const telemetryMaxAge = 10 * time.Minute

// detectionMethodTranscript marks a verdict read from the agent's own transcript
// — the strongest provenance available, because it is the deciding hook's own
// words rather than anything we inferred.
const detectionMethodTranscript = "transcript"

// detectionMethodResponseSignature marks a verdict inferred by matching a
// failed tool call's own error text against known security-product wording.
// Weaker than both other sources: it is pattern-matching on free text, not a
// structured verdict from a hook or a direct OS observation.
const detectionMethodResponseSignature = "response_signature"

// preHookState is persisted by PreToolUse when our verdict is ALLOW.
type preHookState struct {
	ToolUseID string    `json:"toolUseId"`
	ToolName  string    `json:"toolName"`
	ToolInput string    `json:"toolInput"`
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
	// Telemetry baseline captured at ALLOW time. ToolCategory selects which
	// confirmation the sweeper runs; FileSnapshots is the pre-write fingerprint
	// of every path the tool declared it would touch.
	ToolCategory  string                   `json:"toolCategory,omitempty"`
	FileSnapshots []telemetry.FileSnapshot `json:"fileSnapshots,omitempty"`
	// TranscriptPath lets the sweeper — running in a later process — read the
	// agent's own record of which hooks ran on this tool call and what they said.
	TranscriptPath string `json:"transcriptPath,omitempty"`
}

// hookVerdict is written by cx-guardrails (Mechanism B) to .hook-verdicts/<id>.json.
type hookVerdict struct {
	Source  string `json:"source"`  // "cx-guardrails"
	Verdict string `json:"verdict"` // "warn" | "block"
	Reason  string `json:"reason"`
	Event   string `json:"event"` // "PreToolUse:Bash"
}

// writePreHookState records that our PreToolUse verdict was ALLOW for this tool use.
// Called only when our own policy returned ALLOW — if we block it, there's no ambiguity.
func writePreHookState(toolUseID string, state preHookState) {
	dir := correlationDir()
	ensureDir(dir)
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	safeID := sanitizeID(toolUseID)
	_ = os.WriteFile(filepath.Join(dir, safeID+".pre.json"), data, 0o600)
}

// markPostToolUseSeen writes an empty marker so the sweeper knows PostToolUse fired.
func markPostToolUseSeen(toolUseID string) {
	dir := correlationDir()
	ensureDir(dir)
	safeID := sanitizeID(toolUseID)
	_ = os.WriteFile(filepath.Join(dir, safeID+".post"), []byte{}, 0o600)
}

// readAndDeleteHookVerdict reads the cx-guardrails verdict file (Mechanism B).
// Returns nil when no verdict file exists for this tool use.
// Uses atomic rename-then-read to prevent TOCTOU between concurrent PostToolUse processes.
func readAndDeleteHookVerdict(toolUseID string) *hookVerdict {
	safeID := sanitizeID(toolUseID)
	path := filepath.Join(hookVerdictsDir(), safeID+".json")
	tmp := path + ".read"
	if err := os.Rename(path, tmp); err != nil {
		return nil
	}
	data, err := os.ReadFile(tmp)
	_ = os.Remove(tmp)
	if err != nil {
		return nil
	}
	var v hookVerdict
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	return &v
}

// sweepOrphanedPre scans the correlation directory for .pre.json files that have
// no corresponding .post file and are older than correlationWindow.
// For each orphan, it emits a tool.third_party_enforcement event and deletes the file.
// Call at the start of any hook handler to catch blocks from the previous tool call.
func sweepOrphanedPre(cfg GovernanceConfig) {
	dir := correlationDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	window := correlationWindow()

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".post") {
			reapStrayPost(dir, entry, window)
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".pre.json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < window {
			continue // too recent — PostToolUse may still fire
		}

		prePath := filepath.Join(dir, entry.Name())
		safeID := strings.TrimSuffix(entry.Name(), ".pre.json")
		postPath := filepath.Join(dir, safeID+".post")

		// If .post exists: tool ran, not blocked — clean up both files.
		if _, err := os.Stat(postPath); err == nil {
			_ = os.Remove(prePath)
			_ = os.Remove(postPath)
			telemetry.Cleanup(telemetryDir(), safeID)
			continue
		}

		// No .post → the tool *may* have been blocked by another hook.
		data, err := os.ReadFile(prePath)
		if err != nil {
			_ = os.Remove(prePath)
			continue
		}
		var state preHookState
		if err := json.Unmarshal(data, &state); err != nil {
			_ = os.Remove(prePath)
			continue
		}
		_ = os.Remove(prePath)

		// Best source first: the agent's transcript records every hook that ran,
		// with its command and the reason it gave. When a third party blocked
		// this call, that is not an inference — it is their own statement.
		blocker, attributed := transcriptBlocker(state)

		// Ground-truth check before accusing anyone: if the OS says the action
		// actually took effect, the missing .post was a delivery gap, not a block.
		result := confirmWithTelemetry(state, safeID)
		telemetry.Cleanup(telemetryDir(), safeID)
		if result == telemetry.Confirmed && !attributed {
			log.Printf("governance: telemetry confirmed %s actually ran for tool_use %s — suppressing false third-party block",
				state.ToolName, state.ToolUseID)
			continue
		}

		emitThirdPartyBlock(state, cfg, result, blocker, attributed)
	}

	telemetry.PurgeStale(telemetryDir(), telemetryMaxAge)
}

// reapStrayPost deletes a .post marker whose .pre partner is gone.
//
// Normally the sweeper removes both together. A .post left on its own means
// PostToolUse arrived after its .pre had already been swept — the signature of
// a call that was still in flight when we declared it blocked. Without this
// they accumulate forever; they are also the cheapest evidence that the
// correlation window is set too short for this workload, so say so once.
func reapStrayPost(dir string, entry os.DirEntry, window time.Duration) {
	info, err := entry.Info()
	if err != nil || time.Since(info.ModTime()) < window {
		return // its .pre may still be mid-sweep
	}
	base := strings.TrimSuffix(entry.Name(), ".post")
	if _, err := os.Stat(filepath.Join(dir, base+".pre.json")); err == nil {
		return // partner is present; the normal path will pair them
	}
	log.Printf("governance: reaping stray .post for tool_use %s — PostToolUse arrived after its .pre was swept "+
		"(consider raising %s)", base, correlationWindowEnv)
	_ = os.Remove(filepath.Join(dir, entry.Name()))
}

// transcriptBlocker asks the agent's transcript which third-party hook stopped
// this tool call, and what reason it gave.
//
// This outranks every other source: it is the blocking hook's own words,
// recorded by the agent, rather than a name guessed from a hook inventory.
func transcriptBlocker(state preHookState) (transcript.HookRecord, bool) {
	if state.TranscriptPath == "" {
		return transcript.HookRecord{}, false
	}
	records := transcript.ThirdParty(transcript.RecordsFor(state.TranscriptPath, state.ToolUseID))
	return transcript.FirstBlocking(records)
}

// confirmWithTelemetry asks the OS whether the tool call took effect.
//
// Returns Confirmed (it ran — suppress the block), NotObserved (it really did
// not run — the block is ground truth), or Inconclusive (no usable signal, fall
// back to the marker-timing heuristic).
func confirmWithTelemetry(state preHookState, safeID string) telemetry.Result {
	switch state.ToolCategory {
	case "filesystem":
		// Deterministic: filesystem state persists, so a late check is as good
		// as an immediate one.
		return telemetry.ConfirmAnyFileWritten(state.FileSnapshots, telemetrySweepWindow)
	case "shell":
		// The watcher spawned at PreToolUse time already did the observing —
		// a process that has exited leaves nothing to sample now.
		v, ok := telemetry.ReadVerdict(telemetryDir(), safeID)
		if !ok {
			return telemetry.Inconclusive
		}
		return v.Result()
	default:
		// MCP calls, web search and friends have no OS-visible side effect we
		// can attribute — the heuristic stays in charge for those.
		return telemetry.Inconclusive
	}
}

// emitThirdPartyBlock spools a tool.third_party_enforcement event for a
// missing PostToolUse marker.
//
// Only the transcript path (attributed) is a real verdict: the blocking hook's
// own recorded words. Everything else — NotObserved, Inconclusive — is telemetry
// *failing to see the tool run*, not telemetry *seeing it get blocked*. That
// distinction matters concretely for fast shell commands: ConfirmProcessRan
// polls the process table every 350ms (see process.go), so a command that
// starts and exits between two polls is indistinguishable from one that never
// ran at all — the watcher reports NotObserved either way. Asserting a BLOCK
// verdict off that signal would flag normal, fast, successful commands as
// violations. So when nothing confirms a block, none is reported: the event is
// still written (visible in the session timeline for debugging) but carries no
// Decision.Action and is never turned into a policy_action / violation.
func emitThirdPartyBlock(
	state preHookState,
	cfg GovernanceConfig,
	telemetryResult telemetry.Result,
	blocker transcript.HookRecord,
	attributed bool,
) {
	// Carry the Claude tool whose PostToolUse marker never arrived (e.g.,
	// "Bash", "Write") so the UI can show what was in question, independent of
	// whether a block is actually being asserted.
	blockedTool := state.ToolName

	if !attributed {
		event := ActivityEvent{
			EventID:   newEventID(),
			Timestamp: time.Now().UTC(),
			Kind:      "tool.third_party_enforcement",
			SessionID: state.SessionID,
			MachineID: MachineID(),
			AgentID:   AgentID(cfg.AgentType),
			UserID:    CurrentUserName(),
			Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
			Tool: ToolInfo{
				Category: "third_party",
				Name:     blockedTool,
				RawName:  blockedTool,
			},
			DetectionMethod: telemetryResult.Method(),
			// No ThirdPartyTool, ThirdPartyAction, or Decision.Action: nothing
			// confirmed who (or whether anyone) blocked this call.
			Decision: DecisionInfo{
				AgentMessage: "No PostToolUse completion observed for " + blockedTool + "; cause unconfirmed",
			},
		}
		log.Printf("governance: unconfirmed completion gap for tool_use %s blocked=%q (telemetry=%s) — recording without a verdict",
			state.ToolUseID, blockedTool, telemetryResult)
		Write(event)
		FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
		return
	}

	// The transcript named the blocker and quoted its reason. Nothing here is
	// inferred, so this is a real, definitive verdict.
	thirdPartyTool := blocker.DisplayName()
	event := ActivityEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.third_party_enforcement",
		SessionID: state.SessionID,
		MachineID: MachineID(),
		AgentID:   AgentID(cfg.AgentType),
		UserID:    CurrentUserName(),
		Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
		Tool: ToolInfo{
			Category: "third_party",
			Name:     blockedTool,
			RawName:  blockedTool,
		},
		ThirdPartyTool:        thirdPartyTool,
		ThirdPartyAction:      "block",
		AttributionConfidence: "definitive",
		DetectionMethod:       detectionMethodTranscript,
		ThirdPartyReason:      blocker.Reason,
		ThirdPartyHookEvent:   blocker.HookName,
		Decision: DecisionInfo{
			Action:       "BLOCK",
			AgentMessage: blocker.Reason,
		},
	}

	log.Printf("governance: third-party block for tool_use %s by=%q blocked=%q reason=%q",
		state.ToolUseID, thirdPartyTool, blockedTool, blocker.Reason)
	Write(event)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
}
