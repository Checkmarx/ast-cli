//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreCommitIntegration(t *testing.T) {
	t.Run("Install and Uninstall Local Hook", func(t *testing.T) {
		tmpDir, cleanup := setupTempDir(t)
		defer cleanup()

		// Initialize Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if out, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %s", err, string(out))
		}

		// Install hook locally
		output := executeCmdNilAssertion(t, "pre-commit install should not fail",
			"hooks", "pre-commit", "secrets-install-git-hook")
		assert.Contains(t, output.String(), "pre-commit installed successfully")

		// Verify hook installation
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
		_, err := os.Stat(hookPath)
		assert.NoError(t, err, "pre-commit hook should exist")

		// Uninstall hook
		output = executeCmdNilAssertion(t, "pre-commit uninstall should not fail",
			"hooks", "pre-commit", "secrets-uninstall-git-hook")
		assert.Contains(t, output.String(), "pre-commit hook uninstalled successfully")

		// Verify hook removal
		_, err = os.Stat(hookPath)
		assert.True(t, os.IsNotExist(err), "pre-commit hook should be removed")
	})

	t.Run("Install and Uninstall Global Hook", func(t *testing.T) {
		tmpDir, cleanup := setupTempDir(t)
		defer cleanup()

		// Initialize Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if out, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %s", err, string(out))
		}

		// Install hook globally
		output := executeCmdNilAssertion(t, "pre-commit global install should not fail",
			"hooks", "pre-commit", "secrets-install-git-hook", "--global")
		assert.Contains(t, output.String(), "pre-commit installed globally successfully")

		// Verify global hook installation
		homeDir, err := os.UserHomeDir()
		assert.NoError(t, err)
		globalHookPath := filepath.Join(homeDir, ".git", "hooks", "pre-commit")
		_, err = os.Stat(globalHookPath)
		assert.NoError(t, err, "global pre-commit hook should exist")

		// Uninstall global hook
		output = executeCmdNilAssertion(t, "pre-commit global uninstall should not fail",
			"hooks", "pre-commit", "secrets-uninstall-git-hook", "--global")
		assert.Contains(t, output.String(), "pre-commit hook uninstalled globally successfully")

		// Verify global hook removal
		_, err = os.Stat(globalHookPath)
		assert.True(t, os.IsNotExist(err), "global pre-commit hook should be removed")
	})

	t.Run("Scan for Secrets", func(t *testing.T) {
		tmpDir, cleanup := setupTempDir(t)
		defer cleanup()

		// Initialize Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if out, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %s", err, string(out))
		}

		// Create a file with a secret
		secretContent := `MOCK CONTENT
ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
MOCK CONTENT`
		filePath := filepath.Join(tmpDir, "secret.txt")
		err := os.WriteFile(filePath, []byte(secretContent), 0644)
		assert.NoError(t, err)

		// Stage the file
		cmdGitAdd := exec.Command("git", "add", "secret.txt")
		cmdGitAdd.Dir = tmpDir
		if out, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("git add failed: %s: %s", err, string(out))
		}

		// Run scan - should detect secret
		err, output := executeCommand(t, "hooks", "pre-commit", "secrets-scan")
		assert.Error(t, err)
		assert.Contains(t, output.String(), "Secret detected")

		// Ignore the secret
		output = executeCmdNilAssertion(t, "pre-commit ignore should not fail",
			"hooks", "pre-commit", "secrets-ignore", "--all")
		assert.Contains(t, output.String(), "Added new IDs to .checkmarx_ignore")

		// Run scan again - should pass
		output = executeCmdNilAssertion(t, "pre-commit scan should pass after ignoring",
			"hooks", "pre-commit", "secrets-scan")
		assert.Contains(t, output.String(), "No secrets detected")
	})

	t.Run("Update Hook", func(t *testing.T) {
		tmpDir, cleanup := setupTempDir(t)
		defer cleanup()

		// Initialize Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if out, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %s", err, string(out))
		}

		// Install hook
		output := executeCmdNilAssertion(t, "pre-commit install should not fail",
			"hooks", "pre-commit", "secrets-install-git-hook")
		assert.Contains(t, output.String(), "pre-commit installed successfully")

		// Update hook
		output = executeCmdNilAssertion(t, "pre-commit update should not fail",
			"hooks", "pre-commit", "secrets-update-git-hook")
		assert.Contains(t, output.String(), "pre-commit hook updated successfully")
	})

	t.Run("Ignore Specific Secrets", func(t *testing.T) {
		tmpDir, cleanup := setupTempDir(t)
		defer cleanup()

		// Initialize Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if out, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %s", err, string(out))
		}

		// Create a file with multiple secrets
		secretContent := `MOCK CONTENT
ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
ghp_BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
MOCK CONTENT`
		filePath := filepath.Join(tmpDir, "secrets.txt")
		err := os.WriteFile(filePath, []byte(secretContent), 0644)
		assert.NoError(t, err)

		// Stage the file
		cmdGitAdd := exec.Command("git", "add", "secrets.txt")
		cmdGitAdd.Dir = tmpDir
		if out, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("git add failed: %s: %s", err, string(out))
		}

		// Run scan - should detect secrets
		err, output := executeCommand(t, "hooks", "pre-commit", "secrets-scan")
		assert.Error(t, err)
		assert.Contains(t, output.String(), "Secrets detected")

		// Get the result IDs from the output
		// Note: In a real test, you would need to parse the actual result IDs from the scan output
		resultIds := "mock-id-1,mock-id-2"

		// Ignore specific secrets
		output = executeCmdNilAssertion(t, "pre-commit ignore specific secrets should not fail",
			"hooks", "pre-commit", "secrets-ignore", "--resultIds", resultIds)
		assert.Contains(t, output.String(), "Added new IDs to .checkmarx_ignore")

		// Run scan again - should pass
		output = executeCmdNilAssertion(t, "pre-commit scan should pass after ignoring specific secrets",
			"hooks", "pre-commit", "secrets-scan")
		assert.Contains(t, output.String(), "No secrets detected")
	})
}

func setupTempDir(t *testing.T) (tmpDir string, cleanup func()) {
	origWD, err := os.Getwd()
	assert.NoError(t, err)

	// Create a temporary directory for the test
	tmpDir = t.TempDir()

	// Change working directory to the temporary directory
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	// Return a cleanup function to restore the original working directory
	cleanup = func() {
		assert.NoError(t, os.Chdir(origWD))
	}
	return
}
