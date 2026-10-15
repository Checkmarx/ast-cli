// Package scannerconfig reads the per-scanner on/off switches a developer sets in
// ~/.checkmarx/cx-onboarding.properties (seeded by the agent plugins). Every scanner is ENABLED
// unless its key is explicitly set to a false-like value, and any read problem also means enabled:
// the switches can only narrow what the hooks do, never make them block.
package scannerconfig

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Scanner identifies one guardrail engine.
type Scanner string

const (
	// ASCA is the key for the ASCA (source code) scanner switch.
	ASCA Scanner = "cx_asca_scanner_enabled"
	// SCA is the key for the SCA (open source packages) scanner switch.
	SCA Scanner = "cx_sca_scanner_enabled"
	// KICS is the key for the KICS (IaC) scanner switch.
	KICS Scanner = "cx_kics_scanner_enabled"
	// Secrets is the key for the secrets scanner switch.
	Secrets Scanner = "cx_secret_scanner_enabled"
)

const (
	fileName = "cx-onboarding.properties"
	// maxBytes mirrors the plugins' reader: an oversized file is treated as unreadable (all enabled).
	maxBytes = 8192
	utf8BOM  = "\xef\xbb\xbf"
)

// Path returns ~/.checkmarx/cx-onboarding.properties, or "" when the home directory is unknown.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".checkmarx", fileName)
}

// Enabled reports whether the scanner is switched on in the user's config file.
func Enabled(s Scanner) bool {
	return enabledIn(Path(), s)
}

func enabledIn(path string, s Scanner) bool {
	value, found := valueIn(path, string(s))
	return !found || !isFalse(value)
}

// valueIn returns the last `key=value` for key in the file, or found=false when the file is missing,
// unreadable, oversized or does not set the key. Same parsing rules as the plugins' readers.
func valueIn(path, key string) (value string, found bool) {
	if path == "" {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil || len(raw) > maxBytes {
		return "", false
	}
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(string(raw), utf8BOM)))
	for sc.Scan() { // last occurrence wins
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		value, found = v, true
	}
	return value, found
}

func isFalse(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no", "off", "disabled":
		return true
	}
	return false
}

const thresholdKey = "cx_severity_threshold"

// severityRank orders the severities the guardrails report. Anything not listed (an empty string, a
// package status such as "Malicious", an unexpected label) has no rank and is never relaxed.
var severityRank = map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}

// Threshold returns the minimum severity rank (1 low … 4 critical) a finding needs to block.
// Blank, missing or invalid means low, so nothing is filtered.
func Threshold() int {
	return thresholdIn(Path())
}

func thresholdIn(path string) int {
	value, found := valueIn(path, thresholdKey)
	if !found {
		return 1
	}
	if rank, ok := severityRank[strings.ToLower(strings.TrimSpace(value))]; ok {
		return rank
	}
	return 1
}

// BelowThreshold reports whether a deny whose highest severity is `severity` should be let through
// because it is under cx_severity_threshold. An unranked severity returns false (keep blocking).
func BelowThreshold(severity string) bool {
	rank, ok := severityRank[strings.ToLower(strings.TrimSpace(severity))]
	return ok && rank < Threshold()
}
