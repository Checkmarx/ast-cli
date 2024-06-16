//go:build integration

package integration

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/vorpalengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
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
	scanResult, _ := commands.ExecuteVorpalScan(generateVorpalParams("data/python-vul-file.py", true, true))
	expectedMockResult := mock.ReturnSuccessfulResponseMock()
	//TODO: update mocks when there's a real engine
	assert.DeepEqual(t, scanResult, expectedMockResult)
}

func generateVorpalParams(filePath string, vorpalUpdateVersion, isDefaultagent bool) vorpalengine.VorpalScanParams {
	featureFlagsPath := viper.GetString(commonParams.FeatureFlagsKey)
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)
	jwtWrapper := wrappers.NewJwtWrapper()
	return vorpalengine.VorpalScanParams{
		FilePath:            filePath,
		VorpalUpdateVersion: vorpalUpdateVersion,
		IsDefaultAgent:      isDefaultagent,
		JwtWrapper:          jwtWrapper,
		FeatureFlagsWrapper: featureFlagsWrapper,
	}
}
