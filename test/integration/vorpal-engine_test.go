//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/viper"
	assert2 "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestScanVorpal_NoFileSourceSent_ReturnSuccess(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
}

func TestScanVorpal_SentFileWithoutExtension_FailCommandWithError(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file",
		flag(commonParams.VorpalLatestVersion),
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.FileExtensionIsRequired)
}

func TestExecuteVorpalScan_VorpalLatestVersionSetTrue_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
}

func TestExecuteVorpalScan_VorpalLatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, fmt.Sprintf("Should not fail with error: %v", err))
}

func TestExecuteVorpalScan_CorrectFlagsSent_SuccessfullyReturnMockData(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, fmt.Sprintf("Should not fail with error: %v", err))
}

func TestExecuteVorpalScan_UnsupportedLanguage_Fail(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/positive1.tf",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Scan should not fail with error")
	var scanResult grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResult)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert2.NotNil(t, scanResult.Error)
}

func TestExecuteVorpalScan_InitializeAndRunUpdateVersion_Success(t *testing.T) {
	configuration.LoadConfiguration()
	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	_ = vorpalWrapper.ShutDown()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}
	vorpalWrapper = grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	healthCheckErr := vorpalWrapper.HealthCheck()
	assert2.NotNil(t, healthCheckErr)
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")

	args = []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, _ = executeCommand(t, args...)
	assert.NilError(t, err, "Scan should not fail with error")
}

func TestExecuteVorpalScan_InitializeAndShutdown_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
		flag(commonParams.DebugFlag),
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	if healthCheckErr := vorpalWrapper.HealthCheck(); healthCheckErr != nil {
		t.Failed()
	}
	if shutdownErr := vorpalWrapper.ShutDown(); shutdownErr != nil {
		t.Failed()
	}
	err = vorpalWrapper.HealthCheck()
	assert2.NotNil(t, err)
}
