package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCommit describes one fixture commit. AllowEmptyCommits means fixtures
// don't need real file content.
type testCommit struct {
	email string
	name  string
	when  time.Time
}

func newGitMetadataTestRepo(t *testing.T, remoteURL string, commits []testCommit) string {
	t.Helper()
	repoPath := t.TempDir()

	repo, err := gogit.PlainInit(repoPath, false)
	require.NoError(t, err)

	if remoteURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteURL},
		})
		require.NoError(t, err)
	}

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	// Commits must be created oldest-first so the last one ends up as HEAD.
	for _, c := range commits {
		sig := &object.Signature{Name: c.name, Email: c.email, When: c.when}
		_, err := worktree.Commit("test commit", &gogit.CommitOptions{
			Author:            sig,
			AllowEmptyCommits: true,
		})
		require.NoError(t, err)
	}

	return repoPath
}

func readGitMetadataFiles(t *testing.T, repoPath string) (csvContent string, metadata contributorsMetadata) {
	t.Helper()
	csvBytes, err := os.ReadFile(filepath.Join(repoPath, CheckmarxFolderName, ContributorsFileName))
	require.NoError(t, err)

	metadataBytes, err := os.ReadFile(filepath.Join(repoPath, CheckmarxFolderName, MetadataFileName))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))

	return string(csvBytes), metadata
}

func TestGenerateAndWrite_Success(t *testing.T) {
	now := time.Now()
	repoPath := newGitMetadataTestRepo(t, "https://example.com/org/repo.git", []testCommit{
		{email: "alice@example.com", name: "Alice", when: now.Add(-10 * 24 * time.Hour)},
		{email: "bob@example.com", name: "Bob", when: now.Add(-5 * 24 * time.Hour)},
		{email: "alice@example.com", name: "Alice", when: now.Add(-1 * time.Hour)},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)

	lines := strings.Split(strings.TrimRight(csvContent, "\n"), "\n")
	assert.Len(t, lines, 2, "expected one row per unique email, most recent commit only")

	assert.Equal(t, "https://example.com/org/repo.git", metadata.RepositoryURL)
	assert.Equal(t, 3, metadata.CommitsCount, "commitsCount should count every commit in the window, before dedup")
	assert.NotEmpty(t, metadata.LastCommitHash)
	assert.NotEmpty(t, metadata.LastCommitDate)
}

func TestGenerateAndWrite_MetadataJSONFieldNames(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "https://example.com/org/repo.git", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})
	require.NoError(t, GenerateAndWrite(repoPath, true))

	raw, err := os.ReadFile(filepath.Join(repoPath, CheckmarxFolderName, MetadataFileName))
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &asMap))

	assert.Contains(t, asMap, "repositoryUrl")
	assert.Contains(t, asMap, "lastCommitHash")
	assert.Contains(t, asMap, "lastCommitDate")
	assert.Contains(t, asMap, "commitsCount")
	assert.NotContains(t, asMap, "branchName", "branchName is intentionally omitted, see tech design open questions")
	assert.NotContains(t, asMap, "commitHash", "field is named lastCommitHash, not commitHash")
}

func TestGenerateAndWrite_NoHeaderRow(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, _ := readGitMetadataFiles(t, repoPath)
	firstLine := strings.SplitN(csvContent, ",", 2)[0]

	// A header would read "date" or similar; a real row starts with an RFC3339
	// timestamp, which always begins with a 4-digit year.
	_, err := time.Parse(time.RFC3339, firstLine)
	assert.NoError(t, err, "first line should be a data row (RFC3339 date), not a header")
}

func TestGenerateAndWrite_CSVColumnOrder(t *testing.T) {
	now := time.Now()
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "alice@example.com", name: "Alice Example", when: now},
	})
	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)
	fields := strings.Split(strings.TrimRight(csvContent, "\n"), ",")
	require.Len(t, fields, 4)

	assert.Equal(t, now.Format(time.RFC3339), fields[0], "field 1 must be the commit date")
	assert.Equal(t, metadata.LastCommitHash, fields[1], "field 2 must be the commit hash")
	assert.Equal(t, "alice@example.com", fields[2], "field 3 must be the email")
	assert.Equal(t, "Alice Example", fields[3], "field 4 must be the username")
}

func TestGenerateAndWrite_DedupKeepsMostRecentPerEmail(t *testing.T) {
	now := time.Now()
	olderTime := now.Add(-20 * 24 * time.Hour)
	newerTime := now.Add(-1 * 24 * time.Hour)

	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "alice@example.com", name: "Alice Old", when: olderTime},
		{email: "alice@example.com", name: "Alice New", when: newerTime},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, _ := readGitMetadataFiles(t, repoPath)
	lines := strings.Split(strings.TrimRight(csvContent, "\n"), "\n")
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], newerTime.Format(time.RFC3339))
	assert.Contains(t, lines[0], "Alice New")
	assert.NotContains(t, lines[0], "Alice Old")
}

func TestGenerateAndWrite_DedupIsCaseInsensitiveOnEmail(t *testing.T) {
	now := time.Now()
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "Alice@Example.com", name: "Alice Mixed Case", when: now.Add(-2 * 24 * time.Hour)},
		{email: "alice@example.com", name: "Alice Lower Case", when: now.Add(-1 * time.Hour)},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)
	lines := strings.Split(strings.TrimRight(csvContent, "\n"), "\n")
	assert.Len(t, lines, 1, "differently-cased emails for the same person should dedup to one row")
	assert.Equal(t, 2, metadata.CommitsCount, "commitsCount still counts both raw commits")
}

func TestGenerateAndWrite_ManyUniqueEmailsAllKept(t *testing.T) {
	now := time.Now()
	var commits []testCommit
	for i := 0; i < 5; i++ {
		commits = append(commits, testCommit{
			email: strings.Repeat("u", 1) + string(rune('a'+i)) + "@example.com",
			name:  "User",
			when:  now.Add(-time.Duration(i) * time.Hour),
		})
	}
	repoPath := newGitMetadataTestRepo(t, "", commits)
	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)
	lines := strings.Split(strings.TrimRight(csvContent, "\n"), "\n")
	assert.Len(t, lines, 5, "each unique email should get its own row")
	assert.Equal(t, 5, metadata.CommitsCount)
}

func TestGenerateAndWrite_90DayBoundary(t *testing.T) {
	now := time.Now()
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "old@example.com", name: "TooOld", when: now.Add(-91 * 24 * time.Hour)},
		{email: "recent@example.com", name: "Recent", when: now.Add(-89 * 24 * time.Hour)},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)
	assert.Contains(t, csvContent, "recent@example.com")
	assert.NotContains(t, csvContent, "old@example.com")
	assert.Equal(t, 1, metadata.CommitsCount)
}

func TestGenerateAndWrite_AllCommitsOutsideWindow(t *testing.T) {
	now := time.Now()
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "old@example.com", name: "TooOld", when: now.Add(-200 * 24 * time.Hour)},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	csvContent, metadata := readGitMetadataFiles(t, repoPath)
	assert.Empty(t, csvContent, "no commits in the window means an empty (but present) CSV")
	assert.Equal(t, 0, metadata.CommitsCount)
	assert.NotEmpty(t, metadata.LastCommitHash, "HEAD info is unconditional, independent of the 90-day window")
}

func TestGenerateAndWrite_NoRemote(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	_, metadata := readGitMetadataFiles(t, repoPath)
	assert.Empty(t, metadata.RepositoryURL)
}

func TestGenerateAndWrite_SSHRemoteURL(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "git@github.com:org/repo.git", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))

	_, metadata := readGitMetadataFiles(t, repoPath)
	assert.Equal(t, "git@github.com:org/repo.git", metadata.RepositoryURL)
}

func TestGenerateAndWrite_ReplacesStaleFiles(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))
	_, firstMetadata := readGitMetadataFiles(t, repoPath)

	repo, err := gogit.PlainOpen(repoPath)
	require.NoError(t, err)
	worktree, err := repo.Worktree()
	require.NoError(t, err)
	_, err = worktree.Commit("second commit", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Bob", Email: "bob@example.com", When: time.Now()},
		AllowEmptyCommits: true,
	})
	require.NoError(t, err)

	require.NoError(t, GenerateAndWrite(repoPath, true))
	csvContent, secondMetadata := readGitMetadataFiles(t, repoPath)

	assert.NotEqual(t, firstMetadata.LastCommitHash, secondMetadata.LastCommitHash, "second run should overwrite metadata.json with fresh data")
	assert.Contains(t, csvContent, "bob@example.com")
	assert.Contains(t, csvContent, "alice@example.com")
}

func TestGenerateAndWrite_RunTwiceIdenticalStateProducesIdenticalOutput(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "https://example.com/repo.git", []testCommit{
		{email: "alice@example.com", name: "Alice", when: time.Now()},
	})

	require.NoError(t, GenerateAndWrite(repoPath, true))
	firstCSV, firstMetadata := readGitMetadataFiles(t, repoPath)

	require.NoError(t, GenerateAndWrite(repoPath, true))
	secondCSV, secondMetadata := readGitMetadataFiles(t, repoPath)

	assert.Equal(t, firstCSV, secondCSV)
	assert.Equal(t, firstMetadata, secondMetadata)
}

func TestGenerateAndWrite_NotAGitRepository(t *testing.T) {
	dir := t.TempDir()
	err := GenerateAndWrite(dir, true)
	assert.Error(t, err)
}

func TestGenerateAndWrite_NoCommits(t *testing.T) {
	dir := t.TempDir()
	_, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)

	err = GenerateAndWrite(dir, true)
	assert.Error(t, err, "an empty repo has no HEAD to resolve")
}

func TestGenerateAndWrite_NonExistentPath(t *testing.T) {
	err := GenerateAndWrite(filepath.Join(t.TempDir(), "does-not-exist"), true)
	assert.Error(t, err)
}

// TestBuildContributorsCSV_ExceedsSizeLimitStillSucceeds exercises buildContributorsCSV
// directly with synthetic in-memory commits (no real git repo), since creating enough
// real commits to exceed the 1MB warning threshold would make the test very slow.
func TestBuildContributorsCSV_ExceedsSizeLimitStillSucceeds(t *testing.T) {
	now := time.Now()
	commits := make([]*object.Commit, 0, 20000)
	for i := 0; i < 20000; i++ {
		commits = append(commits, &object.Commit{
			Author: object.Signature{
				Name:  "User",
				Email: "user" + strconv.Itoa(i) + "@example.com",
				When:  now.Add(-time.Duration(i) * time.Second),
			},
		})
	}

	csvData, err := buildContributorsCSV(commits)
	require.NoError(t, err, "exceeding the size warning threshold must not fail generation")
	assert.Greater(t, len(csvData), contributorsSizeLimit)

	lines := strings.Split(strings.TrimRight(string(csvData), "\n"), "\n")
	assert.Len(t, lines, len(commits), "all unique emails should be kept as rows")
}

func TestBuildContributorsCSV_EmptyInput(t *testing.T) {
	csvData, err := buildContributorsCSV(nil)
	require.NoError(t, err)
	assert.Empty(t, csvData)
}

func TestDedupByEmail_EmptyInput(t *testing.T) {
	assert.Empty(t, dedupByEmail(nil))
}

func TestDedupByEmail_PreservesFirstOccurrenceOrder(t *testing.T) {
	now := time.Now()
	commits := []*object.Commit{
		{Author: object.Signature{Email: "a@example.com", When: now}},
		{Author: object.Signature{Email: "b@example.com", When: now}},
		{Author: object.Signature{Email: "a@example.com", When: now}}, // duplicate, later in slice
		{Author: object.Signature{Email: "c@example.com", When: now}},
	}

	deduped := dedupByEmail(commits)
	require.Len(t, deduped, 3)
	assert.Equal(t, "a@example.com", deduped[0].Author.Email)
	assert.Equal(t, "b@example.com", deduped[1].Author.Email)
	assert.Equal(t, "c@example.com", deduped[2].Author.Email)
}

func TestBuildMetadataJSON_Structure(t *testing.T) {
	headCommit := &object.Commit{
		Author: object.Signature{When: time.Date(2025, 9, 30, 10, 35, 5, 0, time.UTC)},
	}

	data, err := buildMetadataJSON("https://example.com/repo.git", headCommit, 42)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &asMap))
	assert.Equal(t, "https://example.com/repo.git", asMap["repositoryUrl"])
	assert.Equal(t, headCommit.Hash.String(), asMap["lastCommitHash"])
	assert.Equal(t, "2025-09-30T10:35:05Z", asMap["lastCommitDate"])
	assert.InDelta(t, 42, asMap["commitsCount"], 0)
}

func TestCommitsSince_SortsNewestFirst(t *testing.T) {
	now := time.Now()
	// Committed out of chronological order to verify commitsSince re-sorts them.
	repoPath := newGitMetadataTestRepo(t, "", []testCommit{
		{email: "a@example.com", name: "A", when: now.Add(-5 * 24 * time.Hour)},
		{email: "b@example.com", name: "B", when: now.Add(-1 * 24 * time.Hour)},
		{email: "c@example.com", name: "C", when: now.Add(-10 * 24 * time.Hour)},
	})

	repo, err := gogit.PlainOpen(repoPath)
	require.NoError(t, err)

	commits, err := commitsSince(repo, now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, commits, 3)

	assert.True(t, commits[0].Author.When.After(commits[1].Author.When))
	assert.True(t, commits[1].Author.When.After(commits[2].Author.When))
}

func TestRemoteURL_ReturnsFirstURL(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "https://example.com/first.git", nil)
	repo, err := gogit.PlainOpen(repoPath)
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/first.git", remoteURL(repo))
}

func TestRemoteURL_NoRemoteConfigured(t *testing.T) {
	repoPath := newGitMetadataTestRepo(t, "", nil)
	repo, err := gogit.PlainOpen(repoPath)
	require.NoError(t, err)

	assert.Empty(t, remoteURL(repo))
}

func TestParseGitLogOutput_BasicParsing(t *testing.T) {
	logOutput := "2026-08-19T18:24:26+05:30\x1fabc123def456789abc123def456789abc12345\x1fuser@example.com\x1fJohn Doe\n" +
		"2026-08-13T12:58:25+03:00\x1fdef456789abc123def456789abc123def45678\x1fjane@example.com\x1fJane Smith"

	commits := parseGitLogOutput(logOutput)
	require.Len(t, commits, 2)

	assert.Equal(t, "2026-08-13T12:58:25+03:00", commits[0]["date"])
	assert.Equal(t, "def456789abc123def456789abc123def45678", commits[0]["hash"])
	assert.Equal(t, "jane@example.com", commits[0]["email"])
	assert.Equal(t, "Jane Smith", commits[0]["name"])

	assert.Equal(t, "2026-08-19T18:24:26+05:30", commits[1]["date"])
	assert.Equal(t, "abc123def456789abc123def456789abc12345", commits[1]["hash"])
	assert.Equal(t, "user@example.com", commits[1]["email"])
	assert.Equal(t, "John Doe", commits[1]["name"])
}

func TestParseGitLogOutput_RevertsToNewestFirst(t *testing.T) {
	logOutput := "2026-07-01T10:00:00+00:00\x1f1111111111111111111111111111111111111111\x1fold@example.com\x1fOld User\n" +
		"2026-07-15T15:00:00+00:00\x1f2222222222222222222222222222222222222222\x1fmid@example.com\x1fMid User\n" +
		"2026-08-19T20:00:00+00:00\x1f3333333333333333333333333333333333333333\x1fnew@example.com\x1fNew User"

	commits := parseGitLogOutput(logOutput)
	require.Len(t, commits, 3)

	assert.Equal(t, "2026-08-19T20:00:00+00:00", commits[0]["date"])
	assert.Equal(t, "2026-07-15T15:00:00+00:00", commits[1]["date"])
	assert.Equal(t, "2026-07-01T10:00:00+00:00", commits[2]["date"])
}

func TestParseGitLogOutput_EmptyInput(t *testing.T) {
	commits := parseGitLogOutput("")
	assert.Nil(t, commits)
}

func TestParseGitLogOutput_SkipsMalformedLines(t *testing.T) {
	logOutput := "2026-08-19T18:24:26+05:30\x1fabc123def456789abc123def456789abc12345\x1fuser@example.com\x1fJohn Doe\n" +
		"malformed line\n" +
		"2026-08-13T12:58:25+03:00\x1fdef456789abc123def456789abc123def45678\x1fjane@example.com\x1fJane Smith\n" +
		"\n" +
		"another bad line"

	commits := parseGitLogOutput(logOutput)
	require.Len(t, commits, 2)
	assert.Equal(t, "John Doe", commits[1]["name"])
	assert.Equal(t, "Jane Smith", commits[0]["name"])
}

func TestConvertToCoreCommits_BasicConversion(t *testing.T) {
	now := time.Now()
	commitMaps := []map[string]string{
		{
			"date":  now.Format(time.RFC3339),
			"hash":  "abc123def456789abcdef456789abcdef1234567",
			"email": "user@example.com",
			"name":  "Test User",
		},
	}

	commits := convertToCoreCommits(commitMaps)
	require.Len(t, commits, 1)

	c := commits[0]
	assert.Equal(t, "abc123def456789abcdef456789abcdef1234567", c.Hash.String())
	assert.Equal(t, "user@example.com", c.Author.Email)
	assert.Equal(t, "Test User", c.Author.Name)
	assert.WithinDuration(t, now, c.Author.When, 1*time.Second)
}

func TestConvertToCoreCommits_PreservesOrder(t *testing.T) {
	commitMaps := []map[string]string{
		{
			"date":  "2026-08-19T18:24:26+05:30",
			"hash":  "1111111111111111111111111111111111111111",
			"email": "first@example.com",
			"name":  "First",
		},
		{
			"date":  "2026-08-13T12:58:25+03:00",
			"hash":  "2222222222222222222222222222222222222222",
			"email": "second@example.com",
			"name":  "Second",
		},
	}

	commits := convertToCoreCommits(commitMaps)
	require.Len(t, commits, 2)

	assert.Equal(t, "1111111111111111111111111111111111111111", commits[0].Hash.String())
	assert.Equal(t, "2222222222222222222222222222222222222222", commits[1].Hash.String())
}

func TestConvertToCoreCommits_RFC3339DateParsing(t *testing.T) {
	commitMaps := []map[string]string{
		{
			"date":  "2026-08-19T18:24:26+05:30",
			"hash":  "abc123",
			"email": "user@example.com",
			"name":  "User",
		},
	}

	commits := convertToCoreCommits(commitMaps)
	require.Len(t, commits, 1)

	// Verify the date was parsed correctly (RFC3339 with timezone)
	assert.Equal(t, 2026, commits[0].Author.When.Year())
	assert.Equal(t, time.August, commits[0].Author.When.Month())
	assert.Equal(t, 19, commits[0].Author.When.Day())
}

func TestBuildMetadataJSONFromSystem_Structure(t *testing.T) {
	data, err := buildMetadataJSONFromSystem("https://github.com/example/repo.git", "abc123def456", 42)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &asMap))

	assert.Equal(t, "https://github.com/example/repo.git", asMap["repositoryUrl"])
	assert.Equal(t, "abc123def456", asMap["lastCommitHash"])
	assert.InDelta(t, 42, asMap["commitsCount"], 0)
	assert.NotEmpty(t, asMap["lastCommitDate"])

	// Verify lastCommitDate is valid RFC3339
	_, err = time.Parse(time.RFC3339, asMap["lastCommitDate"].(string))
	require.NoError(t, err)
}

func TestBuildMetadataJSONFromSystem_EmptyRepository(t *testing.T) {
	data, err := buildMetadataJSONFromSystem("", "", 0)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &asMap))

	assert.Equal(t, "", asMap["repositoryUrl"])
	assert.Equal(t, "", asMap["lastCommitHash"])
	assert.InDelta(t, 0, asMap["commitsCount"], 0)
}

func TestGitCommand_Success(t *testing.T) {
	// Create a real git repo for this test
	repoPath := t.TempDir()

	// Initialize a git repository using system git
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	// Configure git user for this test repo (required in CI)
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	// Test git config command
	output, err := gitCommand(repoPath, "config", "--get", "user.name")
	assert.NoError(t, err)
	assert.Equal(t, "Test User", output)
}

func TestGitCommand_InvalidRepo(t *testing.T) {
	invalidPath := t.TempDir()
	// This directory is not a git repo

	_, err := gitCommand(invalidPath, "log", "--oneline")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git command failed")
}

// Tests for URL extraction functions

func TestExtractGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "HTTPS GitHub URL",
			url:       "https://github.com/checkmarx/ast-cli",
			wantOwner: "checkmarx",
			wantRepo:  "ast-cli",
		},
		{
			name:      "HTTPS GitHub URL with .git suffix",
			url:       "https://github.com/checkmarx/ast-cli.git",
			wantOwner: "checkmarx",
			wantRepo:  "ast-cli",
		},
		{
			name:      "Short format GitHub URL",
			url:       "checkmarx/ast-cli",
			wantOwner: "checkmarx",
			wantRepo:  "ast-cli",
		},
		{
			name:      "Invalid format - too few parts",
			url:       "invalid",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "Empty URL",
			url:       "",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := extractGitHubOwnerRepo(tt.url)
			assert.Equal(t, tt.wantOwner, owner, "owner mismatch")
			assert.Equal(t, tt.wantRepo, repo, "repo mismatch")
		})
	}
}

func TestExtractGitLabGroupProject(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantGroup string
		wantRepo  string
		wantHost  string
	}{
		{
			name:      "HTTPS GitLab URL",
			url:       "https://gitlab.com/checkmarx/ast-cli",
			wantGroup: "checkmarx",
			wantRepo:  "ast-cli",
			wantHost:  "gitlab.com",
		},
		{
			name:      "HTTPS GitLab URL with .git suffix",
			url:       "https://gitlab.com/checkmarx/ast-cli.git",
			wantGroup: "checkmarx",
			wantRepo:  "ast-cli",
			wantHost:  "gitlab.com",
		},
		{
			name:      "Self-hosted GitLab",
			url:       "https://gitlab.internal.com/checkmarx/ast-cli",
			wantGroup: "checkmarx",
			wantRepo:  "ast-cli",
			wantHost:  "gitlab.internal.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, project, host := extractGitLabGroupProject(tt.url)
			assert.Equal(t, tt.wantGroup, group, "group mismatch")
			assert.Equal(t, tt.wantRepo, project, "project mismatch")
			assert.Equal(t, tt.wantHost, host, "host mismatch")
		})
	}
}

func TestExtractBitbucketWorkspaceRepo(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantWorkspace string
		wantRepo      string
	}{
		{
			name:          "HTTPS Bitbucket URL",
			url:           "https://bitbucket.org/checkmarx/ast-cli",
			wantWorkspace: "checkmarx",
			wantRepo:      "ast-cli",
		},
		{
			name:          "HTTPS Bitbucket URL with .git suffix",
			url:           "https://bitbucket.org/checkmarx/ast-cli.git",
			wantWorkspace: "checkmarx",
			wantRepo:      "ast-cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo := extractBitbucketWorkspaceRepo(tt.url)
			assert.Equal(t, tt.wantWorkspace, workspace, "workspace mismatch")
			assert.Equal(t, tt.wantRepo, repo, "repo mismatch")
		})
	}
}

func TestExtractAzureDevOpsOrgRepo(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantOrg  string
		wantRepo string
	}{
		{
			name:     "HTTPS Azure DevOps URL with _git",
			url:      "https://dev.azure.com/checkmarx/project/_git/ast-cli",
			wantOrg:  "checkmarx",
			wantRepo: "ast-cli",
		},
		{
			name:     "HTTPS Azure DevOps URL with .git suffix",
			url:      "https://dev.azure.com/checkmarx/project/_git/ast-cli.git",
			wantOrg:  "checkmarx",
			wantRepo: "ast-cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo := extractAzureDevOpsOrgRepo(tt.url)
			assert.Equal(t, tt.wantOrg, org, "org mismatch")
			assert.Equal(t, tt.wantRepo, repo, "repo mismatch")
		})
	}
}

// Tests for privacy detection with mocked HTTP responses

func TestFileExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantTrue bool
	}{
		{
			name: "file exists",
			setup: func(t *testing.T) string {
				f, err := os.CreateTemp(t.TempDir(), "test")
				require.NoError(t, err)
				path := f.Name()
				require.NoError(t, f.Close())
				return path
			},
			wantTrue: true,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			wantTrue: false,
		},
		{
			name: "directory exists",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantTrue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			exists := fileExists(path)
			assert.Equal(t, tt.wantTrue, exists)
		})
	}
}

func TestPrivacyDetectionWithMockedHTTP(t *testing.T) {
	t.Run("detectRepositoryPrivacy returns private for empty path", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Empty directory with no .git or .checkmarx files
		isPrivate := detectRepositoryPrivacy(tmpDir)
		assert.True(t, isPrivate, "empty directory should default to private")
	})

	t.Run("detectRepositoryPrivacy returns private when checking nonexistent path", func(t *testing.T) {
		isPrivate := detectRepositoryPrivacy("/nonexistent/path/that/does/not/exist")
		assert.True(t, isPrivate, "nonexistent path should default to private")
	})

	t.Run("isPrivateByURL returns private for empty URL", func(t *testing.T) {
		isPrivate := isPrivateByURL("")
		assert.True(t, isPrivate, "empty URL should be private")
	})

	t.Run("isPrivateByURL routes unknown platforms to private", func(t *testing.T) {
		isPrivate := isPrivateByURL("https://unknown-git-hosting.com/team/repo")
		assert.True(t, isPrivate, "unknown platform should default to private")
	})
}

func TestIsRepoPublicWithMockedServer(t *testing.T) {
	t.Run("public repository returns true when HTTP 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Test with mocked server URL
		isPublic := isRepoPublic(server.URL)
		assert.True(t, isPublic, "HTTP 200 should indicate public")
	})

	t.Run("private repository returns false when HTTP 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		isPublic := isRepoPublic(server.URL)
		assert.False(t, isPublic, "HTTP 404 should indicate not public (private)")
	})

	t.Run("empty URL returns false (not public/private)", func(t *testing.T) {
		isPublic := isRepoPublic("")
		assert.False(t, isPublic, "empty URL should not be public")
	})

	t.Run("malformed URL returns false (not public/private)", func(t *testing.T) {
		isPublic := isRepoPublic("://invalid")
		assert.False(t, isPublic, "malformed URL should not be public")
	})
}
