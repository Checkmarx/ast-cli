//go:build !integration

package util

import (
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type fakeSecretStore struct{ m map[string]string }

func (f *fakeSecretStore) SetSecret(key, value string) error {
	f.m[key] = value
	return nil
}
func (f *fakeSecretStore) DeleteSecret(key string) error {
	delete(f.m, key)
	return nil
}

func runSet(t *testing.T, name, value string) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String(propNameFlag, "", "")
	cmd.Flags().String(propValFlag, "", "")
	_ = cmd.Flags().Set(propNameFlag, name)
	_ = cmd.Flags().Set(propValFlag, value)
	if err := runSetValue()(cmd, nil); err != nil {
		t.Fatalf("runSetValue(%s) err %v", name, err)
	}
}

// Secret prop routes to the credential store, not plaintext yaml.
func TestConfigureSet_SecretRoutesToStore(t *testing.T) {
	viper.SetConfigFile(filepath.Join(t.TempDir(), "cx.yaml"))
	store := &fakeSecretStore{m: map[string]string{}}
	prev := configuration.Secrets
	configuration.Secrets = store
	t.Cleanup(func() { configuration.Secrets = prev })

	runSet(t, params.AstAPIKey, "tok")

	if store.m[params.AstAPIKey] != "tok" {
		t.Errorf("expected secret in store, got %q", store.m[params.AstAPIKey])
	}
	if got := viper.GetString(params.AstAPIKey); got != "" {
		t.Errorf("expected yaml blanked, got %q", got)
	}
}

// Non-secret prop routes to plain yaml config.
func TestConfigureSet_NonSecretRoutesToYaml(t *testing.T) {
	viper.SetConfigFile(filepath.Join(t.TempDir(), "cx.yaml"))
	prev := configuration.Secrets
	configuration.Secrets = &fakeSecretStore{m: map[string]string{}}
	t.Cleanup(func() { configuration.Secrets = prev })

	runSet(t, params.BaseURIKey, "https://example")

	if got := viper.GetString(params.BaseURIKey); got != "https://example" {
		t.Errorf("expected yaml write, got %q", got)
	}
}
