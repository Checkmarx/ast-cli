package utils

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	gitConfigTimeout = 2 * time.Second
	urlSplitParts    = 2
	gitHubPathParts  = 2
	gitHubHost       = "github.com"
)

// NormalizeGitHubURL converts a GitHub remote (HTTPS or SSH) into
// https://github.com/owner/repo. Credentials are dropped. A non-GitHub URL returns "".
func NormalizeGitHubURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(toHTTPS(raw))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	if host != gitHubHost {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < gitHubPathParts || parts[0] == "" || parts[1] == "" {
		return ""
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" {
		return ""
	}
	return "https://" + gitHubHost + "/" + parts[0] + "/" + repo
}

// GitHubRemoteURL returns the canonical GitHub origin URL for the repository
// that contains dir. An empty dir uses the current working directory.
// A directory that is not a GitHub checkout returns "".
func GitHubRemoteURL(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		dir = wd
	}
	return gitHubRemoteAt(dir)
}

// ResolveGitHubURL prefers an explicit URL from the caller. Recognized GitHub
// remotes are canonicalized; any other explicit value is kept as given.
// When explicit is empty, the origin remote of dir is used.
func ResolveGitHubURL(explicit, dir string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit == "" {
		return GitHubRemoteURL(dir)
	}
	if canonical := NormalizeGitHubURL(explicit); canonical != "" {
		return canonical
	}
	return explicit
}

func gitHubRemoteAt(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitConfigTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "config", "--get", "remote.origin.url")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return NormalizeGitHubURL(string(out))
}

// toHTTPS rewrites common git remote spellings into an https URL string
// that net/url can parse. Already-https values are returned unchanged.
func toHTTPS(raw string) string {
	if !strings.Contains(raw, "://") && strings.Contains(raw, "@") && strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, "@", urlSplitParts)
		if len(parts) == urlSplitParts {
			hostAndPath := strings.SplitN(parts[1], ":", urlSplitParts)
			if len(hostAndPath) == urlSplitParts {
				return "https://" + hostAndPath[0] + "/" + hostAndPath[1]
			}
		}
	}
	if strings.HasPrefix(raw, "ssh://") {
		rest := strings.TrimPrefix(raw, "ssh://")
		if i := strings.Index(rest, "@"); i >= 0 {
			rest = rest[i+1:]
		}
		return "https://" + rest
	}
	if strings.HasPrefix(raw, "git://") {
		return "https://" + strings.TrimPrefix(raw, "git://")
	}
	return raw
}
