package guardrails

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
)

// utf8BOM is the byte-order mark some editors (notably PowerShell's
// Out-File/Set-Content defaults) prepend to UTF-8 files. encoding/json
// rejects it, so we strip it before unmarshalling.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// DenyMessage is the firm instruction appended to every denial. It tells the
// agent to stop — no retries, no workarounds, no alternative approaches.
const DenyMessage = "\n\n" +
	"IMPORTANT: This action was blocked by your organization's security policy. " +
	"Do NOT attempt alternative commands, workarounds, or equivalent approaches to achieve the same result. " +
	"Any alternative method to accomplish this blocked action may be equally harmful to the system and the developer. " +
	"Instead, inform the user that this operation is restricted by organizational policy and suggest they contact their administrator if they need this action performed."

// HooksPolicy is the root shape of ~/.checkmarx/policyhooks.json
type HooksPolicy struct {
	DefaultPolicy DefaultPolicy `json:"default_policy"`
	Tools         ToolsPolicy   `json:"tools"`
}

// DefaultPolicy holds all policy sections that apply at the global scope.
type DefaultPolicy struct {
	BlacklistTools struct {
		Enabled bool              `json:"enabled"`
		Tools   []BlacklistedTool `json:"tools"`
	} `json:"blacklist_tools"`
	RestrictedDirectories PathPolicy       `json:"restricted_directories"`
	RestrictedFiles       PathPolicy       `json:"restricted_files"`
	AllowedDirectories    PathPolicy       `json:"allowed_directories"`
	AllowedFiles          PathPolicy       `json:"allowed_files"`
	ContextPolicy         ContextPolicy    `json:"context_policy"`
	BlastRadiusLimit      BlastRadiusLimit `json:"blast_radius_limit"`
}

// ContextPolicy controls what data may enter the AI's context window.
type ContextPolicy struct {
	Enabled           bool              `json:"enabled"`
	FilesLimits       FilesLimits       `json:"files_limits"`
	ContentScanning   ContentScanning   `json:"content_scanning"`
	BlockedExtensions BlockedExtensions `json:"blocked_extensions"`
}

// FilesLimits restricts how many (and how large) files may be referenced in an AI context.
type FilesLimits struct {
	Enabled            bool `json:"enabled"`
	MaxFileCount       int  `json:"max_file_count"`
	MaxFileSizeKB      int  `json:"max_file_size_kb"`
	MaxTotalFileSizeKB int  `json:"max_total_file_size_kb"`
}

// BlockedExtensions lists file extensions that must never enter the AI context.
type BlockedExtensions struct {
	Enabled    bool     `json:"enabled"`
	Extensions []string `json:"extensions"`
}

// BlastRadiusLimit caps how many files the AI may write during a single session.
type BlastRadiusLimit struct {
	Enabled   bool `json:"enabled"`
	Threshold int  `json:"threshold"`
}

// ContentScanning holds the scanning configuration.
type ContentScanning struct {
	Enabled  bool                 `json:"enabled"`
	Patterns []ContentScanPattern `json:"patterns"`
}

// ContentScanPattern is a single regex rule that blocks sensitive content in prompts.
type ContentScanPattern struct {
	ID          string `json:"id"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
}

// PathPolicy holds OS-specific path lists for restricted or allowed files/directories.
type PathPolicy struct {
	Enabled bool     `json:"enabled"`
	Linux   []string `json:"linux"`
	Windows []string `json:"windows"`
	Mac     []string `json:"mac"`
}

// BlacklistedTool is a single entry in the shell command blacklist.
type BlacklistedTool struct {
	Name     string   `json:"name"`
	OS       []string `json:"os"`
	Category string   `json:"category"`
	Risk     string   `json:"risk"`
}

// ToolsPolicy is the root of the per-tool rule section.
type ToolsPolicy struct {
	Enabled         bool       `json:"enabled"`
	DefaultAuditLog bool       `json:"default_audit_log"`
	Rules           []ToolRule `json:"rules"`
}

// ToolRule defines restrictions and permissions for a specific shell tool.
// Enabled uses *bool so a missing field (nil) is treated as active, preserving
// backward-compatibility with older policy files that don't set the flag.
type ToolRule struct {
	Enabled               *bool         `json:"enabled,omitempty"`
	ID                    string        `json:"id"`
	Tool                  []string      `json:"tool"`
	OS                    []string      `json:"os"`
	ArgsInclude           []string      `json:"args_include"`
	ArgsExclude           []string      `json:"args_exclude"`
	RestrictedDirectories PathPolicy    `json:"restricted_directories"`
	RestrictedFiles       PathPolicy    `json:"restricted_files"`
	AllowedDirectories    PathPolicy    `json:"allowed_directories"`
	AllowedFiles          PathPolicy    `json:"allowed_files"`
	MergeStrategy         MergeStrategy `json:"merge_strategy"`
	AuditLog              bool          `json:"audit_log"`
}

// MergeStrategy controls how a tool rule's path lists are combined with the
// global default_policy values. Applied independently per field.
//
//	merge    = global ∪ rule list
//	override = rule list only (replaces global)
//	default  = global list only (rule's values ignored)
type MergeStrategy struct {
	RestrictedDirectories string `json:"restricted_directories"`
	RestrictedFiles       string `json:"restricted_files"`
	AllowedDirectories    string `json:"allowed_directories"`
	AllowedFiles          string `json:"allowed_files"`
}

// blastRadiusCount tracks how many files have been written during this session.
// Kept as a package-level atomic so concurrent BeforeFileEdit handlers are safe.
var blastRadiusCount int32

// totalFileSizeBytes accumulates the byte length of all proposed file edits this session.
// Incremented in BeforeFileEdit before any bytes are written to disk.
var totalFileSizeBytes int64

// ResetBlastRadiusCount resets the session-level file write counter. Exposed for tests.
func ResetBlastRadiusCount() {
	atomic.StoreInt32(&blastRadiusCount, 0)
}

// ResetTotalFileSizeCount resets the session-level total bytes counter. Exposed for tests.
func ResetTotalFileSizeCount() {
	atomic.StoreInt64(&totalFileSizeBytes, 0)
}

// CheckAndIncrementBlastRadius increments the file-write counter and returns
// blocked=true with a reason if the configured threshold has been exceeded.
func CheckAndIncrementBlastRadius() (blocked bool, reason string) {
	limit := LoadBlastRadiusLimit()
	if limit == nil || !limit.Enabled || limit.Threshold <= 0 {
		return false, ""
	}
	count := int(atomic.AddInt32(&blastRadiusCount, 1))
	if count > limit.Threshold {
		return true, fmt.Sprintf(
			"Blocked by Checkmarx: blast radius limit exceeded. "+
				"This session has written %d files, exceeding the policy threshold of %d.%s",
			count, limit.Threshold, DenyMessage,
		)
	}
	return false, ""
}

// CheckAndIncrementTotalFileSize adds sizeBytes to the running total and returns
// blocked=true if the configured max_total_file_size_kb would be exceeded.
func CheckAndIncrementTotalFileSize(sizeBytes int64) (blocked bool, reason string) {
	limits := LoadFilesLimits()
	if limits == nil || !limits.Enabled || limits.MaxTotalFileSizeKB <= 0 {
		return false, ""
	}
	limitBytes := int64(limits.MaxTotalFileSizeKB) * 1024
	newTotal := atomic.AddInt64(&totalFileSizeBytes, sizeBytes)
	if newTotal > limitBytes {
		return true, fmt.Sprintf(
			"Blocked by Checkmarx: total file size limit exceeded. "+
				"This session has written %d KB, exceeding the policy threshold of %d KB.%s",
			newTotal/1024, limits.MaxTotalFileSizeKB, DenyMessage,
		)
	}
	return false, ""
}

// ShellPolicyPath returns the path to the policy file: ~/.checkmarx/policyhooks.json
func ShellPolicyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".checkmarx", "policyhooks.json")
}

// LoadPolicy reads and parses ~/.checkmarx/policyhooks.json.
// Returns nil on any error (fail-open: a missing or malformed policy should never block the developer).
func LoadPolicy() *HooksPolicy {
	data, err := os.ReadFile(ShellPolicyPath())
	if err != nil {
		return nil
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	var p HooksPolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	return &p
}

// GetOSPaths returns the path list for the current OS from a PathPolicy entry.
func GetOSPaths(pp PathPolicy) []string {
	if !pp.Enabled {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		return pp.Linux
	case "darwin":
		return pp.Mac
	case "windows":
		return pp.Windows
	default:
		return nil
	}
}

// MatchesOS returns true when any of the tool's OS labels match the current OS.
func MatchesOS(toolOS []string, currentOS string) bool {
	for _, o := range toolOS {
		mapped := o
		if o == "mac" {
			mapped = "darwin"
		}
		if mapped == currentOS {
			return true
		}
	}
	return false
}

// LoadBlacklistedCommands reads the policy file and returns all command names
// (lowercased) that are blacklisted on the current OS, together with their metadata.
func LoadBlacklistedCommands() map[string]BlacklistedTool {
	blacklisted := map[string]BlacklistedTool{}
	policy := LoadPolicy()
	if policy == nil {
		return blacklisted // fail-open
	}
	if !policy.DefaultPolicy.BlacklistTools.Enabled {
		return blacklisted
	}
	for _, t := range policy.DefaultPolicy.BlacklistTools.Tools {
		if !MatchesOS(t.OS, runtime.GOOS) {
			continue
		}
		blacklisted[strings.ToLower(t.Name)] = t
	}
	return blacklisted
}

// LoadRestrictedPaths returns the OS-specific restricted file and directory
// lists from the policy file.
func LoadRestrictedPaths() (files []string, dirs []string) {
	policy := LoadPolicy()
	if policy == nil {
		return nil, nil
	}
	return GetOSPaths(policy.DefaultPolicy.RestrictedFiles),
		GetOSPaths(policy.DefaultPolicy.RestrictedDirectories)
}

// LoadEffectiveRestrictedPaths returns the union of the global default
// restricted_files / restricted_directories and each enabled tool rule's
// effective restricted lists, combined per that rule's merge_strategy.
//
// Used by prompt-side checks where no specific tool is matched but any tool
// rule's restriction may still be relevant. Rules disabled, scoped to other
// OSes, or with merge_strategy == "default" contribute nothing beyond the
// global lists; "merge" rules contribute their entries; "override" rules
// contribute their entries (without re-adding global, since global is already
// included once).
func LoadEffectiveRestrictedPaths() (files []string, dirs []string) {
	globalFiles, globalDirs := LoadRestrictedPaths()

	seenF := map[string]struct{}{}
	seenD := map[string]struct{}{}
	add := func(seen map[string]struct{}, dst *[]string, src []string) {
		for _, s := range src {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			*dst = append(*dst, s)
		}
	}
	add(seenF, &files, globalFiles)
	add(seenD, &dirs, globalDirs)

	policy := LoadPolicy()
	if policy == nil || !policy.Tools.Enabled {
		return files, dirs
	}
	for i := range policy.Tools.Rules {
		rule := &policy.Tools.Rules[i]
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if len(rule.OS) > 0 && !MatchesOS(rule.OS, runtime.GOOS) {
			continue
		}
		ef := ResolveRestrictedPaths(globalFiles, GetOSPaths(rule.RestrictedFiles), rule.MergeStrategy.RestrictedFiles)
		ed := ResolveRestrictedPaths(globalDirs, GetOSPaths(rule.RestrictedDirectories), rule.MergeStrategy.RestrictedDirectories)
		add(seenF, &files, ef)
		add(seenD, &dirs, ed)
	}
	return files, dirs
}

// LoadAllowedPaths returns the OS-specific allowed file and directory
// lists from the policy file.
func LoadAllowedPaths() (files []string, dirs []string) {
	policy := LoadPolicy()
	if policy == nil {
		return nil, nil
	}
	return GetOSPaths(policy.DefaultPolicy.AllowedFiles),
		GetOSPaths(policy.DefaultPolicy.AllowedDirectories)
}

// LoadBlastRadiusLimit returns the blast-radius limit config, or nil if disabled / absent.
func LoadBlastRadiusLimit() *BlastRadiusLimit {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	limit := policy.DefaultPolicy.BlastRadiusLimit
	if !limit.Enabled {
		return nil
	}
	return &limit
}

// LoadBlockedExtensions returns the list of file extensions blocked from AI context.
// Returns nil when the feature is disabled or the policy is absent.
func LoadBlockedExtensions() []string {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	cp := policy.DefaultPolicy.ContextPolicy
	if !cp.Enabled || !cp.BlockedExtensions.Enabled {
		return nil
	}
	return cp.BlockedExtensions.Extensions
}

// LoadFilesLimits returns the files-limits config, or nil if disabled / absent.
func LoadFilesLimits() *FilesLimits {
	policy := LoadPolicy()
	if policy == nil {
		return nil
	}
	cp := policy.DefaultPolicy.ContextPolicy
	if !cp.Enabled || !cp.FilesLimits.Enabled {
		return nil
	}
	fl := cp.FilesLimits
	return &fl
}

// FindMatchingToolRule returns the first tool rule whose tool list contains a
// command name appearing anywhere in the command as a whole token, and whose
// OS list matches the current OS. Returns nil if no rule matches, the tools
// section is disabled, or the rule is explicitly disabled.
//
// Scanning the whole command (not just fields[0]) is what lets compound
// invocations like `cd /foo && mvn deploy` match the `mvn` rule so its
// args_exclude can be enforced.
func FindMatchingToolRule(command string) *ToolRule {
	policy := LoadPolicy()
	if policy == nil || !policy.Tools.Enabled {
		return nil
	}
	if strings.TrimSpace(command) == "" {
		return nil
	}
	cmdLower := strings.ToLower(command)
	for i := range policy.Tools.Rules {
		rule := &policy.Tools.Rules[i]
		// Explicit `enabled: false` disables a rule; nil (absent) keeps it active.
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if len(rule.OS) > 0 && !MatchesOS(rule.OS, runtime.GOOS) {
			continue
		}
		for _, name := range rule.Tool {
			if containsAsToken(cmdLower, strings.ToLower(name)) {
				return rule
			}
		}
	}
	return nil
}

// containsAsToken reports whether needle appears in haystack as a whole token —
// flanked on both sides by start-of-string, end-of-string, or any non-word byte
// (space, ;, &, |, /, \, ", ', etc.). Prevents false positives such as matching
// the tool name "mvn" inside paths like "/opt/foo-mvn-bar/".
// Both inputs MUST already be lowercased.
func containsAsToken(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	start := 0
	for start <= len(haystack)-len(needle) {
		idx := strings.Index(haystack[start:], needle)
		if idx < 0 {
			return false
		}
		absolute := start + idx
		if isTokenBoundary(haystack, absolute, absolute+len(needle)) {
			return true
		}
		start = absolute + 1
	}
	return false
}

// isTokenBoundary reports whether s[lo:hi] is bounded on both sides by a
// non-word byte (or start/end of string). Word bytes are a-z, 0-9, '_', '-'.
// The dash is treated as part of a word so "mvn" does NOT match inside
// "foo-mvn" or "mvn-helper".
func isTokenBoundary(s string, lo, hi int) bool {
	isWordByte := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
	}
	if lo > 0 && isWordByte(s[lo-1]) {
		return false
	}
	if hi < len(s) && isWordByte(s[hi]) {
		return false
	}
	return true
}

// ResolveAllowedPaths combines globalPaths and rulePaths according to strategy.
// Valid strategies: "merge", "override", "default" (anything else acts as "default").
func ResolveAllowedPaths(globalPaths, rulePaths []string, strategy string) []string {
	switch strategy {
	case "merge":
		seen := map[string]struct{}{}
		result := make([]string, 0, len(globalPaths)+len(rulePaths))
		for _, p := range append(globalPaths, rulePaths...) {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				result = append(result, p)
			}
		}
		return result
	case "override":
		return rulePaths
	default: // "default" or anything unrecognised
		return globalPaths
	}
}

// ResolveRestrictedPaths combines global and tool-level restricted paths per strategy.
// Semantics are identical to ResolveAllowedPaths; the two names exist for readability
// at call sites that work with different path categories.
func ResolveRestrictedPaths(globalPaths, rulePaths []string, strategy string) []string {
	return ResolveAllowedPaths(globalPaths, rulePaths, strategy)
}

// NormalizeWorkspaceRoot canonicalises a workspace root so it can be compared
// against policy path entries. Cursor reports Windows roots as "/c:/foo/bar";
// strip the leading slash before a drive letter so PathUnderAny's prefix match
// lines up with policy entries like "C:\\foo\\bar\\".
func NormalizeWorkspaceRoot(root string) string {
	r := filepath.ToSlash(root)
	if len(r) >= 3 && r[0] == '/' && isASCIILetter(r[1]) && r[2] == ':' {
		r = r[1:]
	}
	return r
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
