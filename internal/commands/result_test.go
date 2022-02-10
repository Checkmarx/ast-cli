//go:build !integration

package commands

import (
	"fmt"
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"gotest.tools/assert"
)

const fileName = "cx_result"

func TestResultHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results")
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")

	// Remove generated sarif file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatSarif))
}

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")

	// Remove generated sonar file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatSonar))
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatHTML))
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdWrongFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "invalidFormat")
	assert.Equal(t, err.Error(), "bad report format invalidFormat", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithWrongFilterFormat(t *testing.T) {
	_ = execCmdNotNilAssertion(
		t,
		"results",
		"show",
		"--scan-id",
		"MOCK",
		"--report-format",
		"sarif",
		"--filter",
		"limit40",
	)
}

func TestRunGetResultsByScanIdWithMissingOrEmptyScanId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")

	err = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithEmptyOutputPath(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--output-path", "")
}

func TestResultBflHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results bfl")
}

func TestRunGetBflWithMissingOrEmptyScanIdAndQueryId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\", \"scan-id\" not set")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\" not set")
}

func TestRunGetBflWithMultipleScanIdsAndQueryIds(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1,MOCK2", "--query-id", "MOCK")
	assert.Equal(t, err.Error(), "Multiple scan-ids are not allowed.")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1", "--query-id", "MOCK1,MOCK2")
	assert.Equal(t, err.Error(), "Multiple query-ids are not allowed.")
}

func TestRunGetBFLByScanIdAndQueryId(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIdAndQueryIdWithFormatJson(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK", "--format", "JSON")
	assert.NilError(t, err)
}