package commands

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/logger"
)

const (
	// CheckmarxFolderName is the folder where CLI-generated metadata files are stored.
	CheckmarxFolderName = ".checkmarx"
	// ContributorsFileName is the filename for the CSV with contributor information.
	ContributorsFileName = "contributors.csv"
	// MetadataFileName is the filename for the JSON metadata file.
	MetadataFileName = "metadata.json"

	commitHistoryWindow     = 90 * 24 * time.Hour
	defaultRemoteName       = "origin"
	contributorsSizeLimit   = 1 * 1024 * 1024 // repostore ignores contributors.csv above this size
	privacyDetectionTimeout = 5               // seconds for privacy detection HTTP requests
	generatedFilePerm       = 0o644
	generatedDirPerm        = 0o755
	csvFieldCount           = 4
	urlSchemeParts          = 2 // parts when splitting by "://"
	pathParts               = 2 // parts when splitting by "/"
	sshSplitParts           = 2 // parts when splitting SSH URL by "@" or ":"
)

// contributorsMetadata mirrors repostore metadata structure; omits branchName per tech design.
type contributorsMetadata struct {
	RepositoryURL  string `json:"repositoryUrl"`
	LastCommitHash string `json:"lastCommitHash"`
	LastCommitDate string `json:"lastCommitDate"`
	CommitsCount   int    `json:"commitsCount"`
}

// GenerateAndWrite creates contributors.csv (private only) and metadata.json under .checkmarx/; errors should be logged but not fail scan.
func GenerateAndWrite(repoPath string, isPrivateRepo bool) error {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		// Fallback to system git if go-git fails (e.g., unsupported git extensions like worktreeConfig).
		if strings.Contains(err.Error(), "does not support extension") {
			logger.PrintfIfVerbose("go-git cannot open repository (unsupported git extension), falling back to system git")
			return generateViaSystemGit(repoPath, isPrivateRepo)
		}
		return errors.Wrap(err, "could not open local git repository")
	}

	headCommit, err := resolveHeadCommit(repo)
	if err != nil {
		return errors.Wrap(err, "could not resolve HEAD commit")
	}

	since := time.Now().Add(-commitHistoryWindow)
	commits, err := commitsSince(repo, since)
	if err != nil {
		// Fallback to system git for shallow clones or corrupted repos where go-git can't read objects
		if strings.Contains(err.Error(), "object not found") {
			logger.PrintfIfVerbose("go-git cannot read commit history (shallow clone or corrupted repo), falling back to system git")
			return generateViaSystemGit(repoPath, isPrivateRepo)
		}
		return errors.Wrap(err, "could not read commit history")
	}

	// Generate CSV only for private repos
	var csvData []byte
	if isPrivateRepo {
		csvData, err = buildContributorsCSV(commits)
		if err != nil {
			return errors.Wrap(err, "could not build contributors.csv")
		}
		if len(csvData) > contributorsSizeLimit {
			logger.PrintfIfVerbose("contributors.csv is %d bytes, over the 1MB size repostore accepts", len(csvData))
		}
	}

	// Always generate metadata.json (all repos)
	metadataData, err := buildMetadataJSON(remoteURL(repo), headCommit, len(commits))
	if err != nil {
		return errors.Wrap(err, "could not build metadata.json")
	}

	return writeGeneratedFilesConditional(repoPath, csvData, metadataData, isPrivateRepo)
}

// resolveHeadCommit resolves HEAD to its full commit object, not just the hash.
func resolveHeadCommit(repo *gogit.Repository) (*object.Commit, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	return repo.CommitObject(head.Hash())
}

// commitsSince returns commits from last 90 days (by author date, not committer date) sorted newest-first.
func commitsSince(repo *gogit.Repository, since time.Time) ([]*object.Commit, error) {
	iter, err := repo.Log(&gogit.LogOptions{})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var commits []*object.Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if !c.Author.When.Before(since) {
			commits = append(commits, c)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Author.When.After(commits[j].Author.When)
	})
	return commits, nil
}

// dedupByEmail keeps most recent commit per unique email to keep contributors.csv small.
func dedupByEmail(commits []*object.Commit) []*object.Commit {
	seen := make(map[string]bool, len(commits))
	deduped := make([]*object.Commit, 0, len(commits))
	for _, c := range commits {
		email := strings.ToLower(strings.TrimSpace(c.Author.Email))
		if seen[email] {
			continue
		}
		seen[email] = true
		deduped = append(deduped, c)
	}
	return deduped
}

// buildContributorsCSV writes one row per unique email: date, hash, email, username. No header row.
func buildContributorsCSV(commits []*object.Commit) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	for _, c := range dedupByEmail(commits) {
		row := []string{
			c.Author.When.Format(time.RFC3339),
			c.Hash.String(),
			c.Author.Email,
			c.Author.Name,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func buildMetadataJSON(repositoryURL string, headCommit *object.Commit, commitsCount int) ([]byte, error) {
	metadata := contributorsMetadata{
		RepositoryURL:  repositoryURL,
		LastCommitHash: headCommit.Hash.String(),
		LastCommitDate: headCommit.Author.When.Format(time.RFC3339),
		CommitsCount:   commitsCount,
	}
	return json.MarshalIndent(metadata, "", "  ")
}

// remoteURL returns the "origin" remote's first URL, or "" if there is none (e.g. a bare local clone).
func remoteURL(repo *gogit.Repository) string {
	remote, err := repo.Remote(defaultRemoteName)
	if err != nil {
		return ""
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// generateViaSystemGit extracts repo info via system git when go-git fails (e.g., unsupported extensions).
func generateViaSystemGit(repoPath string, isPrivateRepo bool) error {
	remoteURL, err := gitCommand(repoPath, "config", "--get", "remote.origin.url")
	if err != nil {
		return errors.Wrap(err, "could not get remote URL via system git")
	}

	headHash, err := gitCommand(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return errors.Wrap(err, "could not get HEAD commit via system git")
	}

	sinceCutoff := time.Now().Add(-commitHistoryWindow)
	sinceStr := sinceCutoff.Format(time.RFC3339)
	logOutput, err := gitCommand(repoPath, "log", "--since="+sinceStr, "--pretty=format:%aI%x1f%H%x1f%ae%x1f%an")
	if err != nil {
		return errors.Wrap(err, "could not get commit log via system git")
	}

	commits := parseGitLogOutput(logOutput, sinceCutoff)
	if len(commits) == 0 {
		commits = []map[string]string{} // empty list for builds with no commits in 90 days
	}

	// Extract the actual last commit date (newest commit is first in array)
	// If no commits in 90-day window, get HEAD date directly (consistent with go-git path)
	lastCommitDateStr := ""
	if len(commits) > 0 {
		lastCommitDateStr = commits[0]["date"]
	} else {
		// No commits in 90-day window, but HEAD still exists - get its date
		headDate, err := gitCommand(repoPath, "log", "-1", "--format=%aI", "HEAD")
		if err == nil && headDate != "" {
			lastCommitDateStr = headDate
		}
	}

	// Generate CSV only for private repos
	var csvData []byte
	if isPrivateRepo {
		csvData, err = buildContributorsCSV(convertToCoreCommits(commits))
		if err != nil {
			return errors.Wrap(err, "could not build contributors.csv")
		}
		if len(csvData) > contributorsSizeLimit {
			logger.PrintfIfVerbose("contributors.csv is %d bytes, over the 1MB size repostore accepts", len(csvData))
		}
	}

	// Always generate metadata.json
	metadataData, err := buildMetadataJSONFromSystem(remoteURL, headHash, len(commits), lastCommitDateStr)
	if err != nil {
		return errors.Wrap(err, "could not build metadata.json")
	}

	return writeGeneratedFilesConditional(repoPath, csvData, metadataData, isPrivateRepo)
}

// gitCommand runs a git command in the given repo directory and returns trimmed output.
func gitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "git command failed: %s", out.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// parseGitLogOutput parses git log output (date, hash, email, name) filtered by author date and returns newest-first.
func parseGitLogOutput(logOutput string, sinceCutoff time.Time) []map[string]string {
	if logOutput == "" {
		return nil
	}

	lines := strings.Split(logOutput, "\n")
	commits := make([]map[string]string, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x1f")
		if len(parts) != csvFieldCount {
			continue
		}

		// Filter by author date to correctly handle rebased/cherry-picked commits
		authorDate, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			// Skip commits with unparseable dates (shouldn't happen with %aI format)
			continue
		}
		if !sinceCutoff.IsZero() && authorDate.Before(sinceCutoff) {
			// Skip commits older than the cutoff
			continue
		}

		commits = append(commits, map[string]string{
			"date":  parts[0],
			"hash":  parts[1],
			"email": parts[2],
			"name":  parts[3],
		})
	}
	return commits
}

// convertToCoreCommits converts system-git commit maps to go-git Commit objects for CSV building.
func convertToCoreCommits(commits []map[string]string) []*object.Commit {
	result := make([]*object.Commit, 0, len(commits))
	for _, c := range commits {
		// Parse ISO8601 commit date; fall back to zero time on error.
		commitTime, _ := time.Parse(time.RFC3339, c["date"])
		// Parse the commit hash; fall back to zero hash on error.
		hash := plumbing.NewHash(c["hash"])
		commit := &object.Commit{
			Hash: hash,
			Author: object.Signature{
				Email: c["email"],
				Name:  c["name"],
				When:  commitTime,
			},
		}
		result = append(result, commit)
	}
	return result
}

// buildMetadataJSONFromSystem builds metadata.json using data from system git.
func buildMetadataJSONFromSystem(remoteURL, headHash string, commitCount int, lastCommitDate string) ([]byte, error) {
	metadata := contributorsMetadata{
		RepositoryURL:  remoteURL,
		LastCommitHash: headHash,
		LastCommitDate: lastCommitDate,
		CommitsCount:   commitCount,
	}
	return json.Marshal(metadata)
}

// writeGeneratedFilesConditional always writes metadata.json; only writes contributors.csv for private repos.
func writeGeneratedFilesConditional(repoPath string, csvData, metadataData []byte, isPrivateRepo bool) error {
	checkmarxDir := filepath.Join(repoPath, CheckmarxFolderName)
	if err := os.MkdirAll(checkmarxDir, generatedDirPerm); err != nil {
		return errors.Wrap(err, "could not create .checkmarx directory")
	}

	// ALWAYS write metadata.json (all repos)
	if err := os.WriteFile(filepath.Join(checkmarxDir, MetadataFileName), metadataData, generatedFilePerm); err != nil {
		return errors.Wrap(err, "could not write "+MetadataFileName)
	}

	// ONLY write contributors.csv for private repos
	if isPrivateRepo {
		if err := os.WriteFile(filepath.Join(checkmarxDir, ContributorsFileName), csvData, generatedFilePerm); err != nil {
			return errors.Wrap(err, "could not write "+ContributorsFileName)
		}
	}

	return nil
}

// detectRepositoryPrivacy determines if repo is private/public; conservatively defaults to private for unknown repos.
func detectRepositoryPrivacy(repoPath string, httpClient *http.Client) (isPrivate bool) {
	isPrivate = true

	defer func() {
		if r := recover(); r != nil {
			logger.PrintIfVerbose(fmt.Sprintf("Panic in repository privacy detection for %v, treating as PRIVATE", r))
			isPrivate = true
		}
	}()

	// Method 1: Try go-git
	repo, err := gogit.PlainOpen(repoPath)
	if err == nil {
		return isPrivateByURL(remoteURL(repo), httpClient)
	}

	// Method 2: Try system git
	remoteURL, err := gitCommand(repoPath, "config", "--get", "remote.origin.url")
	if err == nil && remoteURL != "" {
		return isPrivateByURL(remoteURL, httpClient)
	}

	// Default: Conservative (treat as private)
	return true
}

// isPrivateByURL detects repository privacy via public APIs (GitHub/GitLab/Bitbucket/Azure); defaults to private on any error.
func isPrivateByURL(remoteURL string, httpClient *http.Client) bool {
	if remoteURL == "" {
		return true // Local-only repo → private
	}

	urlLower := strings.ToLower(remoteURL)

	// Route to appropriate API handler based on platform
	switch {
	case strings.Contains(urlLower, "github.com"):
		return isPrivateGitHub(remoteURL, httpClient)
	case strings.Contains(urlLower, "gitlab"):
		return isPrivateGitLab(remoteURL, httpClient)
	case strings.Contains(urlLower, "bitbucket"):
		return isPrivateBitbucket(remoteURL, httpClient)
	case strings.Contains(urlLower, "dev.azure.com") || strings.Contains(urlLower, "visualstudio.com"):
		return isPrivateAzureDevOps(remoteURL, httpClient)
	default:
		return true // Unknown platform → conservative: default to PRIVATE
	}
}

// isPrivateGitHub checks GitHub repo privacy via direct HTTP URL (HTTP 200 = Public, else = Private).
func isPrivateGitHub(repoURL string, httpClient *http.Client) bool {
	owner, repo := extractGitHubOwnerRepo(repoURL)
	if owner == "" || repo == "" {
		return true
	}

	directURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	isPublic := isRepoPublic(directURL, httpClient)
	return !isPublic
}

// isPrivateGitLab checks gitlab.com repo privacy via HTTP (200 = Public, else = Private); self-hosted URLs default to private for security.
func isPrivateGitLab(repoURL string, httpClient *http.Client) bool {
	groupPath, projectName, _ := extractGitLabGroupProject(repoURL)
	if groupPath == "" || projectName == "" {
		return true
	}

	// Hardcode gitlab.com to prevent SSRF attacks via untrusted remote.origin.url
	directURL := fmt.Sprintf("https://gitlab.com/%s/%s", groupPath, projectName)
	isPublic := isRepoPublic(directURL, httpClient)
	return !isPublic
}

// isPrivateBitbucket checks Bitbucket repo privacy via direct HTTP URL (HTTP 200 = Public, else = Private).
func isPrivateBitbucket(repoURL string, httpClient *http.Client) bool {
	workspace, repo := extractBitbucketWorkspaceRepo(repoURL)
	if workspace == "" || repo == "" {
		return true
	}

	directURL := fmt.Sprintf("https://bitbucket.org/%s/%s", workspace, repo)
	isPublic := isRepoPublic(directURL, httpClient)
	return !isPublic
}

// isPrivateAzureDevOps checks Azure DevOps repo privacy via public API (no auth required).
func isPrivateAzureDevOps(repoURL string, httpClient *http.Client) bool {
	org, repo := extractAzureDevOpsOrgRepo(repoURL)
	if org == "" || repo == "" {
		return true
	}

	directURL := fmt.Sprintf("https://dev.azure.com/%s/_git/%s", org, repo)
	isPublic := isRepoPublic(directURL, httpClient)
	return !isPublic
}

// isRepoPublic checks if repository is publicly accessible via injected HTTP client (HTTP 200 = public, else = private).
func isRepoPublic(repoURL string, httpClient *http.Client) bool {
	resp, err := httpClient.Get(repoURL)
	if err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("Repository accessibility check failed for %s: %v, treating as PRIVATE", repoURL, err))
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	isPublic := resp.StatusCode == http.StatusOK
	if !isPublic {
		logger.PrintIfVerbose(fmt.Sprintf("Repository URL %s returned status %d, treating as PRIVATE", repoURL, resp.StatusCode))
	}
	return isPublic
}

// normalizeSSHURL converts SSH URLs to HTTPS (git@github.com:owner/repo.git → https://github.com/owner/repo.git)
func normalizeSSHURL(repoURL string) string {
	if !strings.Contains(repoURL, "://") && strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
		// Handle git@host:path format
		// git@github.com:owner/repo.git → https://github.com/owner/repo.git
		parts := strings.SplitN(repoURL, "@", sshSplitParts)
		if len(parts) == sshSplitParts {
			hostAndPath := strings.SplitN(parts[1], ":", sshSplitParts)
			if len(hostAndPath) == sshSplitParts {
				return "https://" + hostAndPath[0] + "/" + hostAndPath[1]
			}
		}
	} else if strings.HasPrefix(repoURL, "ssh://") {
		// Handle ssh://git@host/path format → https://host/path
		sshURL := strings.TrimPrefix(repoURL, "ssh://")
		if strings.Contains(sshURL, "@") {
			parts := strings.SplitN(sshURL, "@", sshSplitParts)
			if len(parts) == sshSplitParts {
				return "https://" + parts[1]
			}
		}
	}
	return repoURL
}

// Extract owner/repo from GitHub URLs: https://github.com/owner/repo or owner/repo
func extractGitHubOwnerRepo(repoURL string) (owner, repo string) {
	repoURL = normalizeSSHURL(repoURL)
	url := strings.TrimSuffix(repoURL, ".git")
	parts := strings.FieldsFunc(url, func(r rune) bool { return r == '/' })
	if len(parts) >= pathParts {
		return parts[len(parts)-pathParts], parts[len(parts)-1]
	}
	return "", ""
}

// extractGitLabGroupProject extracts group/project from GitLab URLs, supporting nested subgroups (last segment is project, rest is group).
func extractGitLabGroupProject(repoURL string) (group, project, host string) {
	repoURL = normalizeSSHURL(repoURL)
	url := strings.TrimSuffix(repoURL, ".git")

	// Extract host if present
	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", urlSchemeParts)
		hostAndPath := strings.SplitN(parts[1], "/", pathParts)
		if len(hostAndPath) == pathParts {
			host = hostAndPath[0]
			url = hostAndPath[1]
		}
	}

	parts := strings.FieldsFunc(url, func(r rune) bool { return r == '/' })
	if len(parts) >= pathParts {
		// Take last two segments (project is always the last one; group/subgroup is everything before)
		project = parts[len(parts)-1]
		group = strings.Join(parts[:len(parts)-1], "/")
		return
	}
	return "", "", host
}

// Extract workspace/repo from Bitbucket URLs: https://bitbucket.org/workspace/repo
func extractBitbucketWorkspaceRepo(repoURL string) (workspace, repo string) {
	repoURL = normalizeSSHURL(repoURL)
	url := strings.TrimSuffix(repoURL, ".git")
	parts := strings.FieldsFunc(url, func(r rune) bool { return r == '/' })
	if len(parts) >= pathParts {
		return parts[len(parts)-pathParts], parts[len(parts)-1]
	}
	return "", ""
}

// Extract org/repo from Azure DevOps URLs: https://dev.azure.com/org/_git/repo or https://ssh.dev.azure.com/v3/org/project/repo
func extractAzureDevOpsOrgRepo(repoURL string) (org, repo string) {
	repoURL = normalizeSSHURL(repoURL)
	url := strings.TrimSuffix(repoURL, ".git")
	if strings.Contains(url, "dev.azure.com") {
		parts := strings.Split(url, "/")
		for i, part := range parts {
			if part == "dev.azure.com" && i+1 < len(parts) {
				org := parts[i+1]
				// Find _git segment (HTTPS format)
				for j := i + 2; j < len(parts); j++ {
					if parts[j] == "_git" && j+1 < len(parts) {
						repo := parts[j+1]
						return org, repo
					}
				}
			} else if part == "ssh.dev.azure.com" && i+1 < len(parts) && parts[i+1] == "v3" {
				// SSH format: ssh.dev.azure.com/v3/org/project/repo
				if i+3 < len(parts) {
					org := parts[i+2]
					if i+4 < len(parts) {
						repo := parts[i+4]
						return org, repo
					}
				}
			}
		}
	}
	return "", ""
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
