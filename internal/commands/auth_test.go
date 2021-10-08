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
	err := executeTestCommand(cmd, "auth", "register", "--username", "username",
		"--password", "password")
	assert.NilError(t, err)

	err = executeTestCommand(cmd, "auth", "register", "-u", "username",
		"-p", "password", "--roles", "admin,user")
	assert.NilError(t, err)
}

func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "auth", "register")
	assert.Assert(t, err != nil)
}
