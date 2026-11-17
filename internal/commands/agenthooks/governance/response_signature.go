package governance

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Detecting a block that never touched the hook framework.
//
// The transcript (see package transcript) only sees what happened *inside*
// Claude Code's hook system. Some blockers act below that: antivirus killing a
// spawned process, a firewall dropping a connection, an OS permission denial.
// In those cases the tool call still "completes" as far as Claude Code is
// concerned — it just fails — so PostToolUse fires normally and its
// tool_response carries the real error text. This file reads that text and
// matches it against a small table of known security-product signatures.
//
// This is pattern matching on free text, not a structured verdict — weaker
// evidence than the transcript's own words, and weaker than telemetry's direct
// OS observation. It exists only to cover the gap those two leave: a real
// external block that produced no hook record and, because the call did run,
// no missing-marker signal either.

// responseSignature is one recognizable block message.
type responseSignature struct {
	Name       string         // human-readable product name
	Pattern    *regexp.Regexp // matched case-insensitively against the failure text
	Confidence string         // "definitive" for a named product, "candidate" for a generic phrase
}

// blockSignatures is deliberately small and conservative. Vendor-specific
// phrasing is "definitive" — nothing else produces that exact string. Generic
// phrasing (bare "access is denied") is excluded entirely: it is the normal
// text for mundane permission errors (a locked file, a missing sudo) far more
// often than for a security product, and a wrong accusation is worse than a
// missing one.
var blockSignatures = []responseSignature{
	{"Windows Defender", regexp.MustCompile(`(?i)windows defender|microsoft defender|operation did not complete successfully because the file contains a virus`), "definitive"},
	{"Windows Group Policy", regexp.MustCompile(`(?i)blocked by (your organization'?s? )?group policy|this program is blocked by group policy`), "definitive"},
	{"CrowdStrike Falcon", regexp.MustCompile(`(?i)crowdstrike|falcon sensor`), "definitive"},
	{"SentinelOne", regexp.MustCompile(`(?i)sentinelone`), "definitive"},
	{"Symantec Endpoint Protection", regexp.MustCompile(`(?i)symantec endpoint protection|symantec.{0,20}blocked`), "definitive"},
	{"McAfee", regexp.MustCompile(`(?i)mcafee`), "definitive"},
	{"Corporate proxy/firewall", regexp.MustCompile(`(?i)blocked by (your (organization'?s? )?)?(network administrator|corporate proxy|content filter|web filter)|err_network_access_denied`), "candidate"},
	{"Data loss prevention", regexp.MustCompile(`(?i)\bdlp\b.{0,30}(block|polic)|blocked by data loss prevention`), "candidate"},
}

// toolResponseFields is the subset of a Bash/Write tool_response we read.
// Shape confirmed against the ast-cx-hooks contract test: Bash reports
// {"output","exitCode"}; other tools vary but converge on the same names.
type toolResponseFields struct {
	ExitCode json.Number `json:"exitCode"`
	Success  *bool       `json:"success"`
	Error    string      `json:"error"`
	Output   string      `json:"output"`
	Stderr   string      `json:"stderr"`
	Message  string      `json:"message"`
}

// failureText extracts the combined error/output text from a tool_response,
// and reports whether the response actually represents a failure. A signature
// is only searched for on a failure — matching vendor phrases against a
// successful call's output would misfire on innocent mentions of the words.
func failureText(raw json.RawMessage) (text string, failed bool) {
	if len(raw) == 0 {
		return "", false
	}
	var f toolResponseFields
	if err := json.Unmarshal(raw, &f); err != nil {
		return "", false
	}

	if f.Success != nil && !*f.Success {
		failed = true
	}
	if code, err := f.ExitCode.Int64(); err == nil && code != 0 {
		failed = true
	}
	if f.Error != "" {
		failed = true
	}
	if !failed {
		return "", false
	}

	var b strings.Builder
	for _, s := range []string{f.Error, f.Stderr, f.Message, f.Output} {
		if s != "" {
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String()), true
}

// matchBlockSignature checks failure text against the known-signature table.
// Returns the first match; the table is small enough that order is stable and
// more specific vendor names are listed first.
func matchBlockSignature(text string) (name, confidence string, ok bool) {
	if text == "" {
		return "", "", false
	}
	for _, sig := range blockSignatures {
		if sig.Pattern.MatchString(text) {
			return sig.Name, sig.Confidence, true
		}
	}
	return "", "", false
}

// responseSignatureCategories limits detection to tools whose failures can
// plausibly indicate an external block. A failed WebSearch or MCP call is
// almost always an API error, not a security product — including those would
// only add false positives.
var responseSignatureCategories = map[string]bool{
	"shell":      true,
	"filesystem": true,
}

// detectResponseSignatureBlock is the entry point: given a tool call's
// category and its raw tool_response, returns a match if one is found.
func detectResponseSignatureBlock(toolCategory string, rawResponse json.RawMessage) (name, reason, confidence string, ok bool) {
	if !responseSignatureCategories[toolCategory] {
		return "", "", "", false
	}
	text, failed := failureText(rawResponse)
	if !failed {
		return "", "", "", false
	}
	name, confidence, ok = matchBlockSignature(text)
	if !ok {
		return "", "", "", false
	}
	return name, firstLines(text, 2), confidence, true
}

// firstLines trims free-form error text down to its first n lines — enough to
// carry the matched phrase and immediate context without a full stack trace.
func firstLines(s string, n int) string {
	lines := strings.SplitN(strings.TrimSpace(s), "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
