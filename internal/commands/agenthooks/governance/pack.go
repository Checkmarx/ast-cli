package governance

import (
	"encoding/json"
	"os"
	"regexp"
	"time"
)

// GovernancePack is the local policy file synced from the Policy API Server.
// It is the sole input to every governance decision — no network calls on the hot path.
type GovernancePack struct {
	PackVersion int          `json:"packVersion"`
	Policies    []PackPolicy `json:"policies"`
	PIIPolicy   *PIIPolicy   `json:"pii_policy,omitempty"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	patterns    []*regexp.Regexp // pre-compiled at load time; nil slot = malformed pattern (skip)
}

// PackPolicy is a single rule within the governance pack.
type PackPolicy struct {
	PolicyID     string `json:"policyId"`
	PolicyName   string `json:"policyName"`
	Version      int    `json:"policyVersion"`
	Type         string `json:"type"`               // "filesystem" | "shell" | "mcp" | "web_search" | "prompt"
	Action       string `json:"action"`             // "BLOCK" | "ALERT"
	MatchTarget  string `json:"matchTarget"`        // "path" | "command" | "mcp_tool" | "query" | "prompt"
	Pattern      string `json:"pattern"`            // regex applied to the match target value
	AgentMessage string `json:"agentMessage,omitempty"` // optional custom message shown in the agent
}

// Decision is the outcome of evaluating a RuntimeEvent against the GovernancePack.
type Decision struct {
	Action        string // "ALLOW" | "ALERT" | "BLOCK"
	PolicyID      string
	PolicyName    string
	AgentMessage  string
	PolicyVersion int
	PackVersion   int
	MatchedOn     string // which matchTarget triggered the match: "path" | "command" | "mcp_tool" | "query" | "prompt"
	MatchedValue  string // the actual value that matched
	LatencyMs     int64  // time from event receipt to verdict, measured by the handler
}

// Message returns the text shown to the developer inside the agent on BLOCK or ALERT.
// Uses AgentMessage when set by the admin; falls back to PolicyName.
func (d Decision) Message() string {
	if d.AgentMessage != "" {
		return d.AgentMessage
	}
	return d.PolicyName
}

// Load reads the local policy-pack.json and pre-compiles all policy regexes.
// Returns nil when the file is absent or malformed; callers treat nil as fail-open (ALLOW everything).
func Load() *GovernancePack {
	data, err := os.ReadFile(localPackPath())
	if err != nil {
		return nil
	}
	var pack GovernancePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil
	}
	pack.compilePatterns()
	return &pack
}

// compilePatterns pre-compiles every policy's regex pattern.
// Malformed patterns compile to nil — matchPolicy skips nil entries, so the
// rest of the pack remains functional even if one policy has a bad regex.
func (p *GovernancePack) compilePatterns() {
	p.patterns = make([]*regexp.Regexp, len(p.Policies))
	for i, pol := range p.Policies {
		re, err := regexp.Compile(pol.Pattern)
		if err != nil {
			p.patterns[i] = nil // malformed pattern: skip on evaluation
			continue
		}
		p.patterns[i] = re
	}
}

// Evaluate runs each policy in order against the event and returns the first match.
// Returns ALLOW when no policy matches or the pack is empty.
// ev.Tool.Category must be "prompt" for prompt.submit events.
// Compiles patterns on first call when the pack was constructed without Load().
func (p *GovernancePack) Evaluate(ev RuntimeEvent) Decision {
	if p.patterns == nil {
		p.compilePatterns()
	}
	for i := range p.Policies {
		pol := &p.Policies[i]
		if pol.Type != ev.Tool.Category {
			continue
		}
		re := p.patterns[i]
		if re == nil {
			continue // malformed regex: skip without crashing
		}
		if matched, value := matchPolicy(pol, re, ev); matched {
			return Decision{
				Action:        pol.Action,
				PolicyID:      pol.PolicyID,
				PolicyName:    pol.PolicyName,
				AgentMessage:  pol.AgentMessage,
				PolicyVersion: pol.Version,
				PackVersion:   p.PackVersion,
				MatchedOn:     pol.MatchTarget,
				MatchedValue:  value,
			}
		}
	}
	return Decision{Action: "ALLOW"}
}

// matchPolicy returns (true, matchedValue) when the pre-compiled regex matches the relevant event field.
// Filesystem events check all paths so a BLOCK fires if any single path is restricted.
func matchPolicy(pol *PackPolicy, re *regexp.Regexp, ev RuntimeEvent) (bool, string) {
	switch pol.MatchTarget {
	case "path":
		for _, p := range ev.Tool.Paths {
			if re.MatchString(p) {
				return true, p
			}
		}
	case "command":
		if ev.Tool.Command != "" && re.MatchString(ev.Tool.Command) {
			return true, ev.Tool.Command
		}
	case "mcp_tool":
		if ev.Tool.MCP != nil {
			target := ev.Tool.MCP.Server + "/" + ev.Tool.MCP.Name
			if re.MatchString(target) {
				return true, target
			}
		}
	case "query":
		if ev.Tool.Query != "" && re.MatchString(ev.Tool.Query) {
			return true, ev.Tool.Query
		}
	case "prompt":
		// Prompt content is never evaluated for PII reasons. The Pattern field acts
		// as a sentinel: any pattern that matches the literal string "prompt" activates
		// the rule when a non-empty prompt is submitted. Admins should use ".*" or "prompt".
		if ev.Prompt.Length > 0 && re.MatchString("prompt") {
			return true, "prompt"
		}
	}
	return false, ""
}

// readLocalPackVersion returns the current packVersion from the local pack file.
// Returns -1 when the file cannot be read or parsed.
func readLocalPackVersion() int {
	pack := Load()
	if pack == nil {
		return -1
	}
	return pack.PackVersion
}
