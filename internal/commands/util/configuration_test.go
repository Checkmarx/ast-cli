package util

import (
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()
	assert.Assert(t, cmd != nil, "Config command must exist")

	// Test show command
	err := executeTestCommand(cmd, "show")
	assert.NilError(t, err)

	// Test configure command
	err = executeTestCommand(cmd, "set", "--prop-name", "cx_client_id", "--prop-value", "dummy-client_id")
	assert.NilError(t, err)

	// Test configure command unknown property
	err = executeTestCommand(cmd, "set", "--prop-name", "nonexistent_prop", "--prop-value", "dummy")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "Failed to set property: unknown property or bad value")
}

func TestGetConfigFilePath(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Check if the config file path is correct",
			want:    ".checkmarx/checkmarxcli.yaml",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			got, err := configuration.GetConfigFilePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			asserts.True(t, strings.HasSuffix(got, tt.want))
		})
	}
}

func TestWriteSingleConfigKeyToExistingFile(t *testing.T) {
	configFilePath, _ := configuration.GetConfigFilePath()
	err := configuration.WriteSingleConfigKey(configFilePath, "cx_asca_port", 0)
	assert.NilError(t, err)
}

func TestWriteSingleConfigKeyToNonExistingFile(t *testing.T) {
	configFilePath := "non-existing-file"

	file, err := os.Open(configFilePath)
	asserts.NotNil(t, err)
	asserts.Nil(t, file)

	err = configuration.WriteSingleConfigKey(configFilePath, "cx_asca_port", 0)
	assert.NilError(t, err)

	file, err = os.Open(configFilePath)
	assert.NilError(t, err)
	defer func(file *os.File) {
		_ = file.Close()
		_ = os.Remove(configFilePath)
		_ = os.Remove(configFilePath + ".lock")
	}(file)
	asserts.NotNil(t, file)
}

func TestChangedOnlyAscaPortInConfigFile(t *testing.T) {
	configFilePath, _ := configuration.GetConfigFilePath()

	oldConfig, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)

	err = configuration.WriteSingleConfigKey(configFilePath, "cx_asca_port", -1)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, -1, config["cx_asca_port"])

	// Assert all the other properties are the same
	for key, value := range oldConfig {
		if key != "cx_asca_port" {
			asserts.Equal(t, value, config[key])
		}
	}
}
