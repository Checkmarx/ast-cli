// Package hooktelemetry adds the Dev Assist analytics fields to the telemetry events the agent hooks
// send (scan and remediation). The plugin that launches `cx hooks` passes what only it knows (agent
// version, plugin version, plugin type) through environment variables; the session id comes from the
// CLI config, and the repository URL from the hook's working directory.
package hooktelemetry

import (
	"fmt"
	"os"
	"strings"
	"sync"

	gogit "github.com/go-git/go-git/v5"
	"github.com/spf13/viper"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	// StatusSuccess is the status sent for an event whose operation completed.
	StatusSuccess = "success"
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

	findingIDsMu sync.Mutex
	findingIDs   []string
)

// SetFindingIDs records the IDs (CVE, rule id, ...) of the findings a guardrail is about to deny, so
// the remediation event for that deny can report them as eventAttributes.vulnerabilityId. `cx hooks`
// handles one hook event per process, so the last recorded set always belongs to the current deny.
// Empty IDs and duplicates are dropped; at most maxFindingIDs are kept.
func SetFindingIDs(ids []string) {
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
	findingIDsMu.Lock()
	findingIDs = kept
	findingIDsMu.Unlock()
}

// TakeFindingIDs returns the IDs recorded by SetFindingIDs, comma-separated ("" when none), and clears
// them so a later event in the same process never reuses them.
func TakeFindingIDs() string {
	findingIDsMu.Lock()
	defer findingIDsMu.Unlock()
	ids := strings.Join(findingIDs, ",")
	findingIDs = nil
	return ids
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

// SeverityCounts renders severities as "Critical=n,High=n,Medium=n,Low=n,Info=n", the scan event's
// statusResult. Severities outside the known five are not counted.
func SeverityCounts(severities []string) string {
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
		parts = append(parts, fmt.Sprintf("%s=%d", s, counts[s]))
	}
	return strings.Join(parts, ",")
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
