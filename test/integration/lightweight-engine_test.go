//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestScanVorpal_NoFileSourceSent_FailCommandWithError(t *testing.T) {
	//bindKeysToEnvAndDefault(t)
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.VorpalLatestVersion),
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.FileSourceFlagIsRequired)
}

func TestScanVorpal_SentFileWithoutExtension_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"scan", "vorpal",
		flag(commonParams.SourcesFlag), "my-file",
		flag(commonParams.VorpalLatestVersion),
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.FileExtensionIsRequired)
}

func TestExecuteVorpalScan_CorrectFlagsSent_SuccessfullyReturnMockData(t *testing.T) {

	scanResult, _ := commands.ExecuteVorpalScan("source.cs", true)
	expectedMockResult := commands.ReturnSuccessfulResponseMock()

	assert.DeepEqual(t, scanResult, expectedMockResult)
}
