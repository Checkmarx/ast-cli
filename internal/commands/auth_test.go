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
	err := executeTestCommand(cmd, "-v", "auth", "create-oath2-client", "client", "--username", "username",
		"--password", "password", "--admin-client-id", "client-id", "--admin-client-secret", "secret")
	assert.NilError(t, err)

	err = executeTestCommand(cmd, "-v", "auth", "create-oath2-client", "client", "--username", "username",
		"--password", "password", "--admin-client-id", "client-id", "--admin-client-secret", "secret", "--roles", "admin,user")
	assert.NilError(t, err)
}

func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "create-oath2-client", "client")
	assert.Assert(t, err != nil)
}
