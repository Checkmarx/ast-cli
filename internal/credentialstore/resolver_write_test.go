package credentialstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

// PolicyDisabled must route writes to the config-file layer so reads and
// writes land in the same layer — the keyring is never touched.
func TestStoreDisabledWritesYAMLNotKeyring(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "checkmarxcli.yaml")
	resolver := NewResolver(yamlPath, PolicyDisabled, nil)

	assert.NoError(t, resolver.Store(context.Background(), CredentialAPIKey, "yaml-value"))

	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "cx_apikey: yaml-value")

	value, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "yaml-value", value)
}

func TestClearDisabledRemovesYAMLAndReportsMissing(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "checkmarxcli.yaml")
	resolver := NewResolver(yamlPath, PolicyDisabled, nil)
	ctx := context.Background()

	assert.ErrorIs(t, resolver.Clear(ctx, CredentialClientSecret), ErrNotFound)

	assert.NoError(t, resolver.Store(ctx, CredentialClientSecret, "to-be-cleared"))
	assert.NoError(t, resolver.Clear(ctx, CredentialClientSecret))

	value, err := resolver.Resolve(ctx, CredentialClientSecret)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, value)
}

// PolicyRequired round-trips through the keyring store only; the YAML layer
// is neither read nor written.
func TestStoreRequiredRoundTripIgnoresYAML(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := NewCredentialStore(CanonicalConfigPath(t.TempDir()))
	resolver := NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyRequired, store)
	ctx := context.Background()

	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "required-value"))

	if _, err := os.Stat(resolver.filePath); !os.IsNotExist(err) {
		t.Fatalf("required mode must not create the YAML file")
	}
	value, err := resolver.Resolve(ctx, CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "required-value", value)
}

// The default (auto) policy keeps writing to the keyring; plaintext removal is
// the caller's responsibility (setConfigProperty/persistLogin gate it on
// StoresInConfigFile).
func TestStoreAutoWritesKeyringOnly(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "checkmarxcli.yaml")
	assert.NoError(t, os.WriteFile(yamlPath, []byte("cx_apikey: old\n"), 0o600))
	resolver := NewResolver(yamlPath, PolicyAuto, nil)

	assert.NoError(t, resolver.Store(context.Background(), CredentialAPIKey, "auto-value"))
	assert.False(t, resolver.StoresInConfigFile())

	value, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "auto-value", value)
}

func TestStoresInConfigFileMatrix(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, NewResolver(filepath.Join(dir, "a.yaml"), PolicyAuto, nil).StoresInConfigFile())
	assert.False(t, NewResolver(filepath.Join(dir, "b.yaml"), PolicyRequired, nil).StoresInConfigFile())
	assert.True(t, NewResolver(filepath.Join(dir, "c.yaml"), PolicyDisabled, nil).StoresInConfigFile())
}

func TestStoreAndClearRejectUnknownCredentialName(t *testing.T) {
	resolver := NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyAuto, nil)
	ctx := context.Background()

	assert.ErrorIs(t, resolver.Store(ctx, "not-a-slot", "v"), ErrInvalidName)
	assert.ErrorIs(t, resolver.Clear(ctx, "not-a-slot"), ErrInvalidName)
}

func TestPackageLevelResolveRejectsUnknownCredentialName(t *testing.T) {
	t.Setenv("CX_CONFIG_FILE_PATH", filepath.Join(t.TempDir(), "checkmarxcli.yaml"))
	ResetForTest()
	t.Cleanup(ResetForTest)

	_, err := Resolve("nope")
	assert.ErrorIs(t, err, ErrInvalidName)
}

func TestRemoveConfigFileEntryRemovesPlaintextSlot(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "checkmarxcli.yaml")
	assert.NoError(t, os.WriteFile(yamlPath, []byte("cx_client_secret: plaintext\n"), 0o600))
	resolver := NewResolver(yamlPath, PolicyRequired, nil)

	assert.NoError(t, resolver.RemoveConfigFileEntry(CredentialClientSecret))

	config, err := configfile.Load(yamlPath)
	assert.NoError(t, err)
	assert.NotContains(t, config, "cx_client_secret")
}

func TestStoreDisabledMissingParentDirFails(t *testing.T) {
	resolver := NewResolver(filepath.Join(t.TempDir(), "no-such-dir", "checkmarxcli.yaml"), PolicyDisabled, nil)
	assert.Error(t, resolver.Store(context.Background(), CredentialAPIKey, "v"))
}

func TestClearDisabledMissingParentDirFails(t *testing.T) {
	resolver := NewResolver(filepath.Join(t.TempDir(), "no-such-dir", "checkmarxcli.yaml"), PolicyDisabled, nil)
	err := resolver.Clear(context.Background(), CredentialAPIKey)
	assert.ErrorIs(t, err, ErrNotFound)
}

// Auto mode surfaces a config-file read failure instead of masking it as
// not-found once the keyring layer also misses.
func TestResolveAutoConfigFileReadErrorSurfaces(t *testing.T) {
	dir := t.TempDir()
	resolver := NewResolver(filepath.Join(dir, "checkmarxcli.yaml"), PolicyAuto, newFakeStore())
	_ = os.MkdirAll(resolver.filePath, 0o700) // path exists but is a directory

	_, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.Error(t, err)
}
