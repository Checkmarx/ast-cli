//go:build !integration

package mcp

import (
	"path/filepath"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/credentialstore"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/viper"
)

// A degraded bridge picks up a token that later appears in the keyring: reloadConfig
// runs LoadStoredSecrets, and productionResolveAPIKey then resolves the new token.
func TestReloadConfig_HealsFromStore(t *testing.T) {
	// Point config loading at a nonexistent file so LoadConfiguration is a no-op
	// and never reads the real user config.
	viper.Set(commonParams.ConfigFilePathKey, filepath.Join(t.TempDir(), "absent.yaml"))
	viper.Set(commonParams.AstAPIKey, "")
	t.Setenv(commonParams.AstAPIKeyEnv, "")
	t.Cleanup(func() {
		viper.Set(commonParams.ConfigFilePathKey, "")
		viper.Set(commonParams.AstAPIKey, "")
	})

	store := mock.NewCredentialStoreMock()
	_ = store.SetSecret(commonParams.AstAPIKey, "healed-token")
	prev := credentialstore.Default
	credentialstore.Default = store
	t.Cleanup(func() { credentialstore.Default = prev })

	if got := productionResolveAPIKey(); got == "healed-token" {
		t.Fatalf("precondition: token already resolved before reload")
	}

	reloadConfig()

	if got := productionResolveAPIKey(); got != "healed-token" {
		t.Errorf("expected healed-token after reload, got %q", got)
	}
}
