//go:build !integration

package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewBridgeClient(t *testing.T) {
	t.Run("no proxy configured keeps the default transport", func(t *testing.T) {
		viper.Set(commonParams.ProxyKey, "")
		c := newBridgeClient()
		assert.Equal(t, bridgeRequestTimeout, c.Timeout)
		// nil transport → Go's default, which honors HTTPS_PROXY / HTTP_PROXY / NO_PROXY at request time.
		assert.Nil(t, c.Transport)
	})

	t.Run("configured proxy routes through the proxy-aware client", func(t *testing.T) {
		viper.Set(commonParams.ProxyKey, "http://proxy.corp:8080")
		defer viper.Set(commonParams.ProxyKey, "")
		c := newBridgeClient()
		assert.Equal(t, bridgeRequestTimeout, c.Timeout)
		tr, ok := c.Transport.(*http.Transport)
		assert.True(t, ok, "expected a proxy-aware *http.Transport")
		assert.NotNil(t, tr.Proxy, "expected a proxy resolver")
		req, err := http.NewRequest(http.MethodGet, "https://mcp.example.com", nil)
		assert.NoError(t, err)
		proxyURL, err := tr.Proxy(req)
		assert.NoError(t, err)
		assert.NotNil(t, proxyURL)
		assert.Equal(t, "http://proxy.corp:8080", proxyURL.String())
	})
}

func TestBuildSecurityMCPURL(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		want    string
		wantErr bool
	}{
		{
			name:   "regional iam host maps to ast",
			issuer: "https://eu.iam.checkmarx.net/auth/realms/cx_seg",
			want:   "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg",
		},
		{
			name:   "us no-prefix iam host maps to ast",
			issuer: "https://iam.checkmarx.net/auth/realms/myorg",
			want:   "https://ast.checkmarx.net/api/security-mcp/mcp/myorg",
		},
		{
			name:   "trailing slash tolerated",
			issuer: "https://deu.iam.checkmarx.net/auth/realms/tenant1/",
			want:   "https://deu.ast.checkmarx.net/api/security-mcp/mcp/tenant1",
		},
		{
			name:    "non-iam host cannot be mapped (use CX_MCP_URL)",
			issuer:  "https://example.com/auth/realms/x",
			wantErr: true,
		},
		{
			// Dev/on-prem hosts like iam-dev.dev.cxast.net do NOT follow the
			// <region>.iam.checkmarx.net pattern, so derivation must fail and the
			// caller is expected to set CX_MCP_URL (see TestDeriveMCPURL_CXMCPURLOverride).
			name:    "dev host is not auto-mappable",
			issuer:  "https://iam-dev.dev.cxast.net/auth/realms/dev_tenant",
			wantErr: true,
		},
		{
			name:    "missing realm segment",
			issuer:  "https://eu.iam.checkmarx.net",
			wantErr: true,
		},
		{
			name:    "empty issuer",
			issuer:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSecurityMCPURL(tt.issuer)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDeriveMCPURL_FlagOverride covers the --mcp-url flag (preferred for MCP
// clients that pass args but not env), which must win even over CX_MCP_URL.
func TestDeriveMCPURL_FlagOverride(t *testing.T) {
	t.Setenv("CX_MCP_URL", "https://from-env.example.com/api/security-mcp/mcp/x")
	got, err := deriveMCPURL("ignored-because-override-set",
		"https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant/")
	assert.NoError(t, err)
	assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
}

// TestDeriveMCPURL_CXMCPURLOverride covers the env escape hatch used for dev/on-prem
// environments (e.g. ast-master-components.dev.cxast.net / dev_tenant) whose host
// naming the iam->ast mapping cannot derive.
func TestDeriveMCPURL_CXMCPURLOverride(t *testing.T) {
	t.Setenv("CX_MCP_URL", "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant/")
	got, err := deriveMCPURL("ignored-because-override-set", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
}

func TestDeriveMCPURL_NoCredential(t *testing.T) {
	_ = os.Unsetenv("CX_MCP_URL")
	_, err := deriveMCPURL("", "")
	assert.Error(t, err)
}

// makeJWT builds an unsigned (alg=none) JWT carrying the given claims. ParseUnverified
// (used by wrappers.ExtractFromTokenClaims) does not check the signature, so this is
// sufficient to exercise claim extraction in tests.
func makeJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".sig"
}

// stubAccessToken overrides the getAccessToken seam for the duration of the test.
func stubAccessToken(t *testing.T, token string, err error) {
	t.Helper()
	prev := getAccessToken
	getAccessToken = func() (string, error) { return token, err }
	t.Cleanup(func() { getAccessToken = prev })
}

// TestDeriveMCPURL_AstBaseURLClaim: the authoritative path — the access token's
// ast-base-url claim supplies the AST base; the realm comes from the refresh token.
func TestDeriveMCPURL_AstBaseURLClaim(t *testing.T) {
	t.Setenv("CX_MCP_URL", "")
	refresh := makeJWT(map[string]interface{}{"iss": "https://eu.iam.checkmarx.net/auth/realms/cx_seg"})
	stubAccessToken(t, makeJWT(map[string]interface{}{"ast-base-url": "https://eu.ast.checkmarx.net"}), nil)

	got, err := deriveMCPURL(refresh, "")
	assert.NoError(t, err)
	assert.Equal(t, "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg", got)
}

// TestDeriveMCPURL_AstBaseURLClaimTrailingSlash: a base URL with a trailing slash
// must not produce a doubled separator.
func TestDeriveMCPURL_AstBaseURLClaimTrailingSlash(t *testing.T) {
	t.Setenv("CX_MCP_URL", "")
	refresh := makeJWT(map[string]interface{}{"iss": "https://eu.iam.checkmarx.net/auth/realms/cx_seg"})
	stubAccessToken(t, makeJWT(map[string]interface{}{"ast-base-url": "https://eu.ast.checkmarx.net/"}), nil)

	got, err := deriveMCPURL(refresh, "")
	assert.NoError(t, err)
	assert.Equal(t, "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg", got)
}

// TestDeriveMCPURL_DevHostResolvesViaAstBaseURL: a dev host whose iam->ast swap is
// NOT mappable (regression vs TestBuildSecurityMCPURL "dev host is not auto-mappable")
// now resolves automatically because the access token carries ast-base-url.
func TestDeriveMCPURL_DevHostResolvesViaAstBaseURL(t *testing.T) {
	t.Setenv("CX_MCP_URL", "")
	refresh := makeJWT(map[string]interface{}{"iss": "https://iam-dev.dev.cxast.net/auth/realms/dev_tenant"})
	stubAccessToken(t, makeJWT(map[string]interface{}{"ast-base-url": "https://ast-master-components.dev.cxast.net"}), nil)

	got, err := deriveMCPURL(refresh, "")
	assert.NoError(t, err)
	assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
}

// TestDeriveMCPURL_FallsBackToSwap: when the access-token exchange fails OR the claim
// is absent, a standard cloud host still resolves via the offline iam->ast swap.
func TestDeriveMCPURL_FallsBackToSwap(t *testing.T) {
	refresh := makeJWT(map[string]interface{}{"iss": "https://eu.iam.checkmarx.net/auth/realms/cx_seg"})
	want := "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg"

	t.Run("exchange error", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "")
		stubAccessToken(t, "", errors.New("token endpoint unreachable"))
		got, err := deriveMCPURL(refresh, "")
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("claim absent", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "")
		stubAccessToken(t, makeJWT(map[string]interface{}{"sub": "no-base-url-here"}), nil)
		got, err := deriveMCPURL(refresh, "")
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

// TestDeriveMCPURL_LadderPrecedence asserts flag > CX_MCP_URL > ast-base-url(access)
// > iam->ast swap by removing one tier at a time. The ast-base-url and swap tiers are
// disambiguated by using a dev host (swap fails) so a correct result proves the claim
// path was taken, not the swap.
func TestDeriveMCPURL_LadderPrecedence(t *testing.T) {
	devRefresh := makeJWT(map[string]interface{}{"iss": "https://iam-dev.dev.cxast.net/auth/realms/dev_tenant"})
	cloudRefresh := makeJWT(map[string]interface{}{"iss": "https://eu.iam.checkmarx.net/auth/realms/cx_seg"})
	claimToken := makeJWT(map[string]interface{}{"ast-base-url": "https://ast-master-components.dev.cxast.net"})

	t.Run("flag wins over env and claim", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "https://from-env.example.com/api/security-mcp/mcp/x")
		stubAccessToken(t, claimToken, nil)
		got, err := deriveMCPURL(devRefresh, "https://from-flag.example.com/api/security-mcp/mcp/x/")
		assert.NoError(t, err)
		assert.Equal(t, "https://from-flag.example.com/api/security-mcp/mcp/x", got)
	})

	t.Run("env wins over claim", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "https://from-env.example.com/api/security-mcp/mcp/x")
		stubAccessToken(t, claimToken, nil)
		got, err := deriveMCPURL(devRefresh, "")
		assert.NoError(t, err)
		assert.Equal(t, "https://from-env.example.com/api/security-mcp/mcp/x", got)
	})

	t.Run("claim wins over swap", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "")
		stubAccessToken(t, claimToken, nil)
		got, err := deriveMCPURL(devRefresh, "")
		assert.NoError(t, err)
		assert.Equal(t, "https://ast-master-components.dev.cxast.net/api/security-mcp/mcp/dev_tenant", got)
	})

	t.Run("swap is last resort", func(t *testing.T) {
		t.Setenv("CX_MCP_URL", "")
		stubAccessToken(t, "", errors.New("no exchange"))
		got, err := deriveMCPURL(cloudRefresh, "")
		assert.NoError(t, err)
		assert.Equal(t, "https://eu.ast.checkmarx.net/api/security-mcp/mcp/cx_seg", got)
	})
}

func TestRealmFromIssuer(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		want    string
		wantErr bool
	}{
		{name: "standard realm", issuer: "https://eu.iam.checkmarx.net/auth/realms/cx_seg", want: "cx_seg"},
		{name: "trailing slash", issuer: "https://eu.iam.checkmarx.net/auth/realms/tenant1/", want: "tenant1"},
		{name: "no path", issuer: "https://eu.iam.checkmarx.net", wantErr: true},
		{name: "empty issuer", issuer: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := realmFromIssuer(tt.issuer)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---- self-heal / resilience test support ----

// syncBuffer is a thread-safe buffer so the read loop, watcher goroutine, and the
// test can touch stdout output without a data race.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// setupBridgeTest isolates global state: clears the credential, stubs the disk/cache
// seams to no-ops, and speeds the watcher poll. All originals are restored via
// t.Cleanup.
func setupBridgeTest(t *testing.T) {
	t.Helper()
	prevKey := viper.GetString(commonParams.AstAPIKey)
	prevReload, prevInval, prevPoll := reloadConfig, invalidateTokenCache, credentialPollInterval
	t.Cleanup(func() {
		viper.Set(commonParams.AstAPIKey, prevKey)
		reloadConfig = prevReload
		invalidateTokenCache = prevInval
		credentialPollInterval = prevPoll
	})
	viper.Set(commonParams.AstAPIKey, "")
	reloadConfig = func() {}
	invalidateTokenCache = func() {}
	credentialPollInterval = time.Millisecond
	t.Setenv("CHECKMARX_API_KEY", "")
	t.Setenv("CX_MCP_URL", "")
}

// waitFor polls the output buffer until it contains substr or times out.
func waitFor(t *testing.T, buf *syncBuffer, substr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), substr) {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q in output:\n%s", substr, buf.String())
}

func decodeLines(t *testing.T, s string) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid JSON-RPC line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// TestRunBridge_UnauthAnswersInitializeLocally: with no credential, initialize is
// answered locally so the client sees the server Connected (listChanged=true).
func TestRunBridge_UnauthAnswersInitializeLocally(t *testing.T) {
	setupBridgeTest(t)
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n")
	var out syncBuffer
	err := runBridgeIO(in, &out, &http.Client{}, "1.2.3", "", func() string { return "" })
	assert.NoError(t, err)

	lines := decodeLines(t, out.String())
	assert.Len(t, lines, 1)
	result := lines[0]["result"].(map[string]interface{})
	assert.Equal(t, "2025-06-18", result["protocolVersion"])
	tools := result["capabilities"].(map[string]interface{})["tools"].(map[string]interface{})
	assert.Equal(t, true, tools["listChanged"])
	si := result["serverInfo"].(map[string]interface{})
	assert.Equal(t, "Checkmarx Security", si["name"])
	assert.Equal(t, "1.2.3", si["version"])
}

// TestRunBridge_UnauthToolsListEmpty: while unauth, tools/list returns an empty list
// (so the client shows Connected with no tools, ready for the later list_changed).
func TestRunBridge_UnauthToolsListEmpty(t *testing.T) {
	setupBridgeTest(t)
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	var out syncBuffer
	assert.NoError(t, runBridgeIO(in, &out, &http.Client{}, "1.0", "", func() string { return "" }))

	lines := decodeLines(t, out.String())
	assert.Len(t, lines, 2)
	toolsResult := lines[1]["result"].(map[string]interface{})
	assert.Empty(t, toolsResult["tools"])
}

// TestWatcher_SelfHeal_EndToEnd is the headline test: unauth -> Connected (empty) ->
// simulate `cx auth login` -> watcher heals -> notifications/tools/list_changed ->
// client re-fetches tools/list -> REAL tools, with NO manual reconnect.
func TestWatcher_SelfHeal_EndToEnd(t *testing.T) {
	setupBridgeTest(t)

	// Simulate the credential appearing via the injected resolveKey seam (mutex-guarded)
	// so the watcher's poll never races the test write — viper itself is not concurrent-safe.
	var credMu sync.Mutex
	cred := ""
	resolveKey := func() string { credMu.Lock(); defer credMu.Unlock(); return cred }
	setCred := func(v string) { credMu.Lock(); cred = v; credMu.Unlock() }

	var mu sync.Mutex
	var seenInit, seenInitialized, seenToolsList bool
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(raw, &req)
		mu.Lock()
		authHeader = r.Header.Get("Authorization")
		switch req.Method {
		case "initialize":
			seenInit = true
		case "notifications/initialized":
			seenInitialized = true
		case "tools/list":
			seenToolsList = true
		}
		mu.Unlock()
		switch req.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "sess-123")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{"protocolVersion":"2025-06-18","serverInfo":{"name":"security-mcp"}}}`))
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":3,"result":{"tools":[{"name":"codeRemediation"},{"name":"triggerScan"}]}}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
		}
	}))
	defer srv.Close()
	t.Setenv("CX_MCP_URL", srv.URL) // deriveMCPURL returns this; no token exchange needed

	pr, pw := io.Pipe()
	var out syncBuffer
	done := make(chan struct{})
	go func() {
		_ = runBridgeIO(pr, &out, &http.Client{}, "9.9.9", "", resolveKey)
		close(done)
	}()

	// 1. initialize -> local synthetic init (empty tools, listChanged)
	_, _ = io.WriteString(pw, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n")
	waitFor(t, &out, `"listChanged":true`)
	// 2. tools/list while unauth -> empty
	_, _ = io.WriteString(pw, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")
	waitFor(t, &out, `"tools":[]`)
	// 3. simulate `cx auth login` writing a credential another process would have written
	setCred(makeJWT(map[string]interface{}{"iss": "https://eu.iam.checkmarx.net/auth/realms/cx_seg"}))
	// 4. watcher heals and pushes list_changed — no reconnect
	waitFor(t, &out, "notifications/tools/list_changed")
	// 5. client re-fetches tools/list -> now proxied to the remote -> REAL tools
	_, _ = io.WriteString(pw, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`+"\n")
	waitFor(t, &out, "codeRemediation")
	_ = pw.Close()
	<-done

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, seenInit, "remote received the bridge-driven initialize")
	assert.True(t, seenInitialized, "remote received notifications/initialized")
	assert.True(t, seenToolsList, "remote received tools/list after heal")
	assert.NotContains(t, authHeader, "Bearer", "credential forwarded raw, no Bearer prefix")
	assert.NotEmpty(t, authHeader)
}

// TestWatcher_StaysDegraded_NoCredential: with no credential the watcher never
// promotes, writes nothing, and exits cleanly on stop.
func TestWatcher_StaysDegraded_NoCredential(t *testing.T) {
	setupBridgeTest(t)
	var out syncBuffer
	s := &bridgeSession{writer: newSyncWriter(&out), state: stateUnauth, resolveKey: func() string { return "" }}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		s.watchForCredential(&http.Client{}, "", stop)
		close(done)
	}()
	time.Sleep(25 * time.Millisecond) // several poll ticks
	assert.False(t, s.isConnected())
	assert.Empty(t, out.String())
	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit on stop")
	}
}

// TestSyncWriter_NoInterleave: concurrent emits never produce a partial/interleaved
// line — every output line is independently valid JSON.
func TestSyncWriter_NoInterleave(t *testing.T) {
	var out syncBuffer
	sw := newSyncWriter(&out)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			line, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": n, "method": "notifications/tools/list_changed"})
			sw.emitLine(line)
		}(i)
	}
	wg.Wait()
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		assert.True(t, json.Valid([]byte(line)), "line not valid JSON: %q", line)
	}
}

// TestDispatch_AuthedPathUnchanged covers the connected-state 401/403 single-retry
// credential reload — behavior preserved from before the resilience change. The raw
// credential (no Bearer) is forwarded and the access token is never sent.
func TestDispatch_AuthedPathUnchanged(t *testing.T) {
	const initialKey = "initial-key"
	const reloadedKey = "reloaded-key"
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	run := func(srvURL string) *syncBuffer {
		var out syncBuffer
		s := &bridgeSession{state: stateConnected, apiKey: initialKey, mcpURL: srvURL, writer: newSyncWriter(&out), resolveKey: productionResolveAPIKey}
		s.dispatch(&http.Client{}, body)
		return &out
	}

	t.Run("401 then success after reload forwards raw new credential", func(t *testing.T) {
		setupBridgeTest(t)
		viper.Set(commonParams.AstAPIKey, reloadedKey) // simulate the rotated token visible on disk
		var seenAuth []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenAuth = append(seenAuth, r.Header.Get("Authorization"))
			if len(seenAuth) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
		}))
		defer srv.Close()

		out := run(srv.URL)
		assert.Equal(t, []string{initialKey, reloadedKey}, seenAuth) // retried once, raw, no Bearer
		assert.Contains(t, out.String(), `"ok":true`)
	})

	t.Run("no retry when reloaded credential is unchanged", func(t *testing.T) {
		setupBridgeTest(t)
		viper.Set(commonParams.AstAPIKey, initialKey) // same as the one already in use
		var calls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		out := run(srv.URL)
		assert.Equal(t, 1, calls) // no retry — credential did not change
		assert.Contains(t, out.String(), `"error"`)
	})

	t.Run("exactly one retry then a JSON-RPC error", func(t *testing.T) {
		setupBridgeTest(t)
		viper.Set(commonParams.AstAPIKey, reloadedKey)
		var calls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		out := run(srv.URL)
		assert.Equal(t, 2, calls) // initial + exactly one retry, no loop
		assert.Contains(t, out.String(), `"error"`)
	})
}

// TestAuthedSelfHeal_ReReadsDisk proves the 401/403 path re-reads config from DISK
// (reloadConfig) BEFORE resolveKey — the new key only becomes visible after the
// disk re-read, so a token rotated by another process is actually picked up (this
// fails without reloadConfig because viper is a stale startup snapshot).
func TestAuthedSelfHeal_ReReadsDisk(t *testing.T) {
	setupBridgeTest(t)
	const oldKey = "old-key"
	const newKey = "new-key"
	viper.Set(commonParams.AstAPIKey, oldKey)                           // stale in-memory value
	reloadConfig = func() { viper.Set(commonParams.AstAPIKey, newKey) } // disk re-read brings the new token

	var seenAuth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		if len(seenAuth) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
	}))
	defer srv.Close()

	var out syncBuffer
	s := &bridgeSession{state: stateConnected, apiKey: oldKey, mcpURL: srv.URL, writer: newSyncWriter(&out), resolveKey: productionResolveAPIKey}
	s.dispatch(&http.Client{}, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))

	assert.Equal(t, []string{oldKey, newKey}, seenAuth) // retry used the disk-refreshed key
	assert.Contains(t, out.String(), `"ok":true`)
}

// TestEstablishRemoteSession_DoesNotEmitInitResult: the bridge-driven remote
// initialize captures the session id + proto and drives notifications/initialized,
// but must NOT emit an init result to the client (it already got the local one).
func TestEstablishRemoteSession_DoesNotEmitInitResult(t *testing.T) {
	setupBridgeTest(t)
	var seenInitialized bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(raw, &req)
		switch req.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "sid-9")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{"protocolVersion":"2025-06-18"}}`))
		case "notifications/initialized":
			seenInitialized = true
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer srv.Close()

	var out syncBuffer
	s := &bridgeSession{writer: newSyncWriter(&out), clientProto: "2025-06-18", version: "1.0"}
	ok := s.establishRemoteSession(&http.Client{}, srv.URL, "raw-key")

	assert.True(t, ok)
	assert.Equal(t, "sid-9", s.id)
	assert.True(t, s.remoteReady)
	assert.True(t, seenInitialized)
	assert.Empty(t, out.String(), "no init result should be emitted to the client")
}
