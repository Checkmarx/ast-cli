package credentialstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
)

const migratedAPIKey = "old-api-key"

// readStoredValueForTest reads a key directly from the config file on disk.
func readStoredValueForTest(t *testing.T, configPath, key string) string {
	t.Helper()
	config, err := configfile.Load(configPath)
	assert.NoError(t, err)
	return stringValue(config[key])
}

func TestRunMigrationFirstRunMigratesAndRemoves(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\ncx_base_uri: https://keep.example.com\n")
	store := newFakeStore()
	resolver := NewResolver(configPath, PolicyAuto, store)

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Empty(t, value)
	stored, err := store.Get(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, migratedAPIKey, stored)

	baseURI := readStoredValueForTest(t, configPath, "cx_base_uri")
	assert.Equal(t, "https://keep.example.com", baseURI)
}

func TestRunMigrationIdempotentSecondRun(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	resolver := NewResolver(configPath, PolicyAuto, newFakeStore())

	resolver.RunMigration(context.Background())
	resolver.RunMigration(context.Background())

	content := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Empty(t, content)
}

func TestRunMigrationAlreadyMigratedRemovesOnly(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	store := newFakeStore()
	store.values[CredentialAPIKey] = migratedAPIKey
	resolver := NewResolver(configPath, PolicyAuto, store)

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Empty(t, value)
	stored, err := store.Get(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, migratedAPIKey, stored)
}

func TestRunMigrationConflictKeepsYAMLAndStore(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	store := newFakeStore()
	store.values[CredentialAPIKey] = "different-key"
	resolver := NewResolver(configPath, PolicyAuto, store)

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Equal(t, migratedAPIKey, value)
	stored, err := store.Get(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "different-key", stored)
}

func TestRunMigrationSetFailureKeepsYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	store := newFakeStore()
	store.setErr = errors.New("keyring write refused")
	resolver := NewResolver(configPath, PolicyAuto, store)

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Equal(t, migratedAPIKey, value)
}

func TestRunMigrationDisabledIsNoOp(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	store := newFakeStore()
	resolver := NewResolver(configPath, PolicyDisabled, store)

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Equal(t, migratedAPIKey, value)
	assert.Equal(t, 0, store.calls())
}

type recordingProvider struct {
	mu       sync.Mutex
	accounts []string
	values   map[string]string
}

func (r *recordingProvider) Get(_ context.Context, _, account string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts = append(r.accounts, account)
	value, ok := r.values[account]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (r *recordingProvider) Set(_ context.Context, _, account, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = make(map[string]string)
	}
	r.values[account] = value
	return nil
}

func (r *recordingProvider) Delete(_ context.Context, _, _ string) error {
	return nil
}

func TestMigrationAccountsIsolatedPerConfigFile(t *testing.T) {
	pathA := filepath.Join(t.TempDir(), "a", "checkmarxcli.yaml")
	pathB := filepath.Join(t.TempDir(), "b", "checkmarxcli.yaml")
	canonicalA := CanonicalConfigPath(pathA)
	canonicalB := CanonicalConfigPath(pathB)

	assert.NotEqual(
		t,
		AccountFor(canonicalA, CredentialAPIKey),
		AccountFor(canonicalB, CredentialAPIKey),
	)

	backendA := &recordingProvider{}
	backendB := &recordingProvider{}
	storeA := &keyCredentialStore{canonicalPath: canonicalA, backend: backendA}
	storeB := &keyCredentialStore{canonicalPath: canonicalB, backend: backendB}

	_, _ = storeA.Get(context.Background(), CredentialAPIKey)
	_, _ = storeB.Get(context.Background(), CredentialAPIKey)

	accountA := backendA.accounts[0]
	accountB := backendB.accounts[0]
	assert.NotEqual(t, accountA, accountB)
	assert.True(t, strings.HasSuffix(accountA, accountSeparator+CredentialAPIKey))
	assert.NotContains(t, accountA, canonicalA)
	assert.NotContains(t, accountB, canonicalB)
}

// A numeric YAML scalar must surface through stringValue's default branch.
func TestResolveNumericPlaintextScalar(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "checkmarxcli.yaml")
	assert.NoError(t, os.WriteFile(yamlPath, []byte("cx_apikey: 12345\n"), 0o600))
	resolver := NewResolver(yamlPath, PolicyAuto, newFakeStore())

	value, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "12345", value)
}

// An unreadable config file (a directory path) must no-op migration quietly.
func TestRunMigrationUnreadableConfigIsNoop(t *testing.T) {
	resolver := NewResolver(t.TempDir(), PolicyAuto, newFakeStore())
	assert.NotPanics(t, func() { resolver.RunMigration(context.Background()) })
}
