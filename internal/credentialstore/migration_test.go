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
	"github.com/gofrs/flock"
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

// verifyFailStore reports ErrNotFound on the first Get (so migrateOne routes
// into migrateAndRemove) then fails every subsequent Get, so the
// post-Set verification read in migrateAndRemove observes an error.
type verifyFailStore struct {
	gets int
}

func (s *verifyFailStore) Get(context.Context, string) (string, error) {
	s.gets++
	if s.gets == 1 {
		return "", ErrNotFound
	}
	return "", errors.New("keyring read refused")
}

func (s *verifyFailStore) Set(context.Context, string, string) error { return nil }
func (s *verifyFailStore) Delete(context.Context, string) error      { return nil }

func TestRunMigrationVerificationFailureKeepsYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	resolver := NewResolver(configPath, PolicyAuto, &verifyFailStore{})

	resolver.RunMigration(context.Background())

	value := readStoredValueForTest(t, configPath, params.AstAPIKey)
	assert.Equal(t, migratedAPIKey, value)
}

// A config file lock held by another process must not fail migration outright:
// the keyring write already succeeded, so the leftover YAML entry is a
// cosmetic cleanup failure logged verbosely, not a hard error.
func TestRunMigrationConfigFileRemovalLockedIsQuiet(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	lock := flock.New(configPath + ".lock")
	locked, lockErr := lock.TryLock()
	assert.NoError(t, lockErr)
	assert.True(t, locked)
	t.Cleanup(func() { _ = lock.Unlock() })

	resolver := NewResolver(configPath, PolicyAuto, newFakeStore())
	assert.NotPanics(t, func() { resolver.RunMigration(context.Background()) })

	stored, err := resolver.store.Get(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, migratedAPIKey, stored)
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

func TestViperKeyForUnknownCredentialReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", viperKeyFor("unknown"))
}

// migrateOne is only ever called by RunMigration after it has already
// verified the YAML value is non-empty; this pins that guard as belt-and-
// suspenders for any future direct caller.
func TestMigrateOneEmptyYAMLValueIsNoOp(t *testing.T) {
	store := newFakeStore()
	resolver := NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyAuto, store)

	resolver.migrateOne(context.Background(), CredentialAPIKey, map[string]interface{}{})

	assert.Equal(t, 0, store.calls())
}

// An unreadable config file (a directory path) must no-op migration quietly.
func TestRunMigrationUnreadableConfigIsNoop(t *testing.T) {
	resolver := NewResolver(t.TempDir(), PolicyAuto, newFakeStore())
	assert.NotPanics(t, func() { resolver.RunMigration(context.Background()) })
}

// Migrate is the package-level wrapper over Default().RunMigration; this
// pins it to the injected default resolver so it never touches the real
// keyring or the user's actual config file.
func TestMigratePackageLevelWrapper(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	writePlaintextConfig(t, configPath, "cx_apikey: "+migratedAPIKey+"\n")
	store := newFakeStore()
	SetDefaultResolverForTest(NewResolver(configPath, PolicyAuto, store))

	Migrate()

	stored, err := store.Get(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, migratedAPIKey, stored)
	assert.Empty(t, readStoredValueForTest(t, configPath, params.AstAPIKey))
}
