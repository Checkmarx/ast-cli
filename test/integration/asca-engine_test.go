//go:build integration

package integration

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/asca/ascaconfig"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/viper"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestScanASCA_NoFileSourceSent_ReturnSuccess(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.ASCALatestVersion),
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteASCAScan_ASCALatestVersionSetTrue_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.ASCALatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteASCAScan_NoSourceAndASCALatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())
	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_NotExistingFile_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_ASCALatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_NoEngineInstalledAndASCALatestVersionSetFalse_Success(t *testing.T) {
	configuration.LoadConfiguration()

	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())

	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_CorrectFlagsSent_SuccessfullyReturnMockData(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_UnsupportedLanguage_Fail(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
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

func TestExecuteASCAScan_InitializeAndRunUpdateVersion_Success(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.ASCALatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}
	ASCAWrapper = grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	healthCheckErr := ASCAWrapper.HealthCheck()
	asserts.NotNil(t, healthCheckErr)
	err, bytes := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")
	var scanResults grpcs.ScanResult
	err = json.Unmarshal(bytes.Bytes(), &scanResults)
	assert.NilError(t, err, "Failed to unmarshal scan result")
	assert.Assert(t, scanResults.Message == services.FilePathNotProvided, "should return message: ", services.FilePathNotProvided)
}

func TestExecuteASCAScan_InitializeAndShutdown_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "asca",
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

	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	if healthCheckErr := ASCAWrapper.HealthCheck(); healthCheckErr != nil {
		assert.Assert(t, healthCheckErr == nil, "Health check failed with error: ", healthCheckErr)
	}
	if shutdownErr := ASCAWrapper.ShutDown(); shutdownErr != nil {
		assert.Assert(t, shutdownErr == nil, "Shutdown failed with error: ", shutdownErr)
	}
	err = ASCAWrapper.HealthCheck()
	asserts.NotNil(t, err)
}

func TestExecuteASCAScan_EngineNotRunningWithLicense_Success(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.DebugFlag),
		flag(commonParams.AgentFlag), "JetBrains",
	}
	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "User has license, should not fail")
}

func TestExecuteASCAScan_Asca_location_Flag_Success(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())
	tempDir := t.TempDir()
	// Download ZIP
	resp, err := http.Get(ascaconfig.Params.DownloadURL)
	asserts.Nil(t, err)
	defer resp.Body.Close()
	asserts.Equal(t, http.StatusOK, resp.StatusCode)

	// Save ZIP file
	zipPath := filepath.Join(tempDir, ascaconfig.Params.FileName)
	file, err := os.Open(zipPath)
	asserts.Nil(t, err)
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	asserts.Nil(t, err)
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var extractedExePath string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		asserts.Nil(t, err)

		if header.Typeflag == tar.TypeReg &&
			filepath.Base(header.Name) == ascaconfig.Params.ExecutableFile {

			extractedExePath = filepath.Join(tempDir, filepath.Base(header.Name))

			outFile, err := os.Create(extractedExePath)
			asserts.Nil(t, err)

			_, err = io.Copy(outFile, tarReader)
			asserts.Nil(t, err)

			outFile.Close()
		}
	}
	asserts.NotEmpty(t, extractedExePath, "Executable not found inside zip")
	// Ensure executable permissions
	err = os.Chmod(extractedExePath, 0755)
	asserts.Nil(t, err)

	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.DebugFlag),
		flag(commonParams.AgentFlag), "JetBrains",
		flag(commonParams.ASCALocationFlag), tempDir,
	}

	err, _ = executeCommand(t, args...)
	assert.NilError(t, err, "should not fail")
	ASCAWrapper = grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()

	time.Sleep(500 * time.Millisecond)
}

func TestExecuteASCAScan_Asca_location_Flag_ThrowError_No_Executable(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())

	tempDir := t.TempDir()
	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.DebugFlag),
		flag(commonParams.AgentFlag), "JetBrains",
		flag(commonParams.ASCALocationFlag), tempDir,
	}

	err, _ := executeCommand(t, args...)
	asserts.NotNil(t, err, " Expected error due to missing executable in custom location")
	asserts.ErrorContains(t, err, "No ASCA executable found in provided location")
}

func TestExecuteASCAScan_Asca_location_Flag_ThrowError_InvalidPath(t *testing.T) {
	configuration.LoadConfiguration()
	ASCAWrapper := grpcs.NewASCAGrpcWrapper(viper.GetInt(commonParams.ASCAPortKey))
	_ = ASCAWrapper.ShutDown()
	_ = os.RemoveAll(ascaconfig.Params.WorkingDir())

	args := []string{
		"scan", "asca",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.DebugFlag),
		flag(commonParams.AgentFlag), "JetBrains",
		flag(commonParams.ASCALocationFlag), "/definitely/invalid/path",
	}
	err, _ := executeCommand(t, args...)
	asserts.ErrorContains(t, err, "Failed to validate ASCA custom location")
}
