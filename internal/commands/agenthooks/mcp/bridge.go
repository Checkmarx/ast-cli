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
	"time"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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
const (
	securityMCPPathPrefix = "/api/security-mcp/mcp/"
	bridgeRequestTimeout  = 120 * time.Second
	jsonrpcInternalError  = -32000 // JSON-RPC reserved server-error code
	httpAccepted          = 202    // POST of a notification/response: no body to relay
)

// bridgeSession holds the MCP session state negotiated as messages flow through.
type bridgeSession struct {
	id    string // Mcp-Session-Id, echoed back on every subsequent request
	proto string // negotiated protocolVersion, sent as MCP-Protocol-Version
}

const mcpURLFlag = "mcp-url"

// NewBridgeCommand creates the hidden "cx mcp bridge" subcommand.
func NewBridgeCommand() *cobra.Command {
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

The credential is read from cx config (or CX_APIKEY); the realm-scoped URL is
derived from the credential's JWT issuer claim. For dev/on-prem where it can't
be derived, override it with the --mcp-url flag or the CX_MCP_URL env var. The
flag is preferred for MCP clients that pass args but not env to the server.`,
		Hidden: true, // internal plumbing for the Claude Code / IDE plugins
		RunE: func(cmd *cobra.Command, _ []string) error {
			urlOverride, _ := cmd.Flags().GetString(mcpURLFlag)
			return runBridge(urlOverride)
		},
	}
	cmd.Flags().String(mcpURLFlag, "", "Override the Security MCP URL (dev/on-prem where it can't be derived from the credential)")
	return cmd
}

func runBridge(urlOverride string) error {
	apiKey := resolveAPIKey()
	mcpURL, err := deriveMCPURL(apiKey, urlOverride)
	if apiKey == "" || err != nil {
		// Fatal startup failure. Write to stderr and exit directly: returning an
		// error would route through main.exitIfError, which prints to STDOUT and
		// would corrupt the MCP protocol channel.
		fmt.Fprintln(os.Stderr, "cx mcp bridge: no usable Checkmarx credential or could not derive the Security MCP URL. "+
			"Run 'cx auth login' (or /cx-cli-setup), or set CX_MCP_URL for on-prem/custom domains.")
		os.Exit(1)
	}

	client := &http.Client{Timeout: bridgeRequestTimeout}
	sess := &bridgeSession{}
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)

	for {
		line, readErr := in.ReadString('\n')
		if body := bytes.TrimSpace([]byte(line)); len(body) > 0 {
			apiKey, mcpURL = sess.dispatch(client, mcpURL, apiKey, body, out)
		}
		if readErr != nil {
			break // EOF — the client closed the connection
		}
	}
	return nil
}

// resolveAPIKey returns the credential cx resolved at startup (CX_APIKEY env /
// cx config / active session), falling back to CHECKMARX_API_KEY for parity with
// the previous Python bridge.
func resolveAPIKey() string {
	if k := strings.TrimSpace(viper.GetString(commonParams.AstAPIKey)); k != "" {
		return k
	}
	if k := strings.TrimSpace(os.Getenv("CHECKMARX_API_KEY")); k != "" {
		return k
	}
	return ""
}

// deriveMCPURL builds the realm-scoped Security MCP URL. An explicit override
// wins (the --mcp-url flag, else the CX_MCP_URL env var); otherwise the realm and
// host come from the credential's JWT `iss` claim, mapping the IAM host to the AST
// host (<region>.iam -> <region>.ast, or iam. -> ast.).
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
	return buildSecurityMCPURL(issuer)
}

// buildSecurityMCPURL maps a JWT issuer (e.g. https://eu.iam.checkmarx.net/auth/realms/<realm>)
// to the realm-scoped Security MCP URL (https://eu.ast.checkmarx.net/api/security-mcp/mcp/<realm>).
func buildSecurityMCPURL(issuer string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(issuer), "/"))
	if err != nil || parsed.Host == "" {
		return "", errors.New("could not parse issuer host from API key")
	}

	var astBase string
	switch host := parsed.Host; {
	case strings.Contains(host, ".iam."):
		astBase = "https://" + strings.Replace(host, ".iam.", ".ast.", 1)
	case strings.HasPrefix(host, "iam."):
		astBase = "https://ast." + strings.TrimPrefix(host, "iam.")
	default:
		return "", errors.New("could not map IAM host to AST host; set CX_MCP_URL for on-prem/custom domains")
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	realm := segments[len(segments)-1]
	if realm == "" {
		return "", errors.New("could not derive realm from issuer")
	}
	return astBase + securityMCPPathPrefix + realm, nil
}

// dispatch forwards one JSON-RPC request and relays the response. On 401/403 it
// re-reads the credential and retries once, but only if the credential actually
// changed — a rotated `cx auth login` token self-heals without a restart, while a
// dead key fails fast instead of looping. Returns the (possibly refreshed)
// credential and URL for the next iteration.
func (s *bridgeSession) dispatch(client *http.Client, mcpURL, apiKey string, body []byte, out *bufio.Writer) (newKey, newURL string) {
	resp, err := s.post(client, mcpURL, apiKey, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cx mcp bridge: request failed: %v\n", err)
		writeJSONRPCError(out, body, "")
		return apiKey, mcpURL
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		reloaded := resolveAPIKey()
		if reloaded != "" && reloaded != apiKey {
			// The URL is stable across a credential reload (it comes from the
			// override or the realm, not the token instance), so only the
			// credential is refreshed here.
			apiKey = reloaded
			retry, retryErr := s.post(client, mcpURL, apiKey, body)
			if retryErr != nil {
				fmt.Fprintf(os.Stderr, "cx mcp bridge: retry after credential reload failed: %v\n", retryErr)
				writeJSONRPCError(out, body, "")
				return apiKey, mcpURL
			}
			s.finish(retry, body, out)
			return apiKey, mcpURL
		}
		fmt.Fprintf(os.Stderr, "cx mcp bridge: HTTP %d from MCP endpoint (no fresh credential to retry with)\n", resp.StatusCode)
		writeJSONRPCError(out, body, fmt.Sprintf("HTTP %d (auth failed — run /cx-cli-setup to re-authenticate)", resp.StatusCode))
		return apiKey, mcpURL
	}

	s.finish(resp, body, out)
	return apiKey, mcpURL
}

func (s *bridgeSession) post(client *http.Client, mcpURL, apiKey string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, mcpURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Authorization", apiKey)
	if s.id != "" {
		req.Header.Set("Mcp-Session-Id", s.id)
	}
	if s.proto != "" {
		req.Header.Set("MCP-Protocol-Version", s.proto)
	}
	return client.Do(req)
}

func (s *bridgeSession) finish(resp *http.Response, body []byte, out *bufio.Writer) {
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "cx mcp bridge: HTTP %d from MCP endpoint\n", resp.StatusCode)
		writeJSONRPCError(out, body, fmt.Sprintf("HTTP %d", resp.StatusCode))
		resp.Body.Close()
		return
	}
	s.handleResponse(resp, out)
}

func (s *bridgeSession) handleResponse(resp *http.Response, out *bufio.Writer) {
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		s.id = sid
	}
	if resp.StatusCode == httpAccepted {
		_, _ = io.Copy(io.Discard, resp.Body)
		return
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		s.pumpSSE(resp.Body, out)
		return
	}
	raw, _ := io.ReadAll(resp.Body)
	s.emit(out, raw)
}

// pumpSSE parses an SSE stream and emits each buffered `data:` payload as a single
// JSON-RPC line until the stream ends.
func (s *bridgeSession) pumpSSE(body io.Reader, out *bufio.Writer) {
	reader := bufio.NewReader(body)
	var dataLines []string
	flush := func() {
		if len(dataLines) > 0 {
			s.emit(out, []byte(strings.Join(dataLines, "\n")))
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
func (s *bridgeSession) emit(out *bufio.Writer, raw []byte) {
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
	_, _ = out.Write(raw)
	_ = out.WriteByte('\n')
	_ = out.Flush()
}

// writeJSONRPCError surfaces a JSON-RPC error for a request id so the client never
// hangs on a failed call. Notifications and unparseable/batch lines have no single
// id to reply to and are skipped.
func writeJSONRPCError(out *bufio.Writer, requestLine []byte, detail string) {
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
	_, _ = out.Write(reply)
	_ = out.WriteByte('\n')
	_ = out.Flush()
}
