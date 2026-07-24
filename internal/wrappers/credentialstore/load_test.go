package credentialstore

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

func resetViperSecrets() {
	viper.Set(params.AstAPIKey, "")
	viper.Set(params.AccessKeySecretConfigKey, "")
}

func swapDefault(t *testing.T, s CredentialStore) {
	t.Helper()
	prev := Default
	Default = s
	t.Cleanup(func() { Default = prev })
}

func TestLoadStoredSecrets_CopiesIntoViper(t *testing.T) {
	resetViperSecrets()
	t.Setenv(params.AstAPIKeyEnv, "")
	t.Setenv(params.AccessKeySecretEnv, "")

	store := newFakeStore()
	store.m[params.AstAPIKey] = "stored-apikey"
	store.m[params.AccessKeySecretConfigKey] = "stored-secret"
	swapDefault(t, store)

	LoadStoredSecrets()

	if got := viper.GetString(params.AstAPIKey); got != "stored-apikey" {
		t.Errorf("apikey: got %q", got)
	}
	if got := viper.GetString(params.AccessKeySecretConfigKey); got != "stored-secret" {
		t.Errorf("secret: got %q", got)
	}
}

func TestLoadStoredSecrets_EnvWins(t *testing.T) {
	resetViperSecrets()
	t.Setenv(params.AstAPIKeyEnv, "env-value")

	store := newFakeStore()
	store.m[params.AstAPIKey] = "stored-apikey"
	swapDefault(t, store)

	LoadStoredSecrets()

	if got := viper.GetString(params.AstAPIKey); got == "stored-apikey" {
		t.Errorf("env should win, stored value was copied: %q", got)
	}
}

func TestLoadStoredSecrets_KeyringWinsOverYaml(t *testing.T) {
	resetViperSecrets()
	t.Setenv(params.AstAPIKeyEnv, "")
	t.Setenv(params.AccessKeySecretEnv, "")

	keyringLike := newFakeStore()
	yamlLike := newFakeStore()
	keyringLike.m[params.AstAPIKey] = "from-keyring"
	yamlLike.m[params.AstAPIKey] = "from-yaml"
	swapDefault(t, NewChainStore(keyringLike, yamlLike))

	LoadStoredSecrets()

	if got := viper.GetString(params.AstAPIKey); got != "from-keyring" {
		t.Errorf("keyring should win, got %q", got)
	}
}
