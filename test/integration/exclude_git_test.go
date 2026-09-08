//go:build integration

package integration

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// Integration tests for --exclude-git-folder flag
// Clones a public repo once at package level and reuses for all tests

var (
	clonedRepoPath string
	cloneOnce      sync.Once
	cloneErr       error
)

// clonePublicRepoOnce clones the repo exactly once and caches the path
func clonePublicRepoOnce() string {
	cloneOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "ast-cli-git-test-*")
		if err != nil {
			cloneErr = err
			return
		}

		// Clone public repo: ast-vscode-extension
		repoURL := "https://github.com/Checkmarx/ast-vscode-extension.git"
		cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
		if err := cmd.Run(); err != nil {
			cloneErr = err
			return
		}

		// Verify .git folder exists in cloned repo
		gitPath := filepath.Join(tempDir, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			cloneErr = err
			return
		}

		clonedRepoPath = tempDir
	})
	return clonedRepoPath
}

func TestExcludeGitFolder_WithFlag(t *testing.T) {
	repoPath := clonePublicRepoOnce()
	assert.NilError(t, cloneErr, "Failed to clone public repository")

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), repoPath,
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "main",
		flag(params.ExcludeGitFolderFlag),
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "scan create with --exclude-git-folder should succeed")

	logText := buf.String()
	assert.Assert(t, strings.Contains(logText, "The folder .git is being excluded"), "Expected .git exclusion message not found in logs")
	assert.Assert(t, strings.Contains(logText, "--exclude-git-folder flag passed"), "Expected --exclude-git-folder flag confirmation message not found in logs")
}

func TestExcludeGitFolder_IncludeCsvJson(t *testing.T) {
	repoPath := clonePublicRepoOnce()
	assert.NilError(t, cloneErr, "Failed to clone public repository")

	args := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), repoPath,
		flag(params.ScanTypes), params.IacType,
		flag(params.BranchFlag), "main",
		flag(params.ExcludeGitFolderFlag),
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "scan create with --exclude-git-folder should succeed")

	logText := buf.String()
	assert.Assert(t, strings.Contains(logText, "The folder .git is being excluded"), "Expected .git exclusion message not found in logs")
	assert.Assert(t, strings.Contains(logText, "--exclude-git-folder flag passed"), "Expected --exclude-git-folder flag confirmation message not found in logs")
	assert.Assert(t, strings.Contains(logText, "Included: .checkmarx/metadata.json"), "Included COntributor.csv and metadata.json")
}
