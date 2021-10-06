//go:build !integration
// +build !integration

package commands

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/assert"

	"github.com/checkmarxDev/ast-cli/internal/commands/util"
)

const fileName = "cx_result"

func TestResultHelp(t *testing.T) {
	executeCmdWithNilErrorAssertion(t, "help", "result")
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	executeCmdWithNilErrorAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "sarif")

	// Remove generated sarif file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatSarif))
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	executeCmdWithNilErrorAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	executeCmdWithNilErrorAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatHTML))
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	executeCmdWithNilErrorAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdWrongFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "invalidFormat")
	assert.Assert(t, err != nil, "An error should have been thrown since provided report format doesn't exist")
	assert.Equal(t, err.Error(), "bad report format invalidFormat", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithWrongFilterFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "sarif", "--filter", "limit40")
	assert.Assert(t, err != nil, "An error should have been thrown since the filter isn't with the proper format")
	assert.Equal(t, err.Error(), "Failed listing results: Invalid filters. Filters should be in a KEY=VALUE format", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithMissingOrEmptyScanId(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result")
	assert.Assert(t, err != nil, "An error should have been thrown since scan ID flag is missing")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")

	err = executeTestCommand(cmd, "result", "--scan-id", "")
	assert.Assert(t, err != nil, "An error should have been thrown since scan ID value is empty")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithEmptyOutputPath(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--output-path", "")
	assert.Assert(t, err != nil, "An error should have been thrown since output path is empty")
}
