package governance

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/viper"
)

// GovernanceConfig holds the runtime settings needed by all governance hooks.
// Values are read at subprocess start from ~/.checkmarx/checkmarxcli.yaml so they
// are always fresh — no stale state from the parent CLI process.
type GovernanceConfig struct {
	ServerURL string // base URL of the governance Policy API Server
	Token     string // API key for authenticating governance HTTP requests
	AgentType string // "claude-code" | "cursor"
}

// Dedicated governance config keys in checkmarxcli.yaml — kept separate from
// the existing Checkmarx One keys so governance can point to a different server
// without affecting normal cx operations.
const (
	governanceURLKey    = "cx_governance_url"
	governanceTokenKey  = "cx_governance_api_key"
)

// LoadConfig reads governance settings from the shared Checkmarx CLI configuration.
// Priority for server URL: cx_governance_url → cx_base_uri (fallback).
// Priority for token:      cx_governance_api_key → cx_apikey (fallback).
// Config load failures are non-fatal; governance operates fail-open.
func LoadConfig(agentType string) GovernanceConfig {
	_ = configuration.LoadConfiguration()

	serverURL := viper.GetString(governanceURLKey)
	if serverURL == "" {
		serverURL = viper.GetString(params.BaseURIKey)
	}

	token := viper.GetString(governanceTokenKey)
	if token == "" {
		token = viper.GetString(params.AstAPIKey)
	}

	return GovernanceConfig{
		ServerURL: strings.TrimRight(serverURL, "/"),
		Token:     token,
		AgentType: agentType,
	}
}

// govBaseDir returns the root directory for all governance state on this machine.
// All sub-directories and files are scoped under this path.
// Falls back to ".checkmarx/governance" relative to cwd when the home directory
// cannot be determined — extremely rare, but ensures callers always get a non-empty path.
func govBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("governance: cannot find home directory: %v — using current directory fallback", err)
		return filepath.Join(".checkmarx", "governance")
	}
	return filepath.Join(home, ".checkmarx", "governance")
}

func spoolDir() string { return filepath.Join(govBaseDir(), "audit-queue") }

func logDir() string { return filepath.Join(govBaseDir(), "logs") }

func pendingAlertsDir() string { return filepath.Join(govBaseDir(), ".pending-alerts") }

func localPackPath() string { return filepath.Join(govBaseDir(), "policy-pack.json") }

func agentIdentityPath(agentType string) string {
	return filepath.Join(govBaseDir(), "agents", agentType, "agent.json")
}

func machineIDFilePath() string { return filepath.Join(govBaseDir(), "machine-id") }

func correlationDir() string { return filepath.Join(govBaseDir(), ".correlation") }

func hookVerdictsDir() string { return filepath.Join(govBaseDir(), ".hook-verdicts") }

func sessionStateDir() string { return filepath.Join(govBaseDir(), ".session-state") }

// ensureDir creates the directory if it does not exist.
func ensureDir(path string) {
	_ = os.MkdirAll(path, 0o700)
}
