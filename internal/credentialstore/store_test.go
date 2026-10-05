package credentialstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func TestCredentialStoreDeleteRoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := NewCredentialStore(CanonicalConfigPath(filepath.Join(t.TempDir(), "checkmarxcli.yaml")))
	ctx := context.Background()

	assert.ErrorIs(t, store.Delete(ctx, CredentialAPIKey), ErrNotFound)

	assert.NoError(t, store.Set(ctx, CredentialAPIKey, "value"))
	assert.NoError(t, store.Delete(ctx, CredentialAPIKey))

	_, err := store.Get(ctx, CredentialAPIKey)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCredentialStoreRejectsUnknownCredentialName(t *testing.T) {
	store := NewCredentialStore(CanonicalConfigPath(filepath.Join(t.TempDir(), "checkmarxcli.yaml")))
	ctx := context.Background()

	_, err := store.Get(ctx, "unknown")
	assert.ErrorIs(t, err, ErrInvalidName)

	assert.ErrorIs(t, store.Set(ctx, "unknown", "value"), ErrInvalidName)
	assert.ErrorIs(t, store.Delete(ctx, "unknown"), ErrInvalidName)
}
