//go:build integration

package integration

import (
	"gotest.tools/assert"
	"os"
	"strings"
	"testing"
)

var filePath = "data/config.yaml"

func TestIntegration_LoadConfiguration_EnvVarConfigFilePath(t *testing.T) {
	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	cmd, buffer := createRedirectedTestCommand(t)

	err := execute(cmd, "configure", "show")
	assert.NilError(t, err)

	output := buffer.String()
	if !strings.Contains(output, "cx_base_uri: https://example.com") {
		t.Errorf("expected output to contain 'cx_base_uri: https://example.com', but it did not")
	}
}

func TestIntegration_LoadConfiguration_FileNotFound(t *testing.T) {
	// Set an invalid file path
	os.Setenv("CX_CONFIG_FILE_PATH", "data/nonexistent_config.yaml")
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	cmd, buffer := createRedirectedTestCommand(t)

	// Execute the command
	err := execute(cmd, "configure", "show")

	// Verify that an error occurred
	assert.ErrorContains(t, err, "The specified file does not exist")

	// Verify the output contains the expected error message
	output := buffer.String()
	if !strings.Contains(output, "The specified file does not exist") {
		t.Errorf("expected output to contain 'The specified file does not exist', but it did not")
	}
}

func TestIntegration_LoadConfiguration_FileWithoutPermission_UsingConfigFile(t *testing.T) {

	// Set the file to have no permissions
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("failed to set file permissions: %v", err)
	}
	defer os.Chmod(filePath, 0644) // Restore permissions for cleanup

	// Set the environment variable to point to the restricted file
	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
	defer os.Unsetenv("CX_CONFIG_FILE_PATH")

	cmd, buffer := createRedirectedTestCommand(t)

	// Execute the command
	err := execute(cmd, "configure", "show")

	// Verify that an error occurred
	assert.ErrorContains(t, err, "Access to the specified file is restricted")

	// Verify the output contains the expected error message
	output := buffer.String()
	if !strings.Contains(output, "Access to the specified file is restricted") {
		t.Errorf("expected output to contain 'Access to the specified file is restricted', but it did not")
	}
}

//
//func TestIntegration_SetConfigProperty_EnvVarConfigFilePath(t *testing.T) {
//	// Set environment variable
//	os.Setenv("CX_CONFIG_FILE_PATH", filePath)
//	defer os.Unsetenv("CX_CONFIG_FILE_PATH")
//
//	// Create command
//	cmd, buffer := createRedirectedTestCommand(t)
//
//	// Execute command
//	err := execute(cmd, "configure", "set", "--prop-name", "new_key", "--prop-value", "new_value")
//	assert.NilError(t, err)
//
//	// Verify output
//	output := buffer.String()
//	asserts.Contains(t, output, "Setting property [new_key] to value [new_value]")
//
//	// Verify configuration
//	asserts.Equal(t, viper.GetString("new_key"), "new_value")
//}
