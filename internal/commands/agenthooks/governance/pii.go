package governance

import (
	"regexp"
	"sort"
	"strings"
)

// PIIPolicy configures PII scanning for prompt.submit events.
type PIIPolicy struct {
	Enabled        bool               `json:"enabled"`
	Action         string             `json:"action"` // default: "alert" | "block"
	BuiltIn        []string           `json:"built_in"`
	CustomPatterns []PIICustomPattern `json:"custom_patterns,omitempty"`
}

// PIICustomPattern is a user-defined PII detection rule.
type PIICustomPattern struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Action  string `json:"action,omitempty"` // overrides PIIPolicy.Action when set
}

// PIIMatchMeta is the serialized form stored in ActivityEvent.
// Contains no offsets and no original matched text — only type + action.
type PIIMatchMeta struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

// PIIScanResult is the output of PIIScanner.Scan.
type PIIScanResult struct {
	Found        bool
	Matches      []PIIMatchMeta
	Decision     string // "ALLOW" | "ALERT" | "BLOCK"
	RedactedText string // prompt with PII replaced by [PII:name] tokens; empty when Found=false
}

// PIIScanner evaluates prompt text against a compiled set of PII patterns.
type PIIScanner struct {
	patterns []piiCompiledPattern
}

type piiCompiledPattern struct {
	name   string
	re     *regexp.Regexp
	action string
}

// piiMatchRange is used internally during scanning to track byte offsets.
type piiMatchRange struct {
	start, end int
	name       string
	action     string
}

// builtInPIIPatterns is the default pattern set included when named in PIIPolicy.BuiltIn.
// Built-in patterns that are inherently high-risk ("block") escalate above a "alert" default.
var builtInPIIPatterns = []struct {
	name          string
	pattern       string
	defaultAction string
}{
	{"email", `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, "alert"},
	{"ssn", `\b\d{3}[-\s]\d{2}[-\s]\d{4}\b`, "block"},
	{"credit_card", `\b(?:\d[ \-]?){13,16}\b`, "block"},
	{"phone", `(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}`, "alert"},
	{"ip_address", `\b(?:\d{1,3}\.){3}\d{1,3}\b`, "alert"},
	{"api_key", `(?:sk-[a-zA-Z0-9]{20,}|AKIA[A-Z0-9]{16}|ghp_[a-zA-Z0-9]{36}|glpat-[a-zA-Z0-9\-]{20}|xox[bpoas]-[a-zA-Z0-9\-]+)`, "block"},
	{"jwt", `eyJ[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+`, "block"},
}

// NewPIIScanner constructs a scanner from a PIIPolicy.
// Returns nil when policy is nil or disabled — callers treat nil scanner as scan-disabled (ALLOW all).
func NewPIIScanner(policy *PIIPolicy) *PIIScanner {
	if policy == nil || !policy.Enabled {
		return nil
	}
	enabled := make(map[string]bool, len(policy.BuiltIn))
	for _, n := range policy.BuiltIn {
		enabled[strings.ToLower(n)] = true
	}

	defaultAction := strings.ToLower(policy.Action)
	if defaultAction == "" {
		defaultAction = "alert"
	}

	sc := &PIIScanner{}

	for _, def := range builtInPIIPatterns {
		if !enabled[def.name] {
			continue
		}
		re, err := regexp.Compile(def.pattern)
		if err != nil {
			continue
		}
		// High-risk built-ins escalate above the default action but never de-escalate.
		action := defaultAction
		if def.defaultAction == "block" {
			action = "block"
		}
		sc.patterns = append(sc.patterns, piiCompiledPattern{def.name, re, action})
	}

	for _, cp := range policy.CustomPatterns {
		re, err := regexp.Compile(cp.Pattern)
		if err != nil {
			continue
		}
		action := strings.ToLower(cp.Action)
		if action == "" {
			action = defaultAction
		}
		sc.patterns = append(sc.patterns, piiCompiledPattern{cp.Name, re, action})
	}
	return sc
}

// Scan checks text for PII. Returns PIIScanResult with Found=false and Decision="ALLOW" when clean.
// When PII is found, RedactedText contains the prompt with matched ranges replaced by [PII:name].
func (s *PIIScanner) Scan(text string) PIIScanResult {
	if s == nil || len(s.patterns) == 0 {
		return PIIScanResult{Decision: "ALLOW"}
	}

	var ranges []piiMatchRange
	for _, pat := range s.patterns {
		for _, loc := range pat.re.FindAllStringIndex(text, -1) {
			ranges = append(ranges, piiMatchRange{
				start:  loc[0],
				end:    loc[1],
				name:   pat.name,
				action: pat.action,
			})
		}
	}
	if len(ranges) == 0 {
		return PIIScanResult{Decision: "ALLOW"}
	}

	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })

	// Build deduplicated match metadata and determine highest severity across all matches.
	seen := make(map[string]bool)
	var matches []PIIMatchMeta
	decision := "ALERT"
	for _, r := range ranges {
		if !seen[r.name] {
			seen[r.name] = true
			matches = append(matches, PIIMatchMeta{Name: r.name, Action: r.action})
		}
		if r.action == "block" {
			decision = "BLOCK"
		}
	}

	return PIIScanResult{
		Found:        true,
		Matches:      matches,
		Decision:     decision,
		RedactedText: redactPII(text, ranges),
	}
}

// redactPII replaces matched byte ranges with [PII:name] tokens.
// Overlapping ranges are skipped to avoid double-replacement.
func redactPII(text string, ranges []piiMatchRange) string {
	var b strings.Builder
	pos := 0
	for _, r := range ranges {
		if r.start < pos {
			continue // skip overlap
		}
		b.WriteString(text[pos:r.start])
		b.WriteString("[PII:")
		b.WriteString(r.name)
		b.WriteString("]")
		pos = r.end
	}
	b.WriteString(text[pos:])
	return b.String()
}
