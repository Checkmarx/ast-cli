package wrappers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/pkg/errors"
)

// pkceScopes matches the scopes requested by the Checkmarx One VS Code
// extension's OAuth flow. The ast-api / iam-api scopes are configured as
// default client scopes on the ide-integration Keycloak client, so they
// are granted automatically without being requested explicitly.
const pkceScopes = "openid offline_access"

const pkceLoginTimeout = 5 * time.Minute

type PKCETokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type PKCELoginOptions struct {
	RealmURL    string
	ClientID    string
	Port        int
	OpenBrowser bool
}

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// LoginWithPKCE runs an OAuth 2.0 Authorization Code + PKCE flow against the
// Keycloak realm at opts.RealmURL and returns the token response. The flow
// starts a one-shot HTTP listener on 127.0.0.1, opens the user's browser to
// the authorize URL, and waits for the redirect callback. The caller is
// responsible for persisting or printing the returned tokens.
func LoginWithPKCE(ctx context.Context, opts PKCELoginOptions) (*PKCETokenResponse, error) {
	if opts.RealmURL == "" {
		return nil, errors.New("realm URL is required")
	}
	if opts.ClientID == "" {
		return nil, errors.New("client-id is required")
	}

	disco, err := discoverOIDC(ctx, opts.RealmURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch OIDC discovery document")
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate PKCE verifier")
	}
	state, err := randomURLSafe(16)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate state")
	}

	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", opts.Port))
	if err != nil {
		return nil, errors.Wrap(err, "failed to start local callback listener")
	}
	defer listener.Close()
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("local listener did not bind to a TCP address")
	}
	// The redirect URI uses the 'localhost' hostname and the '/checkmarx1/callback'
	// path to match the pattern whitelisted on the 'ide-integration' Keycloak client
	// — the same pattern used by the Checkmarx One VS Code extension. Because
	// 'localhost' resolves to ::1 on IPv6-preferring systems, we ALSO bind an IPv6
	// loopback listener on the same port below (best-effort), so the browser callback
	// reaches us whichever family 'localhost' resolves to. Both listeners are
	// loopback-only (safe).
	redirectURI := fmt.Sprintf("http://localhost:%d/checkmarx1/callback", tcpAddr.Port)
	authURL := buildAuthorizeURL(disco.AuthorizationEndpoint, opts.ClientID, redirectURI, state, challenge)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/checkmarx1/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			writeBrowserMessage(w, "Authentication failed.", fmt.Sprintf("%s: %s", errParam, desc))
			resultCh <- callbackResult{err: errors.Errorf("authorization server returned error: %s — %s", errParam, desc)}
			return
		}
		if got := q.Get("state"); got != state {
			writeBrowserMessage(w, "Authentication failed.", "State mismatch — possible CSRF. You can close this tab.")
			resultCh <- callbackResult{err: errors.New("state mismatch in callback — possible CSRF")}
			return
		}
		code := q.Get("code")
		if code == "" {
			writeBrowserMessage(w, "Authentication failed.", "Missing authorization code in callback.")
			resultCh <- callbackResult{err: errors.New("missing authorization code in callback")}
			return
		}
		writeBrowserMessage(w, "Authentication successful.", "You can close this tab and return to the terminal.")
		resultCh <- callbackResult{code: code}
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	// Best-effort IPv6 loopback listener on the same port, so a browser that resolves
	// 'localhost' to ::1 still reaches the callback. If it fails (no IPv6 stack, or the
	// port is taken on ::1), proceed IPv4-only — unchanged from the previous behavior.
	if v6Listener, v6Err := net.Listen("tcp6", fmt.Sprintf("[::1]:%d", tcpAddr.Port)); v6Err == nil {
		defer v6Listener.Close()
		go func() { _ = server.Serve(v6Listener) }()
	} else {
		logger.PrintIfVerbose("OAuth callback: IPv6 loopback listener unavailable, using IPv4 only: " + v6Err.Error())
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Diagnostic messages go to stderr so session mode's eval-able stdout
	// (the env-var assignment line emitted by the caller after this returns)
	// is not polluted. In default mode the diagnostics are still visible in
	// the terminal since stderr renders to the console.
	fmt.Fprintf(os.Stderr, "Opening browser to: %s\n", authURL)
	fmt.Fprintln(os.Stderr, "If the browser does not open, copy and paste the URL above.")
	if opts.OpenBrowser {
		if err := openBrowser(authURL); err != nil {
			logger.PrintIfVerbose("Failed to open browser automatically: " + err.Error())
		}
	}
	fmt.Fprintln(os.Stderr, "Waiting for authentication...")

	var code string
	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		code = res.code
	case <-time.After(pkceLoginTimeout):
		return nil, errors.Errorf("timed out after %s waiting for authentication", pkceLoginTimeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return exchangeCodeForToken(ctx, disco.TokenEndpoint, opts.ClientID, code, verifier, redirectURI)
}

func discoverOIDC(ctx context.Context, realmURL string) (*oidcDiscovery, error) {
	discoURL := strings.TrimRight(realmURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoURL, nil)
	if err != nil {
		return nil, err
	}
	client := GetClient(15)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.Errorf("realm not found at %s — check --tenant and --base-auth-uri", discoURL)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, errors.Wrap(err, "failed to decode discovery document")
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return nil, errors.New("discovery document is missing authorization_endpoint or token_endpoint")
	}
	return &d, nil
}

func buildAuthorizeURL(authEndpoint, clientID, redirectURI, state, challenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", pkceScopes)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return authEndpoint + "?" + q.Encode()
}

func exchangeCodeForToken(ctx context.Context, tokenEndpoint, clientID, code, verifier, redirectURI string) (*PKCETokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := GetClient(30)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tr PKCETokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, errors.Wrap(err, "failed to decode token response")
	}
	if tr.RefreshToken == "" {
		return nil, errors.New("token response did not include a refresh_token — verify that the Keycloak client grants the 'offline_access' scope")
	}
	return &tr, nil
}

// RevokeRefreshToken invalidates the given refresh token at the Keycloak realm
// via the OAuth 2.0 Token Revocation endpoint (RFC 7009). This is deliberately
// the /revoke endpoint and NOT /logout: /logout is RP-initiated logout that
// ends the entire SSO session and would invalidate every token in that
// session — including tokens we want to keep alive in other CLI session modes.
// /revoke targets a single token, leaving sibling tokens in the same session
// untouched, which is what strict storage independence between --session
// modes requires.
//
// Idempotent: a 400 response (token already invalid) is treated as success
// so callers can use this as best-effort cleanup during auto-revoke and
// explicit logout.
func RevokeRefreshToken(ctx context.Context, realmURL, clientID, refreshToken string) error {
	endpoint := strings.TrimRight(realmURL, "/") + "/protocol/openid-connect/revoke"
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("token", refreshToken)
	form.Set("token_type_hint", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := GetClient(15).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errors.Errorf("revoke request failed with status %d: %s", resp.StatusCode, string(body))
}

func newPKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLSafe(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser is a package-level var so tests can intercept the launch and
// simulate the user completing the OAuth flow without a real browser.
var openBrowser = func(targetURL string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL).Start()
	case "darwin":
		return exec.Command("open", targetURL).Start()
	default:
		return exec.Command("xdg-open", targetURL).Start()
	}
}

func writeBrowserMessage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><title>%s</title>
<style>body{font-family:system-ui,sans-serif;max-width:600px;margin:80px auto;padding:0 20px;color:#222}h1{font-size:22px}p{font-size:16px;color:#555}</style>
</head><body><h1>%s</h1><p>%s</p></body></html>`, title, title, body)
}
