package commands

import (
	"os/exec"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--problem-severity", "Critical", "--type", "click", "--sub-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}

func TestTelemetryHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "telemetry", "ai")
}

func TestRunTelemetryAI_PassesGitHubURL(t *testing.T) {
	var got *wrappers.DataForAITelemetry
	cmd := telemetryAISubCommand(mock.TelemetryMockWrapper{
		CustomSendAIDataToLog: func(data *wrappers.DataForAITelemetry) error {
			got = data
			return nil
		},
	})
	cmd.SetArgs([]string{"--type", "click", "--github-url", "git@github.com:acme/widgets.git"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got == nil || got.GithubURL != "https://github.com/acme/widgets" || got.Type != "click" {
		t.Fatalf("payload = %+v", got)
	}
}

func TestRunTelemetryAI_DetectsGitHubURLFromCwd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	gitIn("init")
	gitIn("remote", "add", "origin", "https://github.com/acme/from-cwd.git")
	t.Chdir(dir)

	var got *wrappers.DataForAITelemetry
	cmd := telemetryAISubCommand(mock.TelemetryMockWrapper{
		CustomSendAIDataToLog: func(data *wrappers.DataForAITelemetry) error {
			got = data
			return nil
		},
	})
	cmd.SetArgs([]string{"--type", "click"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got == nil || got.GithubURL != "https://github.com/acme/from-cwd" {
		t.Fatalf("GithubURL = %+v", got)
	}
}
