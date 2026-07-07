package util

import (
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

const (
	cxAscaPort                 = "cx_asca_port"
	cxScsScanOverviewPath      = "cx_scs_scan_overview_path"
	defaultScsScanOverviewPath = "api/micro-engines/read/scans/%s/scan-overview"
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

func TestGetConfigFilePath_CheckmarxConfigFileExists_Success(t *testing.T) {
	want := ".checkmarx/checkmarxcli.yaml"
	got, err := configuration.GetConfigFilePath()

	if err != nil {
		t.Errorf("GetConfigFilePath() error = %v, wantErr = false", err)
		return
	}

	asserts.True(t, strings.HasSuffix(got, want), "Expected config file path to end with %q, but got %q", want, got)
}

func TestWriteSingleConfigKeyToExistingFile_ChangeAscaPortToZero_Success(t *testing.T) {
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)

	configFilePath, _ := configuration.GetConfigFilePath()
	err = configuration.SafeWriteSingleConfigKey(configFilePath, cxAscaPort, 0)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, 0, config[cxAscaPort])
}

func TestWriteSingleConfigKeyNonExistingFile_CreatingTheFileAndWritesTheKey_Success(t *testing.T) {
	configFilePath := "non-existing-file"

	file, err := os.Open(configFilePath)
	asserts.NotNil(t, err)
	asserts.Nil(t, file)

	err = configuration.SafeWriteSingleConfigKey(configFilePath, cxAscaPort, 0)
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

func TestChangedOnlyAscaPortInConfigFile_ConfigFileExistsWithDefaultValues_OnlyAscaPortChangedSuccess(t *testing.T) {
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)

	configFilePath, _ := configuration.GetConfigFilePath()

	oldConfig, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)

	err = configuration.SafeWriteSingleConfigKey(configFilePath, cxAscaPort, -1)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, -1, config[cxAscaPort])

	// Assert all the other properties are the same
	for key, value := range oldConfig {
		if key != cxAscaPort {
			asserts.Equal(t, value, config[key])
		}
	}
}

func TestWriteSingleConfigKeyStringToExistingFile_UpdateScsScanOverviewPath_Success(t *testing.T) {
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)

	configFilePath, _ := configuration.GetConfigFilePath()
	err = configuration.SafeWriteSingleConfigKeyString(configFilePath, cxScsScanOverviewPath, defaultScsScanOverviewPath)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, defaultScsScanOverviewPath, config[cxScsScanOverviewPath])
}

func TestWriteSingleConfigKeyStringNonExistingFile_CreatingTheFileAndWritesTheKey_Success(t *testing.T) {
	configFilePath := "non-existing-file"

	file, err := os.Open(configFilePath)
	asserts.NotNil(t, err)
	asserts.Nil(t, file)

	err = configuration.SafeWriteSingleConfigKeyString(configFilePath, cxScsScanOverviewPath, defaultScsScanOverviewPath)
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

func TestLoadConfigEmptyFile_ZeroByteConfig_ReturnsEmptyMapWithoutError(t *testing.T) {
	configFilePath := "empty-config-file.yaml"

	// A zero-byte config file is the state that previously caused
	// "error decoding YAML: EOF" when cx auth login persisted a fresh token.
	file, err := os.Create(configFilePath)
	assert.NilError(t, err)
	_ = file.Close()
	defer func() {
		_ = os.Remove(configFilePath)
		_ = os.Remove(configFilePath + ".lock")
	}()

	loaded, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, 0, len(loaded), "an empty file must load as an empty config, not an error")
}

func TestWriteSingleConfigKeyStringEmptyFile_ZeroByteConfig_WritesKeyWithoutEOFError(t *testing.T) {
	configFilePath := "empty-config-write.yaml"

	file, err := os.Create(configFilePath)
	assert.NilError(t, err)
	_ = file.Close()
	defer func() {
		_ = os.Remove(configFilePath)
		_ = os.Remove(configFilePath + ".lock")
	}()

	// Previously failed with "error loading config: error decoding YAML: EOF".
	err = configuration.SafeWriteSingleConfigKeyString(configFilePath, cxScsScanOverviewPath, defaultScsScanOverviewPath)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, defaultScsScanOverviewPath, config[cxScsScanOverviewPath])
}

func TestLoadConfigMalformedYaml_CorruptConfig_StillReturnsError(t *testing.T) {
	configFilePath := "malformed-config.yaml"
	// Only the empty-file (io.EOF) case is special-cased. A genuinely corrupt
	// config must still error — this pins that narrow behavior so a future
	// change can't broaden it into silently swallowing (and then truncating)
	// a real config.
	err := os.WriteFile(configFilePath, []byte("foo: [bar"), 0600)
	assert.NilError(t, err)
	defer func() { _ = os.Remove(configFilePath) }()

	cfg, err := configuration.LoadConfig(configFilePath)
	asserts.NotNil(t, err)
	asserts.Nil(t, cfg)
	asserts.True(t, strings.Contains(err.Error(), "error decoding YAML"), "corrupt YAML must still report a decode error")
}

func TestLoadConfigCommentOnlyFile_EffectivelyEmpty_ReturnsEmptyMap(t *testing.T) {
	configFilePath := "comment-only-config.yaml"
	// A comment-only / whitespace-only file decodes to io.EOF in yaml.v3, the
	// same as a zero-byte file — it must load as an empty config, not an error.
	err := os.WriteFile(configFilePath, []byte("# only a comment\n"), 0600)
	assert.NilError(t, err)
	defer func() { _ = os.Remove(configFilePath) }()

	cfg, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, 0, len(cfg))
}

func TestChangedOnlyScsScanOverviewPathInConfigFile_ConfigFileExistsWithDefaultValues_OnlyScsScanOverviewPathChangedSuccess(t *testing.T) {
	err := configuration.LoadConfiguration()
	assert.NilError(t, err)

	configFilePath, _ := configuration.GetConfigFilePath()

	oldConfig, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)

	err = configuration.SafeWriteSingleConfigKeyString(configFilePath, cxScsScanOverviewPath, defaultScsScanOverviewPath)
	assert.NilError(t, err)

	config, err := configuration.LoadConfig(configFilePath)
	assert.NilError(t, err)
	asserts.Equal(t, defaultScsScanOverviewPath, config[cxScsScanOverviewPath])

	// Assert all the other properties are the same
	for key, value := range oldConfig {
		if key != cxScsScanOverviewPath {
			asserts.Equal(t, value, config[key])
		}
	}
}

func TestGetConfigFilePath_CustomFile(t *testing.T) {
	expectedPath := "/custom/path/checkmarxcli.yaml"
	viper.Set(params.ConfigFilePathKey, expectedPath)

	actualPath, err := configuration.GetConfigFilePath()
	assert.NilError(t, err)
	assert.Equal(t, actualPath, expectedPath, "Expected path to match the set value in viper")
}
