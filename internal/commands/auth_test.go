// +build !integration

package commands

/* Renable
func TestAuthHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "auth")
	assert.NilError(t, err)
}
*/

/* Renable
func TestAuthNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "auth")
	assert.NilError(t, err)
}
*/

/* Renable
func TestRunCreateOath2ClientCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "register", "--username", "username",
		"--password", "password")
	assert.NilError(t, err)

	err = executeTestCommand(cmd, "-v", "auth", "register", "-u", "username",
		"-p", "password", "--roles", "admin,user")
	assert.NilError(t, err)
}
*/

/* Renable
func TestRunCreateOath2ClientCommandInvalid(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "auth", "register")
	assert.Assert(t, err != nil)
}
*/
