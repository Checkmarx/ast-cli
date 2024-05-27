//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestScanLightweight_NoFileSourceSent_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"scan", "lightweight",
		flag(commonParams.SourcesFlag), "",
		flag(commonParams.LightweightUpdateVersion), "1.0.0",
		flag(commonParams.FormatFlag), printer.FormatJSON,
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.FileSourceFlagIsRequired)
}

func TestScanLightweight_NoLightWeightUpdateVersionFlagSent_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"scan", "lightweight",
		flag(commonParams.SourcesFlag), "my-file.cs",
		flag(commonParams.LightweightUpdateVersion), "",
		flag(commonParams.FormatFlag), printer.FormatJSON,
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.LightweightUpdateVersionFlagIsRequired)
}

func TestScanLightweight_SentFileWithoutExtension_FailCommandWithError(t *testing.T) {
	bindKeysToEnvAndDefault(t)
	args := []string{
		"scan", "lightweight",
		flag(commonParams.SourcesFlag), "my-file",
		flag(commonParams.LightweightUpdateVersion), "1.0.0",
		flag(commonParams.FormatFlag), printer.FormatJSON,
	}

	err, _ := executeCommand(t, args...)
	assert.ErrorContains(t, err, errorConstants.FileExtensionIsRequired)
}

func TestExecuteLightweightScan_CorrectFlagsSent_SuccessfullyReturnMockData(t *testing.T) {

	scanResult, _ := commands.ExecuteLightweightScan("source.cs", "1.0.0")
	expectedMockResult := commands.ReturnSuccessfulResponseMock()

	assert.DeepEqual(t, scanResult, expectedMockResult)
}
