package wrappers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func installRotateResolver(t *testing.T, initial string) *credentialstore.Resolver {
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := credentialstore.NewCredentialStore(credentialstore.CanonicalConfigPath(t.TempDir()))
	resolver := credentialstore.NewResolver(filepath.Join(t.TempDir(), "checkmarxcli.yaml"), credentialstore.PolicyRequired, store)
	credentialstore.SetDefaultResolverForTest(resolver)
	t.Cleanup(credentialstore.ResetForTest)
	assert.NoError(t, resolver.Store(context.Background(), credentialstore.CredentialAPIKey, initial))
	return resolver
}

func tokenServer(t *testing.T, body string, seen *url.Values) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if seen != nil {
			*seen = r.PostForm
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestExchangeAPIKeySavesRotatedRefreshToken(t *testing.T) {
	resolver := installRotateResolver(t, "old-refresh")
	var form url.Values
	server := tokenServer(t, `{"access_token":"new-access","refresh_token":"new-refresh"}`, &form)

	access, err := exchangeAPIKey("old-refresh", server.URL)

	assert.NoError(t, err)
	assert.Equal(t, "new-access", access)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "old-refresh", form.Get("refresh_token"))
	stored, err := resolver.Resolve(context.Background(), credentialstore.CredentialAPIKey)
	assert.NoError(t, err)
	assert.Equal(t, "new-refresh", stored)
}

func TestExchangeAPIKeyKeepsTokenWhenNotRotated(t *testing.T) {
	resolver := installRotateResolver(t, "same-refresh")
	server := tokenServer(t, `{"access_token":"a","refresh_token":"same-refresh"}`, nil)

	_, err := exchangeAPIKey("same-refresh", server.URL)

	assert.NoError(t, err)
	stored, _ := resolver.Resolve(context.Background(), credentialstore.CredentialAPIKey)
	assert.Equal(t, "same-refresh", stored)
}

func TestExchangeAPIKeyKeepsTokenWhenResponseHasNone(t *testing.T) {
	resolver := installRotateResolver(t, "old-refresh")
	server := tokenServer(t, `{"access_token":"a"}`, nil)

	_, err := exchangeAPIKey("old-refresh", server.URL)

	assert.NoError(t, err)
	stored, _ := resolver.Resolve(context.Background(), credentialstore.CredentialAPIKey)
	assert.Equal(t, "old-refresh", stored)
}
