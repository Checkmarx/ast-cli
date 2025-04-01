//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")

	//// Test case 1: Without license (should fail)
	//t.Run("Without Enterprise License", func(t *testing.T) {
	//	// Set environment to simulate no license
	//	originalEnv := os.Getenv("CX_LICENSE")
	//	os.Setenv("CX_LICENSE", "")
	//	defer os.Setenv("CX_LICENSE", originalEnv)
	//
	//	// Try to install pre-commit hook
	//	args := []string{"hooks", "pre-commit", "secrets-install-git-hook"}
	//	err, output := executeCommand(t, args...)
	//
	//	// Verify error message
	//	assert.Error(t, err)
	//	assert.Contains(t, output.String(), "License validation failed")
	//	assert.Contains(t, output.String(), "Please verify your CxOne License includes Enterprise Secrets")
	//
	//	// Verify hook was not installed
	//	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	//	_, err = os.Stat(hookPath)
	//	assert.True(t, os.IsNotExist(err), "Hook should not be installed without license")
	//})

	// Test case 2: With license (should succeed)
	t.Run("With Enterprise License", func(t *testing.T) {
		// Set environment to simulate valid license
		originalEnv := os.Getenv("CX_LICENSE")
		os.Setenv("CX_LICENSE", "valid-license-key") // Replace with actual license
		defer os.Setenv("CX_LICENSE", originalEnv)

		// Install pre-commit hook
		_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")

		// Verify hook installation
		hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
		assert.FileExists(t, hookPath, "Hook should be installed with valid license")

		// Verify hook content
		data, err := os.ReadFile(hookPath)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "cx-secret-detection", "Hook should contain secret detection code")

		// Clean up - uninstall hook
		_ = executeCmdNilAssertion(t, "Uninstalling pre-commit hook", "hooks", "pre-commit", "secrets-uninstall-git-hook")
	})
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
