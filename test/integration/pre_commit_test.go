//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
)

func TestHooksPreCommitFullIntegration(t *testing.T) {
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")

	// Install pre-commit hook locally
	_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")

	// Verify hook installation
	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	assert.FileExists(t, hookPath, "Hook should be installed")

	// Uninstall pre-commit hook
	_ = executeCmdNilAssertion(t, "Uninstalling pre-commit hook", "hooks", "pre-commit", "secrets-uninstall-git-hook")

	// Verify hook removal
	data, err := os.ReadFile(hookPath)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "cx-secret-detection", "Hook content should be cleared after uninstall")
}

func TestHooksPreCommitLicenseValidation(t *testing.T) {
	// Store original environment variables
	originals := getOriginalEnvVars()

	// Set only the API key to invalid value
	setEnvVars(map[string]string{
		params.AstAPIKeyEnv: invalidAPIKey,
	})

	// Restore original environment variables after test
	defer setEnvVars(originals)

	// Define command arguments for installing the pre-commit hook
	args := []string{
		"hooks", "pre-commit", "secrets-install-git-hook",
	}

	// Execute the command and verify it fails with the expected error
	err, _ := executeCommand(t, args...)
	assert.Error(t, err, "Error validating scan types: Token decoding error: token is malformed: token contains an invalid number of segments")
}

func TestHooksPreCommitSecretDetection(t *testing.T) {
	// Create a temporary directory and initialize git repository
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")

	// Install pre-commit hook
	_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")

	// Copy the mock secret file to the temporary directory
	mockSecretPath := filepath.Join("data", "mock-secret")
	targetPath := filepath.Join(tmpDir, "mock-secret")

	// Read the mock secret file
	secretContent, err := os.ReadFile(mockSecretPath)
	assert.NoError(t, err, "Failed to read mock secret file")

	// Write the content to the target file
	err = os.WriteFile(targetPath, secretContent, 0644)
	assert.NoError(t, err, "Failed to write secret file")

	// Add the file to git
	execCmd(t, tmpDir, "git", "add", "mock-secret")

	// Try to commit the file - should fail due to secret detection
	cmd := exec.Command("git", "commit", "-m", "Add secret file")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	assert.Error(t, err, "Commit should fail due to secret detection")
	assert.Contains(t, string(output), "Secret detection failed", "Error message should indicate secret detection failure")
}

// Helper functions
func execCmd(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "Failed command %s: %s", strings.Join(cmd.Args, " "), string(output))
}

func parseResultIDs(output string) []string {
	// Mock parsing function: Extract IDs from output.
	// Replace with real parsing logic based on actual scan output format
	lines := strings.Split(output, "\n")
	var ids []string
	for _, line := range lines {
		if strings.Contains(line, "Result ID:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				ids = append(ids, strings.TrimSpace(parts[1]))
			}
		}
	}
	return ids
}

func setupTempDir(t *testing.T) (string, func()) {
	originalWD, err := os.Getwd()
	assert.NoError(t, err)

	tmpDir := t.TempDir()
	assert.NoError(t, os.Chdir(tmpDir))

	cleanup := func() {
		assert.NoError(t, os.Chdir(originalWD))
	}

	return tmpDir, cleanup
}
