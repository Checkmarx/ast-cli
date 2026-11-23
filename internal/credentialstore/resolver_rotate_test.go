package credentialstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func newRotateResolver(t *testing.T) *Resolver {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := NewCredentialStore(CanonicalConfigPath(t.TempDir()))
	return NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), PolicyRequired, store)
}

func TestRotateReplacesStoredCredential(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "old"))

	assert.NoError(t, resolver.Rotate(ctx, CredentialAPIKey, "old", "new"))

	value, err := resolver.Resolve(ctx, CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "new", value)
}

func TestRotateSkipsWhenStoredValueChanged(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "fresh-login"))

	assert.NoError(t, resolver.Rotate(ctx, CredentialAPIKey, "old", "new"))

	value, _ := resolver.Resolve(ctx, CredentialAPIKey)
	assert.Equal(t, "fresh-login", value)
}

func TestRotateNoOpForEmptyOrUnchangedValue(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "old"))

	assert.NoError(t, resolver.Rotate(ctx, CredentialAPIKey, "old", ""))
	assert.NoError(t, resolver.Rotate(ctx, CredentialAPIKey, "old", "old"))

	value, _ := resolver.Resolve(ctx, CredentialAPIKey)
	assert.Equal(t, "old", value)
}

func TestRotateDoesNotWriteWhenExplicitOverrideSupplies(t *testing.T) {
	resolver := newRotateResolver(t)
	ctx := context.Background()
	assert.NoError(t, resolver.Store(ctx, CredentialAPIKey, "stored"))
	resolver.SetExplicit(CredentialAPIKey, "from-flag")

	assert.NoError(t, resolver.Rotate(ctx, CredentialAPIKey, "from-flag", "new"))

	resolver.ResetExplicit()
	value, _ := resolver.Resolve(ctx, CredentialAPIKey)
	assert.Equal(t, "stored", value)
}

func TestRotateRejectsInvalidName(t *testing.T) {
	resolver := newRotateResolver(t)
	assert.ErrorIs(t, resolver.Rotate(context.Background(), "bogus", "a", "b"), ErrInvalidName)
}
