//go:build integration

package integration

import (
	"fmt"
	"testing"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/viper"
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

func TestExecuteVorpalScan_InitializeAndRun_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(viper.GetInt(commonParams.VorpalPortKey))
	if healthCheckErr := vorpalWrapper.HealthCheck(); healthCheckErr == nil {
		_ = vorpalWrapper.ShutDown()
	} else {
		t.Failed()
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Sending empty source file should not fail")

	args = []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "data/python-vul-file.py",
		flag(commonParams.AgentFlag), commonParams.DefaultAgent,
	}

	err1, bytes := executeCommand(t, args...)
	fmt.Println(bytes.String())

	assert.NilError(t, err1, "Sending empty source file should not fail")

}
