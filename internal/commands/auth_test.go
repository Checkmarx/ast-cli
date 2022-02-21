//go:build !integration

package commands

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestAuthHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "auth")
}

func TestAuthNoSub(t *testing.T) {
	execCmdNilAssertion(t, "auth")
}

func TestAuthValidate(t *testing.T) {
	execCmdNotNilAssertion(t, "auth", "validate")
}

func TestRunCreateOath2ClientCommand(t *testing.T) {
	args := []string{
		"auth",
		"register",
		"--username",
		"username",
		"--password",
		"password",
		"--roles",
		strings.Join(RoleSlice, ","),
	}
	execCmdNilAssertion(t, args...)
}

func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "auth", "register")
}

func TestRunCreateOath2ClientCommandNoPassword(t *testing.T) {
	err := execCmdNotNilAssertion(
		t, "auth", "register", "--username", "username", "--roles", strings.Join(RoleSlice, ","),
	)
	assert.Equal(t, err.Error(), "failed creating client: Please provide password flag")
}

func TestRunCreateOath2ClientCommandNoRoles(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"auth",
		"register",
		"--username",
		"username",
		"--password",
		"password",
	)
	assert.Equal(t, err.Error(), "required flag(s) \"roles\" not set")
}
