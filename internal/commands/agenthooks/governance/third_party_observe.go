package governance

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/governance/transcript"
)

// Third-party decisions that were *not* blocks.
//
// A block is discovered by the correlation sweeper, because a blocked call
// never reaches PostToolUse. Everything else — an allow, an "ask", or a hook
// that failed outright — happens on calls that did run, so PostToolUse is where
// they surface. In both paths the source is the same: the agent transcript,
// carrying each hook's own command and its own words.

// recordAllThirdPartyEnv, when truthy, emits an event for every third-party
// decision including silent allows.
const recordAllThirdPartyEnv = "CX_GOVERNANCE_RECORD_ALL_THIRDPARTY"

// observeThirdPartyDecisions reads the transcript for a completed tool call and
// spools an event for each third-party hook decision worth recording.
//
// By default a bare allow — no reason, nothing to say — is skipped. Those are
// the overwhelming majority of hook invocations, and one dashboard row per
// silent approval would bury the decisions that carry meaning. Set
// CX_GOVERNANCE_RECORD_ALL_THIRDPARTY=1 to record them too.
// Returns true when the transcript named at least one third-party hook for
// this tool call — including a silent allow. That is the signal callers use to
// decide whether a weaker source (response-signature matching) should even run:
// if the transcript already knows who weighed in, there is nothing left to add.
func observeThirdPartyDecisions(transcriptPath, toolUseID, sessionID, toolName string, cfg GovernanceConfig) bool {
	if transcriptPath == "" || toolUseID == "" {
		return false
	}
	records := transcript.ThirdParty(transcript.RecordsFor(transcriptPath, toolUseID))
	if len(records) == 0 {
		return false
	}

	recordAll := isTruthy(os.Getenv(recordAllThirdPartyEnv))
	var emitted int
	for _, rec := range records {
		if !worthRecording(rec, recordAll) {
			continue
		}
		Write(thirdPartyObservation(rec, sessionID, toolName, cfg))
		emitted++
	}
	if emitted > 0 {
		log.Printf("governance: recorded %d third-party decision(s) for tool_use %s", emitted, toolUseID)
		FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	}
	return true
}

// observeResponseSignatureBlock is the fallback when the transcript found
// nothing: check whether the tool call's own failure text matches a known
// security-product signature. Only runs for shell/filesystem calls, and only
// on a call that actually failed — a successful call is never scanned.
//
// Pattern-matching free text is not a verdict — it is a guess about which
// product (if any) produced that text, so it is recorded for visibility only:
// no ThirdPartyTool, no ThirdPartyAction, no Decision.Action. Source of truth
// for an actual decision is our own PreToolUse evaluation or the transcript;
// this is neither.
func observeResponseSignatureBlock(ev claude.PostToolUseEvent, toolCategory string, cfg GovernanceConfig) {
	name, reason, confidence, ok := detectResponseSignatureBlock(toolCategory, ev.ToolResponse)
	if !ok {
		return
	}
	event := ActivityEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.third_party_enforcement",
		SessionID: ev.SessionID,
		MachineID: MachineID(),
		AgentID:   AgentID(cfg.AgentType),
		UserID:    CurrentUserName(),
		Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
		Tool: ToolInfo{
			Category: "third_party",
			Name:     ev.ToolName,
			RawName:  ev.ToolName,
		},
		DetectionMethod: detectionMethodResponseSignature,
		Decision: DecisionInfo{
			AgentMessage: "Failed " + ev.ToolName + " response resembled a " + name + " signature (confidence=" + confidence + "): " + reason,
		},
	}
	log.Printf("governance: response-signature match for tool_use %s tool=%q product=%q (confidence=%s) — recording without a verdict",
		ev.ToolUseID, ev.ToolName, name, confidence)
	Write(event)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
}

// worthRecording decides whether a third-party decision earns a dashboard row.
func worthRecording(rec transcript.HookRecord, recordAll bool) bool {
	switch rec.Decision {
	case transcript.DecisionDeny:
		// Handled by the correlation sweeper, which owns block attribution.
		return false
	case transcript.DecisionAsk, transcript.DecisionError:
		// An "ask" changed what the user saw; an error means a hook that was
		// meant to vet this call did not run. Both matter regardless of wording.
		return true
	case transcript.DecisionAllow:
		// An allow only earns a row when the hook explained itself.
		return recordAll || strings.TrimSpace(rec.Reason) != ""
	default:
		return recordAll
	}
}

// thirdPartyObservation builds the ActivityEvent for one non-blocking decision.
func thirdPartyObservation(rec transcript.HookRecord, sessionID, toolName string, cfg GovernanceConfig) ActivityEvent {
	action, ourAction := mapThirdPartyDecision(rec.Decision)

	name := rec.DisplayName()
	if name == "" {
		name = "unknown-hook"
	}

	message := rec.Reason
	if message == "" {
		message = name + " returned " + string(rec.Decision) + " for " + toolName
	}

	return ActivityEvent{
		EventID:   newEventID(),
		Timestamp: observationTime(rec),
		Kind:      "tool.third_party_enforcement",
		SessionID: sessionID,
		MachineID: MachineID(),
		AgentID:   AgentID(cfg.AgentType),
		UserID:    CurrentUserName(),
		Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
		Tool: ToolInfo{
			Category: "third_party",
			Name:     toolName,
			RawName:  toolName,
		},
		ThirdPartyTool:   name,
		ThirdPartyAction: action,
		// The transcript names the hook and quotes it verbatim — nothing inferred.
		AttributionConfidence: "definitive",
		DetectionMethod:       detectionMethodTranscript,
		ThirdPartyReason:      rec.Reason,
		ThirdPartyHookEvent:   rec.HookName,
		Decision: DecisionInfo{
			Action:       ourAction,
			AgentMessage: message,
			LatencyMs:    int64(rec.DurationMs),
		},
	}
}

// mapThirdPartyDecision converts a hook verdict into the third-party action we
// record and the governance action the dashboard colours by.
func mapThirdPartyDecision(d transcript.Decision) (thirdPartyAction, ourAction string) {
	switch d {
	case transcript.DecisionDeny:
		return "block", "BLOCK"
	case transcript.DecisionAsk:
		// The third party escalated to the user rather than deciding — closer to
		// a warning than an approval.
		return "ask", "ALERT"
	case transcript.DecisionError:
		// The call proceeded unvetted. Not a block, but not a clean allow either.
		return "error", "ALERT"
	case transcript.DecisionAllow:
		return "allow", "ALLOW"
	default:
		return string(d), "ALLOW"
	}
}

// observationTime prefers the transcript's own timestamp so the dashboard shows
// when the third party actually decided, not when we got round to reading it.
func observationTime(rec transcript.HookRecord) time.Time {
	if !rec.Timestamp.IsZero() {
		return rec.Timestamp.UTC()
	}
	return time.Now().UTC()
}

// isTruthy accepts the usual affirmative spellings for an env flag.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
