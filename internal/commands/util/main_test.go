package util

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/params"
)

// TestMain isolates configuration tests from the user's real config file.
func TestMain(m *testing.M) {
	testConfigDir, err := os.MkdirTemp("", "cx-util-test-config")
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
