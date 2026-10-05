package util

import (
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/logger"
	"gotest.tools/assert"
)

// Resolved credentials are registered with the logger sanitizer at the
// resolver choke point, so debug request dumps never print them even though
// they no longer live in viper.
func TestResolvedSecretIsMaskedInLogs(t *testing.T) {
	store := swapCredentialResolver(t)
	const secret = "qa-secret-abcdef123456"
	store.Store[credentialstore.CredentialAPIKey] = secret

	got, err := credentialstore.Resolve(credentialstore.CredentialAPIKey)
	assert.NilError(t, err)
	assert.Equal(t, secret, got)

	var buf strings.Builder
	original := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(original)

	logger.Print("grant_type=refresh_token&client_id=qa&refresh_token=" + secret)

	assert.Assert(t, !strings.Contains(buf.String(), secret), "plaintext secret leaked to logs: %s", buf.String())
	assert.Assert(t, strings.Contains(buf.String(), "***"), "expected masked log output: %s", buf.String())
}

// A failed secret write must fail the command (fail-hard), not print and
// pretend success.
func TestConfigureSetSecretWriteFailureFailsCommand(t *testing.T) {
	store := swapCredentialResolver(t)
	store.SetErr = errors.New("keyring write failed")

	err := executeTestCommand(NewConfigCommand(), "set", "--prop-name", "cx_apikey", "--prop-value", "whatever")

	assert.Assert(t, err != nil, "expected non-nil error on keyring write failure")
	assert.ErrorContains(t, err, "storing cx_apikey")
}
