package governance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// agentIdentity is the per-agent on-disk identity cache.
type agentIdentity struct {
	AgentID     string `json:"agentId"`
	DeveloperID string `json:"developerId,omitempty"`
}

var (
	machineIDOnce sync.Once
	cachedMachineID string
)

// MachineID returns a stable, per-machine identifier.
// Resolution priority:
//  1. Windows: HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid (registry)
//  2. Linux:   /etc/machine-id
//  3. Fallback: UUID generated once and persisted in ~/.checkmarx/governance/machine-id
func MachineID() string {
	machineIDOnce.Do(func() {
		cachedMachineID = resolveMachineID()
	})
	return cachedMachineID
}

func resolveMachineID() string {
	switch runtime.GOOS {
	case "windows":
		if id := windowsMachineID(); id != "" {
			return id
		}
	case "linux":
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				return id
			}
		}
	case "darwin":
		if id := darwinMachineID(); id != "" {
			return id
		}
	}
	return persistedMachineID()
}

// persistedMachineID reads or generates a UUID stored in the governance directory.
// This is the fallback for Windows machines without the registry key and non-Linux hosts.
func persistedMachineID() string {
	path := machineIDFilePath()
	if data, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}
	id := uuid.New().String()
	ensureDir(govBaseDir())
	_ = os.WriteFile(path, []byte(id), 0o600)
	return id
}

// CurrentUserName returns the OS username.
// Used only for AgentID derivation — never written to local session logs.
func CurrentUserName() string {
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}

// AgentID returns a stable, per-developer-per-agent-per-machine identifier.
// Derived as SHA256(SHA256(username)|SHA256(agentType)|machineId) and cached
// in ~/.checkmarx/governance/agents/<agentType>/agent.json.
func AgentID(agentType string) string {
	if id := readCachedAgentID(agentType); id != "" {
		return id
	}
	id := sha256hex(sha256hex(CurrentUserName()) + "|" + sha256hex(agentType) + "|" + MachineID())
	saveAgentIdentity(agentType, agentIdentity{AgentID: id})
	return id
}

// CachedDeveloperID returns the backend-resolved developer UUID for this agent type,
// or "" if EnsureResolved has not yet been called successfully.
func CachedDeveloperID(agentType string) string {
	ident := readAgentIdentity(agentType)
	if ident == nil {
		return ""
	}
	return ident.DeveloperID
}

// CacheDeveloperID persists the resolved developer UUID returned by the backend.
func CacheDeveloperID(agentType, developerID string) {
	ident := readAgentIdentity(agentType)
	if ident == nil {
		ident = &agentIdentity{AgentID: AgentID(agentType)}
	}
	ident.DeveloperID = developerID
	saveAgentIdentity(agentType, *ident)
}

func readCachedAgentID(agentType string) string {
	ident := readAgentIdentity(agentType)
	if ident == nil {
		return ""
	}
	return ident.AgentID
}

func readAgentIdentity(agentType string) *agentIdentity {
	data, err := os.ReadFile(agentIdentityPath(agentType))
	if err != nil {
		return nil
	}
	var ident agentIdentity
	if err := json.Unmarshal(data, &ident); err != nil {
		return nil
	}
	return &ident
}

func saveAgentIdentity(agentType string, ident agentIdentity) {
	path := agentIdentityPath(agentType)
	ensureDir(filepath.Join(govBaseDir(), "agents", agentType))
	data, _ := json.Marshal(ident)
	_ = os.WriteFile(path, data, 0o600)
}

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
