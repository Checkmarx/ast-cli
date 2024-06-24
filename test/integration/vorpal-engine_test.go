//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfig"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/viper"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestScanVorpal_NoFileSourceSent_ReturnSuccess(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteVorpalScan_VorpalLatestVersionSetTrue_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteVorpalScan_NoSourceAndVorpalLatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()
	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	_ = vorpalWrapper.ShutDown()
	_ = os.RemoveAll(vorpalconfig.Params.WorkingDir())
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteVorpalScan_NotExistingFile_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "not-existing-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Error.Description == fmt.Sprintf(services.FileNotFound, "not-existing-file.py"), "should return error: ", services.FileNotFound)
}

func TestExecuteVorpalScan_VorpalLatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, scanResults := executeCommand(t, args...)
	assert.NilError(t, err, fmt.Sprintf("Should not fail with error: %v", err))
	asserts.NotNil(t, scanResults)
	var scanResult grpcs.ScanResult
	err = json.Unmarshal(scanResults.Bytes(), &scanResult)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	asserts.Nil(t, scanResult.Error)
	asserts.NotNil(t, scanResult.ScanDetails)
}

func TestExecuteVorpalScan_NoEngineInstalledAndVorpalLatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()

	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	_ = vorpalWrapper.ShutDown()
	_ = os.RemoveAll(vorpalconfig.Params.WorkingDir())

	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, scanResults := executeCommand(t, args...)
	assert.NilError(t, err, fmt.Sprintf("Should not fail with error: %v", err))
	asserts.NotNil(t, scanResults)
	var scanResult grpcs.ScanResult
	err = json.Unmarshal(scanResults.Bytes(), &scanResult)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	asserts.Nil(t, scanResult.Error)
	asserts.NotNil(t, scanResult.ScanDetails)
}

func TestExecuteVorpalScan_CorrectFlagsSent_SuccessfullyReturnMockData(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, scanResults := executeCommand(t, args...)
	assert.NilError(t, err, fmt.Sprintf("Should not fail with error: %v", err))
	asserts.NotNil(t, scanResults)
	var scanResult grpcs.ScanResult
	err = json.Unmarshal(scanResults.Bytes(), &scanResult)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	asserts.Nil(t, scanResult.Error)
	asserts.NotNil(t, scanResult.ScanDetails)
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
	asserts.NotNil(t, scanResult.Error)
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
	asserts.NotNil(t, healthCheckErr)
	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteVorpalScan_InitializeAndShutdown_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
		flag(commonParams.DebugFlag),
	}
	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)

	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	if healthCheckErr := vorpalWrapper.HealthCheck(); healthCheckErr != nil {
		assert.Assert(t, healthCheckErr == nil, "Health check failed with error: ", healthCheckErr)
	}
	if shutdownErr := vorpalWrapper.ShutDown(); shutdownErr != nil {
		assert.Assert(t, shutdownErr == nil, "Shutdown failed with error: ", shutdownErr)
	}
	err = vorpalWrapper.HealthCheck()
	asserts.NotNil(t, err)
}

func TestExecuteVorpalScan_EngineNotRunningWithLicense_Success(t *testing.T) {
	configuration.LoadConfiguration()
	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	_ = vorpalWrapper.ShutDown()
	_ = os.RemoveAll(vorpalconfig.Params.WorkingDir())
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.DebugFlag),
		flag(commonParams.AgentFlag), "JetBrains",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "User has license, should not fail")
}
