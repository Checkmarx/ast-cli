package credentialstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/configfile"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

const (
	rotateOldToken = "old-token"
	rotateNewToken = "new-token"
)

func newRotateResolver(t *testing.T) *Resolver {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := NewCredentialStore(CanonicalConfigPath(t.TempDir()))
	return NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyRequired, store)
}

func assertStored(t *testing.T, resolver *Resolver, expected string) {
	value, err := resolver.Resolve(context.Background(), CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, expected, value)
}

func TestRotateReplacesStoredCredential(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, rotateOldToken))

	saved, err := resolver.Rotate(ctx, CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.True(t, saved)
	assertStored(t, resolver, rotateNewToken)
}

func TestRotateSkipsWhenStoredValueChanged(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "fresh-login"))

	saved, err := resolver.Rotate(ctx, CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.False(t, saved)
	assertStored(t, resolver, "fresh-login")
}

func TestRotateNoOpForEmptyOrUnchangedValue(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, rotateOldToken))

	saved, err := resolver.Rotate(ctx, CredentialAPIKey, rotateOldToken, "")
	assert.NoError(t, err)
	assert.False(t, saved)
	saved, err = resolver.Rotate(ctx, CredentialAPIKey, rotateOldToken, rotateOldToken)
	assert.NoError(t, err)
	assert.False(t, saved)

	assertStored(t, resolver, rotateOldToken)
}

func TestRotateDoesNotWriteWhenExplicitOverrideSupplies(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "stored"))
	resolver.SetExplicit(CredentialAPIKey, "from-flag")

	saved, err := resolver.Rotate(ctx, CredentialAPIKey, "from-flag", rotateNewToken)

	assert.NoError(t, err)
	assert.False(t, saved)
	resolver.ResetExplicit()
	assertStored(t, resolver, "stored")
}

func TestRotateDoesNotWriteWhenEnvVarSupplies(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "stored"))
	t.Setenv("CX_APIKEY", "from-env")

	saved, err := resolver.Rotate(ctx, CredentialAPIKey, "from-env", rotateNewToken)

	assert.NoError(t, err)
	assert.False(t, saved)
	t.Setenv("CX_APIKEY", "")
	assertStored(t, resolver, "stored")
}

func TestRotateRejectsInvalidName(t *testing.T) {
	resolver := newRotateResolver(t)
	saved, err := resolver.Rotate(context.Background(), "bogus", "a", "b")
	assert.ErrorIs(t, err, ErrInvalidName)
	assert.False(t, saved)
}

// failingStore is a keyring stand-in whose reads and writes fail with err.
type failingStore struct {
	err error
}

func (s failingStore) Get(_ context.Context, _ string) (string, error) { return "", s.err }

func (s failingStore) Set(_ context.Context, _, _ string) error { return s.err }

func (s failingStore) Delete(_ context.Context, _ string) error { return s.err }

func newConfigFileResolver(t *testing.T, policy Policy, store CredentialStore, stored string) (resolver *Resolver, path string) {
	path = filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	if stored != "" {
		assert.NoError(t, configfile.SetKey(path, viperKeyFor(CredentialAPIKey), stored))
	}
	return NewResolver(path, policy, store), path
}

func storedInFile(t *testing.T, path string) interface{} {
	config, err := configfile.Load(path)
	assert.NoError(t, err)
	return config[viperKeyFor(CredentialAPIKey)]
}

func TestRotateDisabledReplacesConfigFileValue(t *testing.T) {
	resolver, path := newConfigFileResolver(t, PolicyDisabled, nil, rotateOldToken)

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.True(t, saved)
	assert.Equal(t, rotateNewToken, storedInFile(t, path))
}

func TestRotateDisabledSkipsWhenFileValueChanged(t *testing.T) {
	resolver, path := newConfigFileResolver(t, PolicyDisabled, nil, "fresh-login")

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.False(t, saved)
	assert.Equal(t, "fresh-login", storedInFile(t, path))
}

func TestRotateAutoWorkingKeyringReplacesKeyringValue(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := NewCredentialStore(CanonicalConfigPath(t.TempDir()))
	resolver, path := newConfigFileResolver(t, PolicyAuto, store, "")
	assert.NoError(t, resolver.Store(context.Background(), CredentialAPIKey, rotateOldToken))

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.True(t, saved)
	assertStored(t, resolver, rotateNewToken)
	assert.Nil(t, storedInFile(t, path))
}

func TestRotateAutoKeyringUnavailableFallsBackToConfigFile(t *testing.T) {
	store := failingStore{err: ErrKeyringUnavailable}
	resolver, path := newConfigFileResolver(t, PolicyAuto, store, rotateOldToken)

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.NoError(t, err)
	assert.True(t, saved)
	assert.Equal(t, rotateNewToken, storedInFile(t, path))
}

func TestRotateAutoKeyringUnavailableNeverCreatesPlaintextEntry(t *testing.T) {
	store := failingStore{err: ErrKeyringUnavailable}
	resolver, path := newConfigFileResolver(t, PolicyAuto, store, "")

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.Error(t, err)
	assert.False(t, saved)
	assert.Nil(t, storedInFile(t, path))
}

func TestRotateRequiredReportsKeyringWriteFailure(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	good := NewCredentialStore(CanonicalConfigPath(t.TempDir()))
	resolver, _ := newConfigFileResolver(t, PolicyRequired, good, "")
	assert.NoError(t, resolver.Store(context.Background(), CredentialAPIKey, rotateOldToken))
	resolver.store = writeFailingStore{CredentialStore: good, err: ErrAccessDenied}

	saved, err := resolver.Rotate(context.Background(), CredentialAPIKey, rotateOldToken, rotateNewToken)

	assert.ErrorIs(t, err, ErrAccessDenied)
	assert.False(t, saved)
}

// writeFailingStore reads through to the wrapped store but fails every write.
type writeFailingStore struct {
	CredentialStore
	err error
}

func (s writeFailingStore) Set(_ context.Context, _, _ string) error { return s.err }
