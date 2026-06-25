package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The bridge is a transparent stdio<->HTTP proxy to the remote Checkmarx Security
// MCP. It exists so the Claude Code plugin's .mcp.json can launch the remediation
// MCP with `command: "cx", args: ["mcp", "bridge"]` — using cx itself, the one
// binary guaranteed present, instead of bash/node/python (none of which are
// guaranteed across Windows/macOS/Linux or on a native, Bun-based Claude install).
//
// It reads the credential cx already resolved (env CX_APIKEY / cx config, loaded
// at startup), derives the realm-scoped URL from the credential's JWT `iss`
// claim, and forwards newline-delimited JSON-RPC between stdin/stdout and the
// remote MCP's Streamable HTTP endpoint (application/json + text/event-stream,
// Mcp-Session-Id). The credential is sent ONLY in the Authorization header (the
// server exchanges it; no client-side OAuth flow runs here) and is never written
// to stdout/stderr.
//
// Resilient/self-healing: if no usable credential exists at startup (e.g. the
// developer hasn't logged in yet, or the stored token is expired/malformed in a way
// that prevents URL derivation), the bridge does NOT exit. It answers the MCP
// handshake LOCALLY so the client shows the server "Connected" with an empty tool
// list (capabilities.tools.listChanged=true), and a background watcher polls cx
// config on disk; the moment `cx auth login` writes a usable credential, the bridge
// establishes the remote session and pushes notifications/tools/list_changed so the
// client auto-fetches the real tools — no /reload-plugins, no /mcp reconnect.
const (
	securityMCPPathPrefix = "/api/security-mcp/mcp/"
	bridgeRequestTimeout  = 120 * time.Second
	jsonrpcInternalError  = -32000 // JSON-RPC reserved server-error code
	httpAccepted          = 202    // POST of a notification/response: no body to relay
)

// bridgeState is the bridge's connection lifecycle: stateUnauth answers the MCP
// handshake locally and runs a credential watcher; stateConnected proxies to the
// remote (the only state the authed-at-startup path is ever in).
type bridgeState int

const (
	stateUnauth bridgeState = iota
	stateConnected
)

// supportedProtocolVersions are the MCP protocol versions the local handshake can
// advertise. The first is the preferred default when a client omits one.
var supportedProtocolVersions = []string{"2025-06-18", "2024-11-05"}

func defaultProtocolVersion() string { return supportedProtocolVersions[0] }

// bridgeSession holds the MCP session state. Fields touched by both the read loop
// and the watcher goroutine (state/apiKey/mcpURL/clientProto/remoteReady) are
// guarded by mu; id/proto are mutated only after promotion (single-threaded).
type bridgeSession struct {
	mu          sync.Mutex
	state       bridgeState
	apiKey      string // raw credential forwarded to the remote (Authorization)
	mcpURL      string // realm-scoped Security MCP URL
	clientProto string // protocolVersion the client requested at initialize
	remoteReady bool   // the remote initialize handshake has completed
	version     string // cx binary version, for the synthetic serverInfo
	writer      *syncWriter

	id    string // Mcp-Session-Id, echoed back on every subsequent request
	proto string // negotiated protocolVersion, sent as MCP-Protocol-Version
}

// syncWriter serializes all stdout writes so a full JSON-RPC line is written+flushed
// atomically — the read loop and the watcher goroutine share it, and the MCP stdio
// transport forbids interleaved/partial lines.
type syncWriter struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func newSyncWriter(w io.Writer) *syncWriter { return &syncWriter{w: bufio.NewWriter(w)} }

func (sw *syncWriter) emitLine(raw []byte) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	_, _ = sw.w.Write(raw)
	_ = sw.w.WriteByte('\n')
	_ = sw.w.Flush()
}

const mcpURLFlag = "mcp-url"

// Test seams. getAccessToken stubs the (network) refresh_token->access_token
// exchange. reloadConfig re-reads cx config FROM DISK into viper — essential for
// self-heal, since viper is a one-shot startup snapshot and would otherwise never
// see a credential written by a later `cx auth login`. invalidateTokenCache forces
// a fresh access-token exchange. credentialPollInterval paces the watcher.
var (
	getAccessToken = wrappers.GetAccessToken

	reloadConfig = func() {
		_ = configuration.LoadConfiguration()
		wrappers.LoadActiveCredential()
	}
	invalidateTokenCache   = wrappers.InvalidateAccessTokenCache
	credentialPollInterval = 3 * time.Second
)

// NewBridgeCommand creates the hidden "cx mcp bridge" subcommand. version is the cx
// binary version, surfaced in the synthetic serverInfo during the degraded handshake.
func NewBridgeCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Proxy stdio MCP traffic to the Checkmarx Security remediation MCP",
		Long: `Run a stdio<->HTTP bridge to the remote Checkmarx Security MCP.

Intended to be launched by an AI coding assistant as an MCP server:

  {
    "mcpServers": {
      "Checkmarx": { "command": "cx", "args": ["mcp", "bridge"] }
    }
  }

The credential is read from cx config (or CX_APIKEY). The realm-scoped URL is
resolved by, in order: the --mcp-url flag, the CX_MCP_URL env var, the
authoritative "ast-base-url" claim from the exchanged access token (works for
any region/on-prem), then an offline IAM->AST host swap. Override with --mcp-url
(preferred for MCP clients that pass args but not env) or CX_MCP_URL only for
air-gapped setups where the token endpoint is unreachable at startup.

If no usable credential exists at startup the bridge stays up in a degraded state
and connects automatically once you run 'cx auth login' — no restart needed.`,
		Hidden: true, // internal plumbing for the Claude Code / IDE plugins
		RunE: func(cmd *cobra.Command, _ []string) error {
			urlOverride, _ := cmd.Flags().GetString(mcpURLFlag)
			return runBridge(version, urlOverride)
		},
	}
	cmd.Flags().String(mcpURLFlag, "", "Override the Security MCP URL (dev/on-prem where it can't be derived from the credential)")
	return cmd
}

func runBridge(version, urlOverride string) error {
	return runBridgeIO(os.Stdin, os.Stdout, &http.Client{Timeout: bridgeRequestTimeout}, version, urlOverride)
}

// runBridgeIO is the testable core: it wires the session to the given streams,
// decides the startup state, runs the watcher when degraded, and pumps the stdin
// read loop. It never exits the process on a missing credential.
func runBridgeIO(in io.Reader, out io.Writer, client *http.Client, version, urlOverride string) error {
	sess := &bridgeSession{writer: newSyncWriter(out), version: version}

	apiKey := resolveAPIKey()
	mcpURL, err := deriveMCPURL(apiKey, urlOverride)
	if apiKey != "" && err == nil {
		sess.state = stateConnected
		sess.apiKey = apiKey
		sess.mcpURL = mcpURL
	} else {
		sess.state = stateUnauth
		// Degraded notice goes to STDERR only (stdout is the protocol channel).
		fmt.Fprintln(os.Stderr, "cx mcp bridge: no usable Checkmarx credential yet — serving in a degraded state. "+
			"Log in with 'cx auth login' (or /cx-cli-setup) using the default (yaml) or '--session global' mode "+
			"(NOT '--session local', which this process can't see); Checkmarx tools appear automatically once authenticated. "+
			"For on-prem/custom domains, set CX_MCP_URL or pass --mcp-url.")
		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); sess.watchForCredential(client, urlOverride, stop) }()
		// Clean shutdown on EOF: stop the watcher and wait for it to fully exit before
		// returning, so no goroutine outlives the bridge or touches config concurrently.
		defer func() { close(stop); wg.Wait() }()
	}

	reader := bufio.NewReader(in)
	for {
		line, readErr := reader.ReadString('\n')
		if body := bytes.TrimSpace([]byte(line)); len(body) > 0 {
			sess.dispatch(client, body)
		}
		if readErr != nil {
			break // EOF — the client closed the connection
		}
	}
	return nil
}

// resolveAPIKey returns the credential cx resolved (CX_APIKEY env / cx config /
// active session), falling back to CHECKMARX_API_KEY for parity with the previous
// Python bridge. Callers that need a credential written AFTER startup must call
// reloadConfig() first (viper is a one-shot startup snapshot). It is a package var
// so the concurrent self-heal test can simulate a credential appearing without
// racing viper (which is not concurrency-safe).
var resolveAPIKey = func() string {
	if k := strings.TrimSpace(viper.GetString(commonParams.AstAPIKey)); k != "" {
		return k
	}
	if k := strings.TrimSpace(os.Getenv("CHECKMARX_API_KEY")); k != "" {
		return k
	}
	return ""
}

// deriveMCPURL builds the realm-scoped Security MCP URL, region/tenant/on-prem
// agnostic. Resolution order (top wins):
//  1. the --mcp-url flag (explicit override),
//  2. the CX_MCP_URL env var (explicit override),
//  3. the authoritative "ast-base-url" claim from the exchanged ACCESS token —
//     the AST base the IAM server itself issued (works for any region/on-prem),
//  4. an offline IAM->AST host swap from the credential's `iss` claim, for
//     standard cloud regions when the token exchange is unavailable.
//
// The realm (tenant) always comes from the stored credential's `iss` claim and is
// independent of how the AST base host is resolved.
func deriveMCPURL(apiKey, override string) (string, error) {
	override = strings.TrimSpace(override)
	if override == "" {
		override = strings.TrimSpace(os.Getenv("CX_MCP_URL"))
	}
	if override != "" {
		return strings.TrimRight(override, "/"), nil
	}
	if apiKey == "" {
		return "", errors.New("no API key")
	}

	issuer, err := wrappers.ExtractFromTokenClaims(apiKey, "iss")
	if err != nil {
		return "", err
	}
	realm, err := realmFromIssuer(issuer)
	if err != nil {
		return "", err
	}

	// 3. Authoritative: AST base from the access token's ast-base-url claim. On any
	// failure (token endpoint unreachable, claim absent) fall through to the swap.
	if base := astBaseFromAccessToken(); base != "" {
		return joinSecurityMCPURL(base, realm), nil
	}

	// 4. Offline fallback for standard cloud regions.
	astBase, err := astBaseFromIAMHost(issuer)
	if err != nil {
		return "", err
	}
	return joinSecurityMCPURL(astBase, realm), nil
}

// astBaseFromAccessToken exchanges the stored refresh token for an access token
// (cached, non-interactive grant_type=refresh_token) and reads the authoritative
// "ast-base-url" claim from it — the claim is present only on the access token,
// never on the stored refresh token. Returns "" (logging at verbose) on any
// failure so the caller can fall back to the offline host swap. The access token
// is used ONLY for URL discovery here; it is never sent to the MCP server.
func astBaseFromAccessToken() string {
	accessToken, err := getAccessToken()
	if err != nil {
		logger.PrintIfVerbose("cx mcp bridge: access-token exchange failed, falling back to IAM->AST host swap: " + err.Error())
		return ""
	}
	base, err := wrappers.ExtractFromTokenClaims(accessToken, wrappers.BaseURLKey)
	if err != nil {
		logger.PrintIfVerbose("cx mcp bridge: ast-base-url claim unavailable, falling back to IAM->AST host swap: " + err.Error())
		return ""
	}
	return strings.TrimSpace(base)
}

// buildSecurityMCPURL maps a JWT issuer (e.g. https://eu.iam.checkmarx.net/auth/realms/<realm>)
// to the realm-scoped Security MCP URL via the offline IAM->AST host swap
// (https://eu.ast.checkmarx.net/api/security-mcp/mcp/<realm>).
func buildSecurityMCPURL(issuer string) (string, error) {
	astBase, err := astBaseFromIAMHost(issuer)
	if err != nil {
		return "", err
	}
	realm, err := realmFromIssuer(issuer)
	if err != nil {
		return "", err
	}
	return joinSecurityMCPURL(astBase, realm), nil
}

// astBaseFromIAMHost maps a JWT issuer's IAM host to the AST base URL for standard
// cloud regions (<region>.iam -> <region>.ast, or iam. -> ast.). Dev/on-prem/custom
// hosts are not mappable and return an error (use ast-base-url or CX_MCP_URL).
func astBaseFromIAMHost(issuer string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(issuer), "/"))
	if err != nil || parsed.Host == "" {
		return "", errors.New("could not parse issuer host from API key")
	}
	switch host := parsed.Host; {
	case strings.Contains(host, ".iam."):
		return "https://" + strings.Replace(host, ".iam.", ".ast.", 1), nil
	case strings.HasPrefix(host, "iam."):
		return "https://ast." + strings.TrimPrefix(host, "iam."), nil
	default:
		return "", errors.New("could not map IAM host to AST host; set CX_MCP_URL for on-prem/custom domains")
	}
}

// realmFromIssuer extracts the realm (tenant) from a JWT issuer URL whose path ends
// in .../auth/realms/<realm>.
func realmFromIssuer(issuer string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(issuer), "/"))
	if err != nil {
		return "", errors.New("could not parse issuer from API key")
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	realm := segments[len(segments)-1]
	if realm == "" {
		return "", errors.New("could not derive realm from issuer")
	}
	return realm, nil
}

// joinSecurityMCPURL composes the realm-scoped Security MCP endpoint from an AST
// base URL and a realm.
func joinSecurityMCPURL(base, realm string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + securityMCPPathPrefix + realm
}

// dispatch handles one inbound JSON-RPC line. When unauthenticated it answers the
// MCP handshake locally; when connected it proxies to the remote (unchanged).
func (s *bridgeSession) dispatch(client *http.Client, body []byte) {
	s.mu.Lock()
	state := s.state
	apiKey := s.apiKey
	mcpURL := s.mcpURL
	s.mu.Unlock()

	if state == stateUnauth {
		s.dispatchLocal(body)
		return
	}
	s.proxy(client, mcpURL, apiKey, body)
}

// dispatchLocal answers the MCP handshake without any network access so the client
// shows the server Connected while we wait for a credential. initialize advertises
// tools.listChanged=true (the contract that lets the client accept an empty list now
// and re-fetch after notifications/tools/list_changed); tools/list returns empty;
// ping is answered; any other request returns a clear "not authenticated" error.
func (s *bridgeSession) dispatchLocal(body []byte) {
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}
	switch req.Method {
	case "initialize":
		s.emitLocalInitialize(req.ID, req.Params.ProtocolVersion)
	case "notifications/initialized":
		// client readiness signal — no response
	case "tools/list":
		s.emitResult(req.ID, map[string]interface{}{"tools": []interface{}{}})
	case "ping":
		s.emitResult(req.ID, map[string]interface{}{})
	default:
		if len(req.ID) > 0 {
			s.writeError(body, "not authenticated yet — run 'cx auth login' (or /cx-cli-setup); Checkmarx tools enable automatically once authenticated")
		}
	}
}

// emitLocalInitialize answers initialize locally, echoing the client's requested
// protocolVersion (or a supported default) and remembering it for the eventual
// remote handshake.
func (s *bridgeSession) emitLocalInitialize(id json.RawMessage, clientProto string) {
	proto := strings.TrimSpace(clientProto)
	if proto == "" {
		proto = defaultProtocolVersion()
	}
	s.mu.Lock()
	s.clientProto = proto
	s.mu.Unlock()
	s.emitResult(id, map[string]interface{}{
		"protocolVersion": proto,
		"capabilities":    map[string]interface{}{"tools": map[string]interface{}{"listChanged": true}},
		"serverInfo":      map[string]interface{}{"name": "Checkmarx Security", "version": s.version},
		"instructions":    "Checkmarx Security is initializing — its tools appear once you authenticate (run 'cx auth login' or /cx-cli-setup).",
	})
}

// emitResult writes a JSON-RPC result for the given id.
func (s *bridgeSession) emitResult(id json.RawMessage, result interface{}) {
	reply, err := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": result})
	if err != nil {
		return
	}
	s.writer.emitLine(reply)
}

// proxy forwards one JSON-RPC request to the remote and relays the response. On
// 401/403 it re-reads cx config from disk and retries once with the refreshed
// credential — a rotated `cx auth login` token self-heals without a restart, while a
// dead key fails fast instead of looping.
func (s *bridgeSession) proxy(client *http.Client, mcpURL, apiKey string, body []byte) {
	resp, err := s.post(client, mcpURL, apiKey, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx mcp bridge: request failed: %v\n", err)
		s.writeError(body, "")
		return
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		reloadConfig() // re-read disk so a token rotated by another process is visible
		reloaded := resolveAPIKey()
		if reloaded != "" && reloaded != apiKey {
			s.mu.Lock()
			s.apiKey = reloaded
			s.mu.Unlock()
			retry, retryErr := s.post(client, mcpURL, reloaded, body)
			if retryErr != nil {
				fmt.Fprintf(os.Stderr, "cx mcp bridge: retry after credential reload failed: %v\n", retryErr)
				s.writeError(body, "")
				return
			}
			s.finish(retry, body)
			return
		}
		fmt.Fprintf(os.Stderr, "cx mcp bridge: HTTP %d from MCP endpoint (no fresh credential to retry with)\n", resp.StatusCode)
		s.writeError(body, fmt.Sprintf("HTTP %d (auth failed — run /cx-cli-setup to re-authenticate)", resp.StatusCode))
		return
	}

	s.finish(resp, body)
}

func (s *bridgeSession) post(client *http.Client, mcpURL, apiKey string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, mcpURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Accept-Encoding", "identity")
	// Forward the RAW stored credential (API key / refresh token); the server
	// exchanges it server-side. Never send the access token fetched in
	// deriveMCPURL — that is for URL discovery only.
	req.Header.Set("Authorization", apiKey)
	if s.id != "" {
		req.Header.Set("Mcp-Session-Id", s.id)
	}
	if s.proto != "" {
		req.Header.Set("MCP-Protocol-Version", s.proto)
	}
	return client.Do(req)
}

func (s *bridgeSession) finish(resp *http.Response, body []byte) {
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "cx mcp bridge: HTTP %d from MCP endpoint\n", resp.StatusCode)
		s.writeError(body, fmt.Sprintf("HTTP %d", resp.StatusCode))
		resp.Body.Close()
		return
	}
	s.handleResponse(resp)
}

func (s *bridgeSession) handleResponse(resp *http.Response) {
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		s.id = sid
	}
	if resp.StatusCode == httpAccepted {
		_, _ = io.Copy(io.Discard, resp.Body)
		return
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		s.pumpSSE(resp.Body)
		return
	}
	raw, _ := io.ReadAll(resp.Body)
	s.emit(raw)
}

// pumpSSE parses an SSE stream and emits each buffered `data:` payload as a single
// JSON-RPC line until the stream ends.
func (s *bridgeSession) pumpSSE(body io.Reader) {
	reader := bufio.NewReader(body)
	var dataLines []string
	flush := func() {
		if len(dataLines) > 0 {
			s.emit([]byte(strings.Join(dataLines, "\n")))
			dataLines = dataLines[:0]
		}
	}
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			switch trimmed := strings.TrimRight(line, "\r\n"); {
			case trimmed == "": // blank line dispatches the buffered event
				flush()
			case strings.HasPrefix(trimmed, ":"): // SSE comment / keep-alive
			case strings.HasPrefix(trimmed, "data:"):
				dataLines = append(dataLines, strings.TrimLeft(strings.TrimPrefix(trimmed, "data:"), " \t"))
			default: // event:/id:/retry: are not needed for JSON-RPC transport
			}
		}
		if err != nil {
			break
		}
	}
	flush() // flush a trailing event with no terminating blank line
}

// emit writes one validated JSON-RPC message to stdout as a single line, capturing
// the negotiated protocol version from an initialize result. stdout is the MCP
// channel, so non-JSON payloads are dropped rather than corrupting the stream.
func (s *bridgeSession) emit(raw []byte) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || !json.Valid(raw) {
		return
	}
	var probe struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && probe.Result.ProtocolVersion != "" {
		s.proto = probe.Result.ProtocolVersion
	}
	s.writer.emitLine(raw)
}

// writeError surfaces a JSON-RPC error for a request id so the client never hangs on
// a failed call. Notifications and unparseable/batch lines have no single id to
// reply to and are skipped.
func (s *bridgeSession) writeError(requestLine []byte, detail string) {
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.Unmarshal(requestLine, &req); err != nil || len(req.ID) == 0 || req.Method == "" {
		return
	}
	message := "Checkmarx MCP request failed"
	if detail != "" {
		message = "Checkmarx MCP " + detail
	}
	reply, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"error":   map[string]interface{}{"code": jsonrpcInternalError, "message": message},
	})
	if err != nil {
		return
	}
	s.writer.emitLine(reply)
}

// watchForCredential polls cx config on disk while the bridge is degraded. As soon
// as a usable credential appears (e.g. the developer ran `cx auth login`), it
// establishes the remote session, flips to connected, and pushes a
// notifications/tools/list_changed so the client auto-fetches the real tools — then
// the watcher exits. It also exits on EOF (stop closed).
func (s *bridgeSession) watchForCredential(client *http.Client, urlOverride string, stop <-chan struct{}) {
	ticker := time.NewTicker(credentialPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if s.tryHeal(client, urlOverride) {
				return
			}
		}
	}
}

// tryHeal attempts one self-heal cycle. Returns true once the bridge is connected.
func (s *bridgeSession) tryHeal(client *http.Client, urlOverride string) bool {
	if s.isConnected() {
		return true
	}
	reloadConfig() // the definitive disk re-read; viper alone is a stale startup snapshot
	key := resolveAPIKey()
	if key == "" {
		return false // still no credential — stay degraded
	}
	invalidateTokenCache() // a fresh login may target a different tenant
	mcpURL, err := deriveMCPURL(key, urlOverride)
	if err != nil || mcpURL == "" {
		return false
	}
	if !s.establishRemoteSession(client, mcpURL, key) {
		return false // remote not reachable / credential not yet valid — retry next tick
	}
	s.promote(key, mcpURL)
	s.notifyToolsChanged()
	return true
}

// establishRemoteSession performs the remote MCP initialize handshake on the
// bridge's behalf (the client's initialize was answered locally, so the remote never
// saw it). It sends a MINIMAL synthesized initialize (clientInfo=cx-bridge, no
// client capabilities), captures the Mcp-Session-Id + negotiated protocolVersion,
// drives notifications/initialized, and DISCARDS the response body (the client
// already received its handshake result). Returns false if the remote is unreachable
// or rejects the credential, so the watcher retries.
func (s *bridgeSession) establishRemoteSession(client *http.Client, mcpURL, apiKey string) bool {
	s.mu.Lock()
	proto := s.clientProto
	s.mu.Unlock()
	if proto == "" {
		proto = defaultProtocolVersion()
	}

	initReq, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      0,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": proto,
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]interface{}{"name": "cx-bridge", "version": s.version},
		},
	})
	if err != nil {
		return false
	}

	resp, err := s.post(client, mcpURL, apiKey, initReq)
	if err != nil {
		return false
	}
	sid := resp.Header.Get("Mcp-Session-Id")
	code := resp.StatusCode
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if code >= 400 {
		return false
	}

	// Session id + the version we negotiated drive every subsequent proxied request.
	s.id = sid
	s.proto = proto

	// Complete the remote lifecycle so it accepts tools/list and tool calls.
	notif, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if nresp, nerr := s.post(client, mcpURL, apiKey, notif); nerr == nil {
		_, _ = io.Copy(io.Discard, nresp.Body)
		nresp.Body.Close()
	}
	s.remoteReady = true
	return true
}

// promote publishes the resolved credential/URL and flips the bridge to connected.
func (s *bridgeSession) promote(key, mcpURL string) {
	s.mu.Lock()
	s.apiKey = key
	s.mcpURL = mcpURL
	s.state = stateConnected
	s.mu.Unlock()
}

func (s *bridgeSession) isConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == stateConnected
}

// notifyToolsChanged pushes an unsolicited notifications/tools/list_changed so the
// client re-fetches tools/list (now served by the connected remote) with no reload.
func (s *bridgeSession) notifyToolsChanged() {
	notif, err := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
	if err != nil {
		return
	}
	s.writer.emitLine(notif)
}
