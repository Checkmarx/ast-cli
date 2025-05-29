//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/viper"
	"gotest.tools/assert"
	"os"
	"strings"
	"testing"
)

const filePath = "data/config.yaml"

func TestLoadConfiguration_EnvVarConfigFilePath(t *testing.T) {
	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	_ = viper.BindEnv("CX_CONFIG_FILE_PATH")
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)
}

func TestLoadConfiguration_FileNotFound(t *testing.T) {
	os.Setenv("CX_CONFIG_FILE_PATH", "data/nonexistent_config.yaml")
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	_ = viper.BindEnv("CX_CONFIG_FILE_PATH")
	var err error
	err = configuration.LoadConfiguration()
	assert.ErrorContains(t, err, "The specified file does not exist")
}
func TestLoadConfiguration_ValidDirectory(t *testing.T) {
	validDirPath := "data"
	os.Setenv("CX_CONFIG_FILE_PATH", validDirPath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	_ = viper.BindEnv("CX_CONFIG_FILE_PATH")
	err := configuration.LoadConfiguration()
	assert.ErrorContains(t, err, "The specified path points to a directory")
}
func TestLoadConfiguration_FileWithoutPermission_UsingConfigFile(t *testing.T) {
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("failed to set file permissions: %v", err)
	}
	defer os.Chmod(filePath, 0644)

	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	_ = viper.BindEnv("CX_CONFIG_FILE_PATH")
	err := configuration.LoadConfiguration()

	assert.ErrorContains(t, err, "Access to the specified file is restricted")
}

func TestSetConfigProperty_EnvVarConfigFilePath(t *testing.T) {
	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	_ = viper.BindEnv("CX_CONFIG_FILE_PATH")
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)

	err, _ = executeCommand(t, "configure", "set", "--prop-name", "cx_client_id", "--prop-value", "dummy-client_id")
	assert.NilError(t, err)

	content, err := os.ReadFile(filePath)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(content), "dummy-client_id"))

	err, _ = executeCommand(t, "configure", "set", "--prop-name", "cx_client_id", "--prop-value", "example_client_id")
	assert.NilError(t, err)
}

func TestLoadConfiguration_ConfigFilePathFlag(t *testing.T) {
	err, _ := executeCommand(t, "configure", "show", "--config-file-path", filePath)
	assert.NilError(t, err)
}

func TestLoadConfiguration_ConfigFilePathFlagValidDirectory(t *testing.T) {
	err, _ := executeCommand(t, "configure", "show", "--config-file-path", "data")
	assert.ErrorContains(t, err, "The specified path points to a directory")
}

func TestLoadConfiguration_ConfigFilePathFlagFileNotFound(t *testing.T) {
	err, _ := executeCommand(t, "configure", "show", "--config-file-path", "data/nonexistent_config.yaml")
	assert.ErrorContains(t, err, "The specified file does not exist")
}

func TestSetConfigProperty_ConfigFilePathFlag(t *testing.T) {
	err, _ := executeCommand(t, "configure", "set", "--prop-name", "cx_client_id", "--prop-value", "dummy-client_id", "--config-file-path", filePath)
	assert.NilError(t, err)

	content, err := os.ReadFile(filePath)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(content), "dummy-client_id"))

	err, _ = executeCommand(t, "configure", "set", "--prop-name", "cx_client_id", "--prop-value", "example_client_id", "--config-file-path", filePath)
	assert.NilError(t, err)
}

//func TestLoadConfiguration_ConfigFilePathFlagFileWithoutPermission(t *testing.T) {
//	if err := os.Chmod(filePath, 0000); err != nil {
//		t.Fatalf("failed to set file permissions: %v", err)
//	}
//	defer os.Chmod(filePath, 0644)
//	err, _ := executeCommand(t, "configure", "show", "--config-file-path", filePath)
//	assert.ErrorContains(t, err, "Access to the specified file is restricted")
//}
