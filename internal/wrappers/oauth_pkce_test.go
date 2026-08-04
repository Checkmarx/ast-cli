package wrappers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewPKCE_ChallengeIsSHA256OfVerifier(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE returned error: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatal("verifier or challenge is empty")
	}
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != expected {
		t.Errorf("challenge = %q, want %q", challenge, expected)
	}
}

func TestBuildAuthorizeURL_IncludesAllRequiredParams(t *testing.T) {
	authURL := buildAuthorizeURL("https://iam.example.com/auth", "ast-app", "http://127.0.0.1:54321/callback", "state-123", "challenge-abc")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("authURL is not parseable: %v", err)
	}
	q := parsed.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "ast-app",
		"redirect_uri":          "http://127.0.0.1:54321/callback",
		"state":                 "state-123",
		"code_challenge":        "challenge-abc",
		"code_challenge_method": "S256",
	}
	for key, want := range checks {
		if got := q.Get(key); got != want {
			t.Errorf("query[%q] = %q, want %q", key, got, want)
		}
	}
	if got := q.Get("scope"); !strings.Contains(got, "openid") || !strings.Contains(got, "offline_access") {
		t.Errorf("scope %q must include openid and offline_access", got)
	}
}

func TestExchangeCodeForToken_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" {
			http.Error(w, "bad grant_type", http.StatusBadRequest)
			return
		}
		if r.Form.Get("code_verifier") != "the-verifier" {
			http.Error(w, "bad verifier", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "fake-access",
			"refresh_token": "fake-refresh",
			"expires_in":    300,
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	tokens, err := exchangeCodeForToken(context.Background(), srv.URL, "ast-app", "the-code", "the-verifier", "http://127.0.0.1:1/callback")
	if err != nil {
		t.Fatalf("exchangeCodeForToken returned error: %v", err)
	}
	if tokens.RefreshToken != "fake-refresh" || tokens.AccessToken != "fake-access" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
}

func TestExchangeCodeForToken_MissingRefreshToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fake-access",
			"expires_in":   300,
		})
	}))
	defer srv.Close()

	_, err := exchangeCodeForToken(context.Background(), srv.URL, "ast-app", "c", "v", "r")
	if err == nil {
		t.Fatal("expected error on missing refresh_token")
	}
	if !strings.Contains(err.Error(), "offline_access") {
		t.Errorf("error %q should mention offline_access scope", err.Error())
	}
}

func TestExchangeCodeForToken_KeycloakError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant","error_description":"code expired"}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := exchangeCodeForToken(context.Background(), srv.URL, "ast-app", "c", "v", "r")
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error should surface Keycloak's response: %q", err.Error())
	}
}

// End-to-end against a fake Keycloak; openBrowser is stubbed to play the browser.
func TestLoginWithPKCE_HappyPath(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	srv = httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/protocol/openid-connect/auth", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		redirectURI := q.Get("redirect_uri")
		state := q.Get("state")
		go func() {
			resp, err := http.Get(redirectURI + "?code=fake-auth-code&state=" + state)
			if err == nil && resp != nil {
				_ = resp.Body.Close()
			}
		}()
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "fake-auth-code" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "fake-access",
			"refresh_token": "fake-refresh",
			"expires_in":    300,
			"token_type":    "Bearer",
		})
	})

	original := openBrowser
	openBrowser = func(target string) error {
		go func() {
			resp, err := http.Get(target)
			if err == nil && resp != nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	defer func() { openBrowser = original }()

	tokens, err := LoginWithPKCE(context.Background(), PKCELoginOptions{
		RealmURL:    srv.URL,
		ClientID:    "ast-app",
		Port:        0,
		OpenBrowser: true,
	})
	if err != nil {
		t.Fatalf("LoginWithPKCE returned error: %v", err)
	}
	if tokens.RefreshToken != "fake-refresh" {
		t.Errorf("got refresh_token %q, want fake-refresh", tokens.RefreshToken)
	}
}
