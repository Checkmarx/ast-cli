//go:build integration
// +build integration

package integration

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	clientIDPrefix = "ast-plugins-"
	AstUsernameEnv = "CX_AST_USERNAME"
	AstPasswordEnv = "CX_AST_PASSWORD"
)

// Test validate with credentials used in test env
func TestAuthValidate(t *testing.T) {
	validateCommand, buffer := createRedirectedTestCommand(t)

	err := execute(validateCommand, "auth", "validate")
	assert.NilError(t, err, "Validate should pass")

	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading result should pass")

	assert.Assert(t, strings.Contains(string(result), commands.SuccessAuthValidate))
}

// Register with credentials and validate the obtained id/secret pair
func TestAuthRegister(t *testing.T) {
	registerCommand, buffer := createRedirectedTestCommand(t)

	err := execute(registerCommand,
		"auth", "register",
		flag(params.UsernameFlag), viper.GetString(AstUsernameEnv),
		flag(params.PasswordFlag), viper.GetString(AstPasswordEnv),
	)
	assert.NilError(t, err, "Register should pass")

	result, err := io.ReadAll(buffer)
	assert.NilError(t, err, "Reading result should pass")

	lines := strings.Split(string(result), "\n")

	assert.Assert(t, strings.Contains(lines[0], "CX_CLIENT_ID="+clientIDPrefix))
	assert.Assert(t, strings.Contains(lines[1], "CX_CLIENT_SECRET="))

	clientID := strings.Split(lines[0], "=")[1]
	secret := strings.Split(lines[1], "=")[1]
	uuidLen := len(uuid.New().String())

	assert.Assert(t, strings.Contains(clientID, clientIDPrefix))
	assert.Assert(t, len(clientID) == len(clientIDPrefix)+uuidLen)
	assert.Assert(t, len(secret) == uuidLen)

	_, err = uuid.Parse(secret)
	assert.NilError(t, err, "Parsing UUID should pass")

	validateCommand, buffer := createRedirectedTestCommand(t)

	err = execute(validateCommand, "auth", "validate", flag(params.AccessKeyIDFlag), clientID, flag(params.AccessKeySecretFlag), secret)
	assert.NilError(t, err, "Validate should pass")

	result, err = io.ReadAll(buffer)
	assert.NilError(t, err, "Reading result should pass")

	assert.Assert(t, strings.Contains(string(result), commands.SuccessAuthValidate))
}

func TestFailProxyAuth(t *testing.T) {
	proxyUser := viper.GetString(ProxyUserEnv)
	proxyPort := viper.GetInt(ProxyPortEnv)
	proxyHost := viper.GetString(ProxyHostEnv)
	proxyArg := fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyUser, proxyHost, proxyPort)

	validate := createASTIntegrationTestCommand(t)

	args := []string{"auth", "validate", flag(params.DebugFlag), flag(params.ProxyFlag), proxyArg}
	validate.SetArgs(args)

	err := validate.Execute()
	assert.Assert(t, err != nil, "Executing without proxy should fail")
	//goland:noinspection GoNilness
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), "could not reach provided"))
}
