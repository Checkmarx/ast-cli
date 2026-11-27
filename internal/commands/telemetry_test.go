//go:build !integration

package commands

import (
	"io"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

func TestParseEventAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected wrappers.EventAttributes
	}{
		{name: "empty input returns nil", input: "", expected: nil},
		{name: "single pair", input: "apiVersion=v2", expected: wrappers.EventAttributes{"apiVersion": "v2"}},
		{
			name:     "arbitrary keys are all kept",
			input:    "apiVersion=v2;vulnerabilityId=CVE-2021-44228;remediationMode=DevAssist",
			expected: wrappers.EventAttributes{"apiVersion": "v2", "vulnerabilityId": "CVE-2021-44228", "remediationMode": "DevAssist"},
		},
		{name: "value containing equals keeps text after first equals", input: "token=a=b=c", expected: wrappers.EventAttributes{"token": "a=b=c"}},
		{name: "pair without equals is skipped", input: "a=1;bad;c=3", expected: wrappers.EventAttributes{"a": "1", "c": "3"}},
		{name: "empty key is skipped", input: "=v;b=2", expected: wrappers.EventAttributes{"b": "2"}},
		{name: "whitespace around key and value is trimmed", input: " a = 1 ; b=2 ", expected: wrappers.EventAttributes{"a": "1", "b": "2"}},
		{name: "semicolon in value splits into separate pairs", input: "a=x;y=2", expected: wrappers.EventAttributes{"a": "x", "y": "2"}},
		{name: "only malformed pairs returns nil", input: "bad;worse", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEventAttributes(tt.input)
			if tt.expected == nil {
				assert.Assert(t, got == nil, "expected nil attributes")
				return
			}
			assert.Assert(t, got != nil, "expected attributes")
			assert.DeepEqual(t, *got, tt.expected)
		})
	}
}

func TestRunTelemetryAI_PassesEventAttributesAndSessionID(t *testing.T) {
	originalSessionID := viper.GetString(params.SessionIDConfigKey)
	defer viper.Set(params.SessionIDConfigKey, originalSessionID)
	viper.Set(params.SessionIDConfigKey, "existing-session")

	var captured *wrappers.DataForAITelemetry
	telemetryWrapper := mock.TelemetryMockWrapper{
		CustomSendAIDataToLog: func(data *wrappers.DataForAITelemetry) error {
			captured = data
			return nil
		},
	}

	cmd := NewTelemetryCommand(telemetryWrapper)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{
		"ai", "--type", "click", "--engine", "secrets",
		"--event-attributes", "apiVersion=v2;vulnerabilityId=CVE-2021-44228;remediationMode=DevAssist",
	})
	err := cmd.Execute()

	assert.NilError(t, err)
	assert.Assert(t, captured != nil, "expected telemetry data to be sent")
	assert.Equal(t, "existing-session", captured.SessionID, "non-auth events reuse the stored session ID")
	assert.Assert(t, captured.EventAttributes != nil, "expected event attributes")
	assert.DeepEqual(t, *captured.EventAttributes, wrappers.EventAttributes{
		"apiVersion":      "v2",
		"vulnerabilityId": "CVE-2021-44228",
		"remediationMode": "DevAssist",
	})
}

func TestRunTelemetryAI_AuthenticationWithoutAgentSkipsSend(t *testing.T) {
	originalSessionID := viper.GetString(params.SessionIDConfigKey)
	t.Cleanup(func() { viper.Set(params.SessionIDConfigKey, originalSessionID) })
	viper.Set(params.SessionIDConfigKey, "existing-session")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "omitted"},
		{name: "blank", args: []string{"--agent", " "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			cmd := NewTelemetryCommand(mock.TelemetryMockWrapper{
				CustomSendAIDataToLog: func(_ *wrappers.DataForAITelemetry) error {
					called = true
					return nil
				},
			})
			cmd.SetOut(io.Discard)
			cmd.SetArgs(append([]string{"ai", "--type", "authentication", "--status", "success"}, tc.args...))

			assert.NilError(t, cmd.Execute())
			assert.Assert(t, !called, "authentication telemetry should be skipped")
			assert.Equal(t, viper.GetString(params.SessionIDConfigKey), "existing-session")
		})
	}
}

func TestRunTelemetryAI_SendToLogSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"telemetry", "ai", "--ai-provider", "Cursor", "--problem-severity", "Critical", "--type", "click", "--sub-type", "ast-results.viewPackageDetails", "--agent", "Cursor", "--engine", "secrets")
}

func TestTelemetryHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "telemetry", "ai")
}
