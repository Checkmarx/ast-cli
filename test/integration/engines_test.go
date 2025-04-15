//go:build integration

package integration

import (
	"gotest.tools/assert"
	"testing"
)

/*
func TestEnginesApiList_WithOnlyValidApikeyFlag_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"engines", "api-list",
		flag(params.AstAPIKeyFlag), originals[params.AstAPIKeyEnv],
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_WithOnlyValidApikeyEnvVar_Success(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"engines", "api-list",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_WithOnlyInvalidApikeyEnvVar_Fail(t *testing.T) {
	originals := getOriginalEnvVars()

	setEnvVars(map[string]string{
		params.AstAPIKeyEnv:       invalidAPIKey,
		params.AccessKeyIDEnv:     invalidClientID,
		params.AccessKeySecretEnv: invalidClientSecret,
		params.TenantEnv:          invalidTenant,
	})

	defer setEnvVars(originals)

	args := []string{
		"engines", "api-list",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
	assert.Error(t, err, "Error validating scan types: Token decoding error: token is malformed: token contains an invalid number of segments")
}
*/

func TestEnginesApiList_WithoutFlagSuccess(t *testing.T) {
	args := []string{
		"engines", "api-list",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_HelpSuccess(t *testing.T) {
	args := []string{
		"engines",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeSuccess1(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeSuccess2(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "SAST",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeSuccess3(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "SCA",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeSuccess4(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "Iac",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeError1(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "xyz",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_EngineTypeError2(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engine-name", "",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_OutputFormatSuccess1(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "table", "--engine-name", "Iac",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_OutputFormatSuccess2(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "json", "--engine-name", "Iac",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_OutputFormatSuccess3(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "yaml", "--engine-name", "Iac",
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err)
}

func TestEnginesApiList_OutputFormatError1(t *testing.T) {
	args := []string{
		"engines", "list-api", "--output-format", "xyz", "--engine-name", "Iac",
	}

	err, _ := executeCommand(t, args...)
	assert.Equal(t, err.Error(), "Invalid format xyz")
}

func TestEnginesApiList_FlagError1(t *testing.T) {
	args := []string{
		"engines", "list-api", "--engines-name",
	}

	err, _ := executeCommand(t, args...)
	assert.Equal(t, err.Error(), "unknown flag: --engines-name")
}
