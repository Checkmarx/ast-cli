package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	sessionStartTimeout = 500 * time.Millisecond
	hookCallTimeout     = 200 * time.Millisecond
)

// SyncOnSessionStart performs a synchronous policy version check and full pack download
// when the version has changed. Called at session start where a sub-second delay is acceptable.
func SyncOnSessionStart(serverURL, token string) {
	if serverURL == "" {
		return
	}
	remote := fetchPackVersion(serverURL, token, sessionStartTimeout)
	if remote < 0 || remote <= readLocalPackVersion() {
		return // unreachable or no change
	}
	if err := fetchAndWritePack(serverURL, token); err != nil {
		log.Printf("governance: sync on session start failed: %v", err)
	}
}

// CheckAndSyncVersion performs a lightweight version check after a hook verdict is returned.
// Called in a goroutine — it exits quickly if the version has not changed.
// The subprocess may exit before this goroutine completes; that is acceptable.
// The next session start will sync reliably.
func CheckAndSyncVersion(serverURL, token string) {
	if serverURL == "" {
		return
	}
	remote := fetchPackVersion(serverURL, token, hookCallTimeout)
	if remote < 0 || remote <= readLocalPackVersion() {
		return
	}
	if err := fetchAndWritePack(serverURL, token); err != nil {
		log.Printf("governance: async version sync failed: %v", err)
	}
}

// SyncOnce performs a forced pack download regardless of the current version.
// Used by the `cx policy sync` manual command.
func SyncOnce(serverURL, token string) error {
	if serverURL == "" {
		return fmt.Errorf("governance: no server URL configured")
	}
	return fetchAndWritePack(serverURL, token)
}

// fetchPackVersion sends a HEAD request to /api/v1/agent/policies/version and returns
// the X-Pack-Version header value. Returns -1 on any error or timeout so callers always fail-open.
func fetchPackVersion(serverURL, token string, timeout time.Duration) int {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodHead, serverURL+"/api/v1/agent/policies/version", nil)
	if err != nil {
		return -1
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return -1
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return -1
	}
	v, err := strconv.Atoi(resp.Header.Get("X-Pack-Version"))
	if err != nil {
		return -1
	}
	return v
}

// fetchAndWritePack downloads the full governance pack and atomically replaces the local file.
func fetchAndWritePack(serverURL, token string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, serverURL+"/api/v1/agent/policies", nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("governance: server returned %d fetching pack", resp.StatusCode)
	}
	return atomicWritePack(resp.Body)
}

// atomicWritePack writes the pack body to a .tmp file then renames it to the final path,
// ensuring the live policy file is never partially written.
func atomicWritePack(body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	// Validate the JSON before writing — refuse to replace a good pack with garbage.
	var pack GovernancePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return fmt.Errorf("governance: received invalid pack JSON: %w", err)
	}

	packPath := localPackPath()
	ensureDir(govBaseDir())
	tmp := packPath + ".tmp"
	if err := os.WriteFile(tmp, bytes.TrimSpace(data), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, packPath)
}

// RegisterAgentWithBackend POSTs to /api/v1/agents/register to upsert the
// developer_agents bridge row on the backend. Called once when a new agentId is
// first computed (architecture §4.4). Failures are silent — the hook still returns
// the correct verdict regardless of whether registration succeeds.
func RegisterAgentWithBackend(agentType string, cfg GovernanceConfig) {
	if cfg.ServerURL == "" || cfg.Token == "" {
		return
	}
	payload := map[string]any{
		"agentId":   AgentID(agentType),
		"agentType": agentType,
		"machineId": MachineID(),
		"userId":    CurrentUserName(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+"/api/v1/agents/register", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		log.Printf("governance: agent register returned %d", resp.StatusCode)
	}
}

// EnsureResolved attempts to resolve the OS username to a backend developer UUID
// and caches it in the agent identity file. Called asynchronously — failures are silent.
func EnsureResolved(cfg GovernanceConfig) {
	if cfg.ServerURL == "" || cfg.Token == "" {
		return
	}
	if CachedDeveloperID(cfg.AgentType) != "" {
		return // already resolved
	}
	username := CurrentUserName()
	endpoint := fmt.Sprintf("%s/api/v1/developers?username=%s", cfg.ServerURL, url.QueryEscape(username))

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var result struct {
		DeveloperID string `json:"developerId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}
	if result.DeveloperID != "" {
		CacheDeveloperID(cfg.AgentType, result.DeveloperID)
	}
}
