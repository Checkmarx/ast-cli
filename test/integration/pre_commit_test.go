//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestHooksPreCommitInstallAndUninstallPreCommitHook(t *testing.T) {
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

func TestHooksPreCommitValidateLicenseIntegration(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{
		AIEnabled: mock.AIProtectionDisabled,
	}

	err := validateLicense(mockJWT)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "License validation failed")
}

func TestHooksPreCommitUpdatePreCommitHook(t *testing.T) {
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")

	// Install pre-commit hook locally
	_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")

	// Update pre-commit hook
	output := executeCmdNilAssertion(t, "Updating pre-commit hook", "hooks", "pre-commit", "secrets-update-git-hook")
	assert.Contains(t, output, "updated successfully", "Hook should update successfully")
}

// Helper functions
func execCmd(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "Failed command %s: %s", strings.Join(cmd.Args, " "), string(output))
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
