package wrappers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
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

// Keycloak endpoints built directly from the realm URL, matching the IDE
// plugins (no OIDC .well-known discovery).
const (
	authorizeEndpointSuffix = "protocol/openid-connect/auth"
	tokenEndpointSuffix     = "protocol/openid-connect/token"
)

// pkceScopes matches the ide-integration client's scopes; offline_access yields
// the refresh token stored as cx_apikey.
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

// LoginWithPKCE runs an OAuth 2.0 Authorization Code + PKCE flow against the
// realm: opens the browser, listens on a loopback callback, and exchanges the
// returned code for tokens. The caller persists the result.
func LoginWithPKCE(ctx context.Context, opts PKCELoginOptions) (*PKCETokenResponse, error) {
	if opts.RealmURL == "" {
		return nil, errors.New("realm URL is required")
	}
	if opts.ClientID == "" {
		return nil, errors.New("client-id is required")
	}

	realm := strings.TrimRight(opts.RealmURL, "/")
	authorizationEndpoint := realm + "/" + authorizeEndpointSuffix
	tokenEndpoint := realm + "/" + tokenEndpointSuffix

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
	// /checkmarx1/callback on localhost matches the ide-integration client's
	// whitelisted redirect pattern.
	redirectURI := fmt.Sprintf("http://localhost:%d/checkmarx1/callback", tcpAddr.Port)
	authURL := buildAuthorizeURL(authorizationEndpoint, opts.ClientID, redirectURI, state, challenge)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/checkmarx1/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// Anti-CSRF: ignore any request without our exact state, so a stray/forged
		// callback can't complete or abort the pending login.
		if got := q.Get("state"); got != state {
			writeBrowserMessage(w, "Authentication failed.", "State mismatch — this request was ignored. You can close this tab.")
			logger.PrintIfVerbose("OAuth callback: ignoring request with missing/mismatched state")
			return
		}
		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			writeBrowserMessage(w, "Authentication failed.", fmt.Sprintf("%s: %s", errParam, desc))
			resultCh <- callbackResult{err: errors.Errorf("authorization server returned error: %s — %s", errParam, desc)}
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
	// Best-effort IPv6 loopback so 'localhost' resolving to ::1 still reaches us.
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

	// Diagnostics to stderr so stdout stays clean for callers that capture it.
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

	return exchangeCodeForToken(ctx, tokenEndpoint, opts.ClientID, code, verifier, redirectURI)
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

// openBrowser is a var so tests can intercept the launch.
var openBrowser = func(targetURL string) error {
	switch runtime.GOOS {
	case "windows":
		// `start` reads a quoted first arg as the window title, so pass an empty
		// one; escape & to ^& since cmd treats it as a command separator.
		escaped := strings.ReplaceAll(targetURL, "&", "^&")
		return exec.Command("cmd", "/c", "start", "", escaped).Start()
	case "darwin":
		return exec.Command("open", targetURL).Start()
	default:
		return exec.Command("xdg-open", targetURL).Start()
	}
}

func writeBrowserMessage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Escape: body may carry server-supplied error text — never reflect raw HTML.
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><title>%s</title>
<style>body{font-family:system-ui,sans-serif;max-width:600px;margin:80px auto;padding:0 20px;color:#222}h1{font-size:22px}p{font-size:16px;color:#555}</style>
</head><body><h1>%s</h1><p>%s</p></body></html>`, safeTitle, safeTitle, safeBody)
}
