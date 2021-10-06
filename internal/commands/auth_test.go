//go:build !integration
// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestAuthHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "auth")
	assert.NilError(t, err)
}

func TestAuthNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "auth")
	assert.NilError(t, err)
}

func TestRunCreateOath2ClientCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "register", "--username", "username",
		"--password", "password")
	assert.NilError(t, err)

	err = executeTestCommand(cmd, "-v", "auth", "register", "-u", "username",
		"-p", "password", "--roles", "admin,user")
	assert.NilError(t, err)
}

func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "register")
	assert.Assert(t, err != nil)
}

func TestRunCreateOath2ClientCommandNoPassword(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "register", "--username", "username")
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), "failed creating client: Please provide password flag")
}
