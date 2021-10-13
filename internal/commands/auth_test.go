//go:build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestAuthHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "auth")
}

func TestAuthNoSub(t *testing.T) {
	execCmdNilAssertion(t, "auth")
}

func TestRunCreateOath2ClientCommand(t *testing.T) {
	args := []string{"auth", "register", "--username", "username", "--password", "password"}
	execCmdNilAssertion(t, args...)

	// Create Oath2Client with roles
	execCmdNilAssertion(t, append(args, "--roles", "admin,user")...)
}

func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "auth", "register")
}

func TestRunCreateOath2ClientCommandNoPassword(t *testing.T) {
	err := execCmdNotNilAssertion(t, "auth", "register", "--username", "username")
	assert.Equal(t, err.Error(), "failed creating client: Please provide password flag")
}
