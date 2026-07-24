package configuration

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

// fakeSecretStore implements SecretStore in memory, with an optional Set error.
type fakeSecretStore struct {
	m       map[string]string
	failSet bool
}

func newFakeSecretStore() *fakeSecretStore { return &fakeSecretStore{m: map[string]string{}} }

func (f *fakeSecretStore) SetSecret(key, value string) error {
	if f.failSet {
		return errors.New("set boom")
	}
	f.m[key] = value
	return nil
}

func (f *fakeSecretStore) DeleteSecret(key string) error {
	delete(f.m, key)
	return nil
}

// sandboxConfig points viper's config file at a temp path and restores Secrets.
func sandboxConfig(t *testing.T) {
	t.Helper()
	viper.SetConfigFile(filepath.Join(t.TempDir(), "cx.yaml"))
	prev := Secrets
	t.Cleanup(func() { Secrets = prev })
}

func TestSetSecretQuiet_RoutesToStore(t *testing.T) {
	sandboxConfig(t)
	store := newFakeSecretStore()
	Secrets = store

	setSecretQuiet(params.AstAPIKey, "tok")

	if store.m[params.AstAPIKey] != "tok" {
		t.Errorf("store missing value: %q", store.m[params.AstAPIKey])
	}
	if got := viper.GetString(params.AstAPIKey); got != "" {
		t.Errorf("expected yaml key blanked, got %q", got)
	}
}

func TestSetSecretQuiet_FallsBackToYamlOnError(t *testing.T) {
	sandboxConfig(t)
	Secrets = &fakeSecretStore{m: map[string]string{}, failSet: true}

	setSecretQuiet(params.AstAPIKey, "tok")

	if got := viper.GetString(params.AstAPIKey); got != "tok" {
		t.Errorf("expected yaml fallback write, got %q", got)
	}
}

func TestClearSecretQuiet_DeletesAndBlanks(t *testing.T) {
	sandboxConfig(t)
	store := newFakeSecretStore()
	store.m[params.AstAPIKey] = "tok"
	Secrets = store

	clearSecretQuiet(params.AstAPIKey)

	if _, ok := store.m[params.AstAPIKey]; ok {
		t.Errorf("expected store deleted")
	}
	if got := viper.GetString(params.AstAPIKey); got != "" {
		t.Errorf("expected yaml blanked, got %q", got)
	}
}

func TestSetSecretQuiet_NilSecretsWritesYaml(t *testing.T) {
	sandboxConfig(t)
	Secrets = nil

	setSecretQuiet(params.AstAPIKey, "tok")

	if got := viper.GetString(params.AstAPIKey); got != "tok" {
		t.Errorf("expected plain yaml write, got %q", got)
	}
}
