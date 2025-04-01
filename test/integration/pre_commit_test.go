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

	//	// Create file with mock secrets
	//	fileContent := `MOCK SECRET
	//ghp_mocksecretAAAAAAAAAAAAAAAAAAAAAAAAA
	//ANOTHER MOCK SECRET`
	//	secretFilePath := filepath.Join(tmpDir, "secret.txt")
	//	assert.NoError(t, os.WriteFile(secretFilePath, []byte(fileContent), 0644))
	//
	//	// Stage the file
	//	execCmd(t, tmpDir, "git", "add", "secret.txt")
	//
	//	// Run secrets scan (expect failure due to secret detection)
	//	err, output := executeCommand(t, "hooks", "pre-commit", "secrets-scan")
	//	assert.Error(t, err, "Scan should fail due to secret")
	//	assert.Contains(t, output.String(), "Secret detected")
	//
	//	// Extract result IDs from output
	//	resultIds := parseResultIDs(output.String())
	//	assert.NotEmpty(t, resultIds, "Should detect result IDs")
	//
	//	// Ignore detected secrets by ID
	//	output = executeCmdNilAssertion(t, "Ignoring detected secrets", "hooks", "pre-commit", "secrets-ignore", "--resultIds", strings.Join(resultIds, ","))
	//	assert.Contains(t, output.String(), "Added new IDs to .checkmarx_ignore")
	//
	//	// Run secrets scan again (expect success after ignoring)
	//	output = executeCmdNilAssertion(t, "Running secrets scan after ignoring", "hooks", "pre-commit", "secrets-scan")
	//	assert.Contains(t, output.String(), "No secrets detected")

	// Uninstall pre-commit hook
	_ = executeCmdNilAssertion(t, "Uninstalling pre-commit hook", "hooks", "pre-commit", "secrets-uninstall-git-hook")

	// Verify hook removal
	data, err := os.ReadFile(hookPath)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "cx-secret-detection", "Hook content should be cleared after uninstall")
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
