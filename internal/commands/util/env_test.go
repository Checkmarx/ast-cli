package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

// swapCredentialResolver binds a mock-backed resolver so secret lookups never
// reach the real OS keyring.
func swapCredentialResolver(t *testing.T) *mock.CredentialStoreMock {
	t.Helper()
	store := mock.NewCredentialStoreMock()
	path := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	credentialstore.SetDefaultResolverForTest(credentialstore.NewResolver(path, credentialstore.PolicyAuto, store))
	t.Cleanup(credentialstore.ResetForTest)
	return store
}

func TestNewEnvCheckCommand(t *testing.T) {
	cmd := NewEnvCheckCommand()
	assert.Assert(t, cmd != nil, "Env check command must exist")

	err := cmd.Execute()
	assert.NilError(t, err, "Env check command should run with no errors")
}

func captureEnvCheckOutput(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	assert.NilError(t, err)
	os.Stdout = w

	fn()

	assert.NilError(t, w.Close())
	os.Stdout = original

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NilError(t, err)
	return buf.String()
}

func TestRunEnvChecksShowsEffectiveValuesAndRedactsSecrets(t *testing.T) {
	defer viper.Reset()

	store := swapCredentialResolver(t)
	store.Store[credentialstore.CredentialAPIKey] = "supersecretvalue123"
	store.Store[credentialstore.CredentialClientSecret] = "topsecretabcd"
	viper.Set(params.AccessKeyIDConfigKey, "plain-client-id")
	viper.Set(params.BaseURIKey, "https://example.api.test")

	output := captureEnvCheckOutput(t, func() {
		err := runEnvChecks()(nil, nil)
		assert.NilError(t, err)
	})

	assert.Assert(t, strings.Contains(output, fmt.Sprintf("%30v: %s\n", "CX_APIKEY", "******e123")),
		"CX_APIKEY must be printed obfuscated, got:\n%s", output)
	assert.Assert(t, strings.Contains(output, fmt.Sprintf("%30v: %s\n", "CX_CLIENT_SECRET", "******abcd")),
		"CX_CLIENT_SECRET must be printed obfuscated, got:\n%s", output)
	assert.Assert(t, !strings.Contains(output, "supersecretvalue123"), "plaintext api key leaked:\n%s", output)
	assert.Assert(t, !strings.Contains(output, "topsecretabcd"), "plaintext client secret leaked:\n%s", output)

	assert.Assert(t, strings.Contains(output, fmt.Sprintf("%30v: %s\n", "CX_CLIENT_ID", "plain-client-id")))
	assert.Assert(t, strings.Contains(output, fmt.Sprintf("%30v: %s\n", "CX_BASE_URI", "https://example.api.test")))
}

func TestRunEnvChecksOutputIsDeterministic(t *testing.T) {
	swapCredentialResolver(t)

	first := captureEnvCheckOutput(t, func() {
		err := runEnvChecks()(nil, nil)
		assert.NilError(t, err)
	})
	second := captureEnvCheckOutput(t, func() {
		err := runEnvChecks()(nil, nil)
		assert.NilError(t, err)
	})
	assert.Equal(t, first, second, "utils env output must be deterministic across runs")
}
