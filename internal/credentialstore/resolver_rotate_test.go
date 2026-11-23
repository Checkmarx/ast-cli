package credentialstore

import (
	"context"
	"path/filepath"
	"testing"

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
