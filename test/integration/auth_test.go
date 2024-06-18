//go:build integrationzz

package integration

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	AstUsernameEnv                  = "CX_AST_USERNAME"
	AstPasswordEnv                  = "CX_AST_PASSWORD"
	defaultSuccessValidationMessage = "Validation should pass"
	incorrectAuthURL                = "404 Provided Tenant Name is invalid"
)

// Test validate with credentials used in test env
func TestAuthValidate(t *testing.T) {
	err, buffer := executeCommand(t, "auth", "validate")
	assertSuccessAuthentication(t, err, buffer, defaultSuccessValidationMessage)
}

func TestAuthValidateClientAndSecret(t *testing.T) {
	err, buffer := executeCommand(t, "auth", "validate", "--debug", "--apikey", "")
	assertSuccessAuthentication(t, err, buffer, defaultSuccessValidationMessage)
}

// Test validate with credentials from flags
func TestAuthValidateMissingFlagsTogether(t *testing.T) {
	// set base-uri to empty string so that it does not pick up the value from the environment
	err, _ := executeCommand(t, "auth", "validate", "--client-id", "fake-client-id", "--client-secret", "fake-client-secret", "--base-uri", "", "--base-auth-uri", "", "--apikey", "")
	assertError(t, err, wrappers.MissingURI)
}

// Test validate with credentials from flags
func TestAuthValidateEmptyFlags(t *testing.T) {
	err, _ := executeCommand(t, "auth", "validate", "--apikey", "", "--client-id", "")
	assertError(t, err, commands.FailedAuthError)

	err, _ = executeCommand(t, "auth", "validate", "--client-id", "client_id", "--apikey", "", "--client-secret", "")
	assertError(t, err, commands.FailedAuthError)
}

// Tests with base auth uri
func TestAuthValidateWithBaseAuthURI(t *testing.T) {
	validateCommand, buffer := createRedirectedTestCommand(t)

	avoidCachedToken()

	err := execute(validateCommand, "auth", "validate", "--debug", "--apikey", "")
	assertSuccessAuthentication(t, err, buffer, "")

	// valid authentication passing an empty base-auth-uri once it will be picked from environment variables
	err = execute(validateCommand, "auth", "validate", "--base-auth-uri", "")
	assertSuccessAuthentication(t, err, buffer, "")

	avoidCachedToken()

	err = execute(validateCommand, "auth", "validate", "--tenant", "invalid-base-uri")
	assertError(t, err, incorrectAuthURL)
}

// Test validate authentication with empty tenant
func TestAuthValidateWrongTenantWithBaseAuth(t *testing.T) {
	validateCommand, _ := createRedirectedTestCommand(t)

	// avoid picking cached token to allow invalid api key to be used
	avoidCachedToken()

	err := execute(validateCommand, "auth", "validate", "--tenant", "")
	assert.NilError(t, err)
}

// Test validate authentication with a wrong api key
func TestAuthValidateWrongAPIKey(t *testing.T) {
	validateCommand, _ := createRedirectedTestCommand(t)

	// avoid picking cached token to allow invalid api key to be used
	avoidCachedToken()

	err := execute(validateCommand, "auth", "validate", "--apikey", "invalidAPIKey", "--base-auth-uri", "")
	assertError(t, err, fmt.Sprintf(wrappers.APIKeyDecodeErrorFormat, ""))
}

func TestAuthValidateWithEmptyAuthenticationPath(t *testing.T) {
	validateCommand, _ := createRedirectedTestCommand(t)

	// avoid picking cached token to allow invalid api key to be used
	_ = viper.BindEnv("cx_ast_authentication_path", "CX_AST_AUTHENTICATION_PATH")
	viper.SetDefault("cx_ast_authentication_path", "")

	err := execute(validateCommand, "auth", "validate")
	assert.NilError(t, err)
}

func TestAuthValidateOnlyAPIKey(t *testing.T) {
	validateCommand, buffer := createRedirectedTestCommand(t)
	err := execute(validateCommand, "auth", "validate", "--base-uri", "", "--client-id", "", "--client-secret", "")
	assertSuccessAuthentication(t, err, buffer, "")
}

// Register with empty username, password or role
func TestAuthRegisterWithEmptyParameters(t *testing.T) {
	assertRequiredParameter(
		t, "Please provide username flag",
		"auth", "register",
		flag(params.UsernameFlag), "",
		flag(params.PasswordFlag), viper.GetString(AstPasswordEnv),
		flag(params.ClientRolesFlag), "ast-admin,ast-scanner",
	)

	assertRequiredParameter(
		t, "Please provide password flag",
		"auth", "register",
		flag(params.UsernameFlag), viper.GetString(AstUsernameEnv),
		flag(params.PasswordFlag), "",
		flag(params.ClientRolesFlag), "ast-admin,ast-scanner",
	)

	assertRequiredParameter(
		t, "Please provide roles flag",
		"auth", "register",
		flag(params.UsernameFlag), viper.GetString(AstUsernameEnv),
		flag(params.PasswordFlag), viper.GetString(AstPasswordEnv),
		flag(params.ClientRolesFlag), "",
	)
}

// Register with credentials and validate the obtained id/secret pair
func TestAuthRegister(t *testing.T) {
	registerCommand, _ := createRedirectedTestCommand(t)

	_ = execute(
		registerCommand,
		"auth", "register",
		flag(params.UsernameFlag), viper.GetString(AstUsernameEnv),
		flag(params.PasswordFlag), viper.GetString(AstPasswordEnv),
		flag(params.ClientRolesFlag), strings.Join(commands.RoleSlice, ","),
	)
	// Ignored assert as auth register has issues with MFA enabled users
	// AND the CLI user agent is rejected in prod for this command by cloudfront
	// assert.Assert(t, err == nil)
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

//func TestFailProxyAuthByEnv(t *testing.T) {
//	proxyValue := os.Getenv(params.ProxyEnv)
//	defer func() {
//		_ = os.Setenv(params.ProxyEnv, proxyValue)
//	}()
//	_ = os.Setenv(params.ProxyEnv, buildProxyURL())
//
//	validate := createASTIntegrationTestCommand(t)
//	args := []string{"auth", "validate", flag(params.DebugFlag)}
//	validate.SetArgs(args)
//	err := validate.Execute()
//	assert.NilError(t, err)
//
//	proxyValue = os.Getenv(params.CxProxyEnv)
//	defer func() {
//		_ = os.Setenv(params.CxProxyEnv, proxyValue)
//	}()
//	_ = os.Setenv(params.CxProxyEnv, "http://localhost:55555")
//
//	viper.Reset()
//
//	validate = createASTIntegrationTestCommand(t)
//	args = append(args, flag(params.RetryFlag), string(rune(0)))
//	validate.SetArgs(args)
//	err = validate.Execute()
//	assert.Assert(t, err != nil)
//}

// assert success authentication
func assertSuccessAuthentication(t *testing.T, err error, buffer *bytes.Buffer, assertionMessage string) {
	assert.NilError(t, err, assertionMessage)

	result, readingError := io.ReadAll(buffer)
	assert.NilError(t, readingError, "Reading result should pass")
	assert.Assert(t, strings.Contains(string(result), commands.SuccessAuthValidate))
}

// avoid picking cached token
func avoidCachedToken() {
	_ = viper.BindEnv("cx_token_expiry_seconds", "CX_TOKEN_EXPIRY_SECONDS")
	viper.SetDefault("cx_token_expiry_seconds", 10)
}
