package mcp

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/credentialstore"
)

// TestMain isolates bridge tests from the real OS keyring and config file.
func TestMain(m *testing.M) {
	testConfigDir, err := os.MkdirTemp("", "cx-mcp-test-config")
	if err != nil {
		log.Fatalf("failed to create test config dir: %v", err)
	}
	configPath := filepath.Join(testConfigDir, "checkmarxcli.yaml")
	if err := os.WriteFile(configPath, nil, 0o600); err != nil {
		log.Fatalf("failed to seed test config file: %v", err)
	}
	_ = os.Setenv(credentialstore.KeyringModeEnvVar, "disabled")
	_ = os.Setenv(params.ConfigFilePathEnv, configPath)
	credentialstore.ResetForTest()
	exitVal := m.Run()
	_ = os.RemoveAll(testConfigDir)
	os.Exit(exitVal)
}
