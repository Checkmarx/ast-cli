package secrets

import (
	"os"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	unresolvedEditReason = "Checkmarx secret detection could not verify this edit: the text being " +
		"replaced was not found uniquely in the file, so the result cannot be scanned reliably."
	unresolvedEditContext = "Retry with an edit whose Before/old_string text matches exactly one " +
		"location in the file (add more surrounding context if the target text repeats), then the " +
		"edit can be scanned and applied."
)

// ScanFileEdit runs secret detection on the proposed post-edit content.
// Returns blocked=true with a formatted reason, remediation context, and highest severity when the
// edit introduces a secret (delta-detection for edits; any secret for new writes). Findings the
// user already suppressed via `cx ignore-vulnerability` are filtered by the scanner. Fail-open on
// infrastructure errors. Fail closed when the post-edit content cannot be reconstructed.
// The user-visible reason never contains the secret value.
func ScanFileEdit(ev *agenthooks.FileEditEvent, svc *Scanner, telemetryWrapper wrappers.TelemetryWrapper, agent string) (blocked bool, reason, context, severity string) {
	if svc == nil || svc.scan == nil || ev == nil {
		return false, "", "", ""
	}

	findingCount := 0
	defer func() {
		if r := recover(); r != nil {
			logger.PrintfIfVerbose("secrets guardrail: recovered from panic, failing open: %v", r)
			blocked = false
			reason = ""
			context = ""
			severity = ""
		}
		logSecretsTelemetry(telemetryWrapper, agent, ev.SessionID, findingCount)
	}()

	newContent, originalContent, ok := proposedContent(ev.FilePath, ev.Changes)
	if !ok {
		findingCount = 1
		return true, provenanceTag() + " " + unresolvedEditReason, provenanceTag() + " " + unresolvedEditContext, ""
	}
	if newContent == "" {
		return false, "", "", ""
	}

	ignoreFilePath := existingIgnoreFilePath(ev.WorkDir)
	newResults, err := svc.scan(ev.FilePath, newContent, ignoreFilePath)
	if err != nil {
		logger.PrintfIfVerbose("secrets guardrail: scan of proposed content failed, failing open: %v", err)
		return false, "", "", ""
	}
	if len(newResults) == 0 {
		return false, "", "", ""
	}

	findings := newResults
	if originalContent != "" {
		origResults, origErr := svc.scan(ev.FilePath, originalContent, ignoreFilePath)
		if origErr != nil {
			logger.PrintfIfVerbose("secrets guardrail: scan of original content failed, failing open: %v", origErr)
			return false, "", "", ""
		}
		findings = NewFindings(origResults, newResults)
		if len(findings) == 0 {
			return false, "", "", ""
		}
	}

	r, c := formatFindings(ev.FilePath, findings, ev.Agent, ev.WorkDir, ev.SessionID)
	findingCount = len(findings)
	return true, r, c, highestSeverity(findings)
}

func highestSeverity(findings []secretsrealtime.SecretsRealtimeResult) string {
	rank := map[string]int{"Critical": 4, "High": 3, "Medium": 2, "Low": 1}
	best := ""
	bestRank := -1
	for i := range findings {
		if r, ok := rank[findings[i].Severity]; ok && r > bestRank {
			bestRank = r
			best = findings[i].Severity
		}
	}
	return best
}

func logSecretsTelemetry(telemetryWrapper wrappers.TelemetryWrapper, agent, sessionID string, totalCount int) {
	if telemetryWrapper == nil || totalCount == 0 {
		return
	}

	telemetryData := &wrappers.DataForAITelemetry{
		Agent:            agent + "-cli",
		AIProvider:       agent,
		Engine:           "Secrets",
		TotalCount:       totalCount,
		UniqueID:         wrappers.GetUniqueID(),
		Type:             "hooks-detect",
		SubType:          "scan",
		ScanType:         "secrets",
		AiAgentSessionId: sessionID,
	}

	if err := telemetryWrapper.SendAIDataToLog(telemetryData); err != nil {
		logger.PrintfIfVerbose("secrets guardrail: failed to send telemetry: %v", err)
	}
}

// existingIgnoreFilePath returns the realtime ignore-file path anchored at workDir only
// when it exists on disk. A missing file means the user has not suppressed anything yet.
func existingIgnoreFilePath(workDir string) string {
	p := ignore.PathFor(workDir)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
