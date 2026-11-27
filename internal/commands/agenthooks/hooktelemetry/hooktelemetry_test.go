//go:build !integration

package hooktelemetry

import (
	"os"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/viper"
	"gotest.tools/assert"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// testRemotePassword is the password embedded in a credentialed remote URL fixture. It is read from the
// environment so no credential is hardcoded; any non-empty value exercises the stripping logic.
func testRemotePassword() string {
	if v := os.Getenv("CX_TEST_REMOTE_PASSWORD"); v != "" {
		return v
	}
	return "unused"
}

func TestSanitizeRepoURL(t *testing.T) {
	credentialedURL := "https://user:" + testRemotePassword() + "@github.com/acme/repo"
	tests := []struct{ in, want string }{
		{"https://github.com/acme/repo.git", "https://github.com/acme/repo"},
		{"https://token@github.com/acme/repo.git", "https://github.com/acme/repo"},
		{credentialedURL, "https://github.com/acme/repo"},
		{"https://github.com/acme/repo?token=abc#frag", "https://github.com/acme/repo"},
		{"ssh://git@github.com:22/acme/repo.git", "https://github.com/acme/repo"},
		{"git@github.com:acme/repo.git", "https://github.com/acme/repo"},
		{"https://dev.azure.com:443/org/proj/_git/repo/", "https://dev.azure.com/org/proj/_git/repo"},
		{`C:\work\repo`, ""},
		{"/home/me/repo", ""},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, SanitizeRepoURL(tt.in), tt.want, "input %q", tt.in)
	}
}

func TestSeverityCounts(t *testing.T) {
	got := SeverityCounts([]string{"High", "critical", "High", "Low", "unknown"})
	assert.Equal(t, got, "Critical=1,High=2,Medium=0,Low=1,Info=0")
	assert.Equal(t, SeverityCounts(nil), "Critical=0,High=0,Medium=0,Low=0,Info=0")
}

func TestEnrich(t *testing.T) {
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	assert.NilError(t, err)
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"https://token@github.com/acme/repo.git"}})
	assert.NilError(t, err)

	t.Setenv(params.AgentVersionEnv, " 2.1.4 ")
	t.Setenv(params.CxPluginVersionEnv, "1.0.3")
	t.Setenv(params.PluginTypeEnv, "CLI")
	original := viper.GetString(params.SessionIDConfigKey)
	defer viper.Set(params.SessionIDConfigKey, original)
	viper.Set(params.SessionIDConfigKey, "sess-123")

	data := &wrappers.DataForAITelemetry{Type: "hooks-detect"}
	Enrich(data, dir)

	assert.Equal(t, data.AgentVersion, "2.1.4")
	assert.Equal(t, data.CxPluginVersion, "1.0.3")
	assert.Equal(t, data.PluginType, "CLI")
	assert.Equal(t, data.SessionID, "sess-123")
	assert.Equal(t, data.RepoURL, "https://github.com/acme/repo")
	assert.Equal(t, data.Status, StatusSuccess)
}

func TestFindingIDs(t *testing.T) {
	SetFindingIDs([]string{"CVE-1", " ", "CVE-2", "CVE-1"})
	assert.Equal(t, TakeFindingIDs(), "CVE-1,CVE-2", "blanks and duplicates are dropped")
	assert.Equal(t, TakeFindingIDs(), "", "taking clears the recorded ids")

	many := make([]string, 0, maxFindingIDs+5)
	for i := 0; i < maxFindingIDs+5; i++ {
		many = append(many, "id"+string(rune('a'+i)))
	}
	SetFindingIDs(many)
	assert.Equal(t, len(strings.Split(TakeFindingIDs(), ",")), maxFindingIDs, "capped at maxFindingIDs")
}

func TestEnrichToleratesMissingContext(t *testing.T) {
	Enrich(nil, "") // must not panic
	data := &wrappers.DataForAITelemetry{Status: "failure"}
	Enrich(data, t.TempDir()) // not a git repo
	assert.Equal(t, data.RepoURL, "")
	assert.Equal(t, data.Status, "failure", "an explicit status is kept")
}

func TestIgnoreFlags(t *testing.T) {
	t.Setenv("PLUGIN_ROOT", "")
	t.Setenv("CX_COPILOT_HOST", "vscode")
	t.Setenv(params.AgentVersionEnv, " 1.2.3 ")
	t.Setenv(params.CxPluginVersionEnv, "2.3.4")
	t.Setenv(params.PluginTypeEnv, "CLI")

	want := " --optional-flags \"aiProvider=Copilot;agent=Copilot-VSCode;aiAgentSessionId=session-1;agentVersion=1.2.3;cxPluginVersion=2.3.4;pluginType=CLI\""
	assert.Equal(t, IgnoreFlags("Copilot", "session-1"), want)

	t.Setenv(params.AgentVersionEnv, "1;pluginType=unexpected")
	t.Setenv(params.CxPluginVersionEnv, "")
	assert.Equal(t, IgnoreFlags("Cursor", "invalid session"), " --optional-flags \"aiProvider=Cursor;agent=Cursor-cli;pluginType=CLI\"")
	assert.Equal(t, IgnoreFlags("", ""), "")
}
