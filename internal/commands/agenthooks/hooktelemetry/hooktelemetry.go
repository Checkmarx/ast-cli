// Package hooktelemetry adds the Dev Assist analytics fields to the telemetry events the agent hooks
// send (scan and remediation). The plugin that launches `cx hooks` passes what only it knows (agent
// version, plugin version, plugin type) through environment variables; the session id comes from the
// CLI config, and the repository URL from the hook's working directory.
package hooktelemetry

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	gogit "github.com/go-git/go-git/v5"
	"github.com/spf13/viper"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/hostenv"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	// StatusSuccess is the status sent for an event whose operation completed.
	StatusSuccess = "success"
	// EngineOSS identifies dependency scan, remediation, and ignore events.
	EngineOSS = "oss"
	// RemediationModeSelfHealing marks a remediation driven by a hook deny (the agent fixes its own edit).
	RemediationModeSelfHealing = "SelfHealing"

	remoteName      = "origin"
	schemeSeparator = "://"
)

// severityOrder is the order severities appear in a scan event's statusResult.
var severityOrder = []string{"Critical", "High", "Medium", "Low", "Info"}

// maxFindingIDs caps how many IDs one remediation event carries.
const maxFindingIDs = 10

var (
	repoURLMu    sync.Mutex
	repoURLCache = map[string]string{}

	remediationMu    sync.Mutex
	remediationIDs   string
	remediationCount int
)

// FormatFindingIDs returns a comma-separated list of up to maxFindingIDs unique, nonempty IDs.
func FormatFindingIDs(ids []string) string {
	seen := make(map[string]bool, len(ids))
	kept := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		kept = append(kept, id)
		if len(kept) == maxFindingIDs {
			break
		}
	}
	return strings.Join(kept, ",")
}

// SetRemediationFindings records the findings covered by the current hook's remediation instructions.
// The count includes every covered finding even if its ID is missing, repeated, or omitted by the cap.
func SetRemediationFindings(ids []string, count int) {
	formattedIDs := FormatFindingIDs(ids)
	remediationMu.Lock()
	defer remediationMu.Unlock()
	remediationIDs = formattedIDs
	remediationCount = count
}

// TakeRemediationFindings returns and clears the recorded IDs and count.
func TakeRemediationFindings() (ids string, count int) {
	remediationMu.Lock()
	defer remediationMu.Unlock()
	ids, count = remediationIDs, remediationCount
	remediationIDs, remediationCount = "", 0
	return ids, count
}

// ignoreFlagValue keeps values safe in the optional flag parser and in generated shell commands.
var ignoreFlagValue = regexp.MustCompile(`^[A-Za-z0-9._:/+@-]+$`)

// IgnoreFlags carries the hook's attribution to the separate ignore command.
func IgnoreFlags(agent, sessionID string) string {
	if !ignoreFlagValue.MatchString(agent) {
		return ""
	}
	pairs := []string{"aiProvider=" + agent, "agent=" + hostenv.TelemetryAgent(agent)}
	for _, field := range []struct{ key, value string }{
		{"aiAgentSessionId", sessionID},
		{"agentVersion", os.Getenv(params.AgentVersionEnv)},
		{"cxPluginVersion", os.Getenv(params.CxPluginVersionEnv)},
		{"pluginType", os.Getenv(params.PluginTypeEnv)},
	} {
		value := strings.TrimSpace(field.value)
		if ignoreFlagValue.MatchString(value) {
			pairs = append(pairs, field.key+"="+value)
		}
	}
	return fmt.Sprintf(" --optional-flags %q", strings.Join(pairs, ";"))
}

// Enrich fills the fields every hook telemetry event shares. It never fails: a value that cannot be
// determined is left empty and the JSON omits it.
func Enrich(data *wrappers.DataForAITelemetry, workDir string) {
	if data == nil {
		return
	}
	data.AgentVersion = strings.TrimSpace(os.Getenv(params.AgentVersionEnv))
	data.CxPluginVersion = strings.TrimSpace(os.Getenv(params.CxPluginVersionEnv))
	data.PluginType = strings.TrimSpace(os.Getenv(params.PluginTypeEnv))
	data.SessionID = viper.GetString(params.SessionIDConfigKey)
	data.RepoURL = RepoURL(workDir)
	if data.Status == "" {
		data.Status = StatusSuccess
	}
}

// ScanSeveritySummary returns the highest detected severity and the counts for statusResult.
// Severities outside the known five are not counted.
func ScanSeveritySummary(severities []string) (highest, statusResult string) {
	counts := make(map[string]int, len(severityOrder))
	for _, s := range severities {
		for _, known := range severityOrder {
			if strings.EqualFold(s, known) {
				counts[known]++
				break
			}
		}
	}
	parts := make([]string, 0, len(severityOrder))
	for _, s := range severityOrder {
		count := counts[s]
		if highest == "" && count > 0 {
			highest = s
		}
		parts = append(parts, fmt.Sprintf("%s=%d", s, count))
	}
	return highest, strings.Join(parts, ",")
}

// RepoURL returns the credential-free "origin" URL of the git repository containing workDir, or "" when
// there is none. The result is cached per workDir for the life of the process.
func RepoURL(workDir string) string {
	if workDir == "" {
		return ""
	}
	repoURLMu.Lock()
	defer repoURLMu.Unlock()
	if u, ok := repoURLCache[workDir]; ok {
		return u
	}
	u := SanitizeRepoURL(originURL(workDir))
	repoURLCache[workDir] = u
	return u
}

func originURL(workDir string) string {
	repo, err := gogit.PlainOpenWithOptions(workDir, &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return ""
	}
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return ""
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// SanitizeRepoURL removes every credential from a git remote URL so it is safe to send: the userinfo of
// any scheme (tokens, passwords and usernames), the query and the fragment. An SCP-style remote
// (git@host:org/repo) becomes https://host/org/repo. A local path, or anything unrecognised, returns "".
func SanitizeRepoURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var host, path string
	if i := strings.Index(raw, schemeSeparator); i >= 0 {
		rest := raw[i+len(schemeSeparator):]
		if at := strings.LastIndexByte(rest, '@'); at >= 0 {
			rest = rest[at+1:]
		}
		host, path, _ = strings.Cut(rest, "/")
		path = "/" + path
	} else {
		// SCP-style: [user@]host:path. A Windows drive (C:\...) or a bare path has no host part.
		colon := strings.IndexByte(raw, ':')
		if colon <= 1 || strings.ContainsAny(raw[:colon], `/\`) {
			return ""
		}
		host = raw[:colon]
		if at := strings.LastIndexByte(host, '@'); at >= 0 {
			host = host[at+1:]
		}
		path = "/" + strings.TrimPrefix(raw[colon+1:], "/")
	}
	if j := strings.IndexAny(path, "?#"); j >= 0 {
		path = path[:j]
	}
	// Drop a port so ssh and https remotes of the same repository match.
	if k := strings.LastIndexByte(host, ':'); k >= 0 {
		host = host[:k]
	}
	if host == "" {
		return ""
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	return "https://" + host + path
}
