//go:build integration

package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHooksPreCommitInstallAndUninstallPreCommitHook(t *testing.T) {
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()

	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")

	// Install pre-commit hook locally
	_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")

	// Uninstall pre-commit hook
	_ = executeCmdNilAssertion(t, "Uninstalling cx-secret-detection hook", "hooks", "pre-commit", "secrets-uninstall-git-hook")

}

func TestHooksPreCommitUpdatePreCommitHook(t *testing.T) {
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()
	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")
	// Install pre-commit hook locally
	_ = executeCmdNilAssertion(t, "Installing pre-commit hook", "hooks", "pre-commit", "secrets-install-git-hook")
	// Update pre-commit hook
	_ = executeCmdNilAssertion(t, "Updating pre-commit hook", "hooks", "pre-commit", "secrets-update-git-hook")
	// Uninstall pre-commit hook
	_ = executeCmdNilAssertion(t, "Uninstalling cx-secret-detection hook", "hooks", "pre-commit", "secrets-uninstall-git-hook")
}

func TestHooksPreCommitSecretsIgnore(t *testing.T) {
	tmpDir, cleanup := setupTempDir(t)
	defer cleanup()
	// Initialize Git repository
	execCmd(t, tmpDir, "git", "init")
	// Ignore precommit hook command
	_ = executeCmdNilAssertion(t, "precommit secrets Ignore", "hooks", "pre-commit", "secrets-ignore", "--all")
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
