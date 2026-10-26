// Package scannerconfig reads the per-scanner on/off switches a developer sets in
// ~/.checkmarx/cx-onboarding.properties (seeded by the agent plugins). Every scanner is ENABLED
// unless its key is explicitly set to a false-like value, and any read problem also means enabled:
// the switches can only narrow what the hooks do, never make them block.
//
// GitHub Copilot agent mode in VS Code has its own state file,
// ~/.checkmarx/cx-onboarding-ide.properties, written by the VS Code extension: the Self-Healing
// switch (cx_self_healing_enabled) and the scanner switches the VS Code settings set. It is read
// only for events from VS Code; the terminal hosts never read it.
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
	// ideFileName is the VS Code extension's state file (see IDEPath).
	ideFileName = "cx-onboarding-ide.properties"
	// maxBytes mirrors the plugins' reader: an oversized file is treated as unreadable (all enabled).
	maxBytes = 8192
	utf8BOM  = "\xef\xbb\xbf"

	ideSchemaKey      = "cx_schema_version"
	ideSchemaVersion  = "1"
	ideSelfHealingKey = "cx_self_healing_enabled"
)

// Path returns ~/.checkmarx/cx-onboarding.properties, or "" when the home directory is unknown.
func Path() string {
	return homeFile(fileName)
}

// IDEPath returns ~/.checkmarx/cx-onboarding-ide.properties, or "" when the home directory is unknown.
func IDEPath() string {
	return homeFile(ideFileName)
}

func homeFile(name string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".checkmarx", name)
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
	props, ok := readProps(path)
	if !ok {
		return "", false
	}
	value, found = props[key]
	return value, found
}

// readProps parses the file's `key=value` lines (last occurrence wins, # comments, BOM stripped).
// ok is false when the file is missing, unreadable or oversized.
func readProps(path string) (props map[string]string, ok bool) {
	if path == "" {
		return nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil || len(raw) > maxBytes {
		return nil, false
	}
	props = map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(string(raw), utf8BOM)))
	for sc.Scan() { // last occurrence wins
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		props[strings.TrimSpace(k)] = v
	}
	return props, true
}

// ideProps returns the VS Code state file's keys when the file is valid: it exists, is readable, is at
// most 8 KB, and its cx_schema_version is absent or 1. An invalid file returns ok=false.
func ideProps(path string) (props map[string]string, ok bool) {
	props, ok = readProps(path)
	if !ok {
		return nil, false
	}
	if v, set := props[ideSchemaKey]; set && strings.TrimSpace(v) != ideSchemaVersion {
		return nil, false
	}
	return props, true
}

// SelfHealingEnabled reports whether Checkmarx Self-Healing is on in VS Code. It is OFF only when the
// VS Code state file is valid and cx_self_healing_enabled is a false-like value; a missing, unreadable
// or unknown-schema file, a missing key or any other value means ON.
func SelfHealingEnabled() bool {
	return selfHealingIn(IDEPath())
}

func selfHealingIn(path string) bool {
	props, ok := ideProps(path)
	if !ok {
		return true
	}
	v, set := props[ideSelfHealingKey]
	return !set || !isFalse(v)
}

// IDEEnabled is Enabled for events from VS Code: when the VS Code state file is valid its
// cx_*_scanner_enabled keys decide (a missing key means enabled); otherwise it falls back to the
// terminal's ~/.checkmarx/cx-onboarding.properties, exactly like Enabled.
func IDEEnabled(s Scanner) bool {
	return ideEnabledIn(IDEPath(), Path(), s)
}

func ideEnabledIn(idePath, cliPath string, s Scanner) bool {
	props, ok := ideProps(idePath)
	if !ok {
		return enabledIn(cliPath, s)
	}
	v, set := props[string(s)]
	return !set || !isFalse(v)
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
