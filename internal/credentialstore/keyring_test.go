package credentialstore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	keyring "github.com/zalando/go-keyring"
)

func TestOSKeyringRoundtrip(t *testing.T) {
	keyring.MockInit()
	provider := osKeyring{}
	ctx := context.Background()
	service := fmt.Sprintf("ast-cli-test-%s", t.Name())
	account := "account"

	_, err := provider.Get(ctx, service, account)
	assert.ErrorIs(t, err, ErrNotFound)

	assert.NoError(t, provider.Set(ctx, service, account, "secret-value"))
	got, err := provider.Get(ctx, service, account)
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", got)

	assert.NoError(t, provider.Delete(ctx, service, account))
	_, err = provider.Get(ctx, service, account)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, provider.Delete(ctx, service, account), ErrNotFound)
}

func TestOSKeyringEmptyValueTreatedAsNotFound(t *testing.T) {
	keyring.MockInit()
	provider := osKeyring{}
	ctx := context.Background()
	service := fmt.Sprintf("ast-cli-test-%s", t.Name())
	account := "empty"

	keyringSetErr := keyring.Set(service, account, "")
	assert.NoError(t, keyringSetErr)
	_, err := provider.Get(ctx, service, account)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOSKeyringErrorClassification(t *testing.T) {
	cases := []struct {
		raw  error
		want error
	}{
		{raw: errors.New("dbus: failed to connect to socket"), want: ErrKeyringUnavailable},
		{raw: errors.New("service not available"), want: ErrKeyringUnavailable},
		{raw: errors.New("access denied by policy"), want: ErrAccessDenied},
		{raw: errors.New("authentication failed"), want: ErrAccessDenied},
		{raw: errors.New("something entirely unexpected"), want: ErrKeyringFailure},
	}
	for _, tc := range cases {
		t.Run(tc.raw.Error(), func(t *testing.T) {
			keyring.MockInitWithError(tc.raw)
			t.Cleanup(keyring.MockInit)
			provider := osKeyring{}
			ctx := context.Background()

			_, err := provider.Get(ctx, "svc", "acct")
			assert.ErrorIs(t, err, tc.want)
			assert.NotErrorIs(t, err, ErrNotFound)

			err = provider.Set(ctx, "svc", "acct", "value")
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

// TestKeyringBackendContextCanceled exercises the guard's ctx branch of every
// operation without waiting on the fixed op timeout: a pre-canceled context
// must surface as ErrKeyringUnavailable wrapping the context error.
func TestKeyringBackendContextCanceled(t *testing.T) {
	keyring.MockInit()
	provider := osKeyring{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Get(ctx, "svc", "acct")
	assert.ErrorIs(t, err, ErrKeyringUnavailable)
	assert.ErrorIs(t, err, context.Canceled)

	err = provider.Set(ctx, "svc", "acct", "value")
	assert.ErrorIs(t, err, ErrKeyringUnavailable)
	assert.ErrorIs(t, err, context.Canceled)

	err = provider.Delete(ctx, "svc", "acct")
	assert.ErrorIs(t, err, ErrKeyringUnavailable)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestOSKeyringClassificationSweepsAllIndicators(t *testing.T) {
	cases := []struct {
		raw  error
		want error
	}{
		{raw: errors.New("service not available"), want: ErrKeyringUnavailable},
		{raw: errors.New("keyring unavailable"), want: ErrKeyringUnavailable},
		{raw: errors.New("connection timeout"), want: ErrKeyringUnavailable},
		{raw: errors.New("failed to connect to bus"), want: ErrKeyringUnavailable},
	}
	for _, tc := range cases {
		keyring.MockInitWithError(tc.raw)
		t.Cleanup(keyring.MockInit)
		provider := osKeyring{}
		err := provider.Set(context.Background(), "svc", "acct", "v")
		assert.ErrorIs(t, err, tc.want)
	}
}
