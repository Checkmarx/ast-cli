//go:build !integration

package commands

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/assert"

	"github.com/checkmarx/ast-cli/internal/commands/util"
)

const fileName = "cx_result"

func TestResultHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "result")
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	execCmdNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "sarif")

	// Remove generated sarif file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatSarif))
}

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "sonar")

	// Remove generated sarif file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatSarif))
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatJSON))
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	execCmdNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	os.Remove(fmt.Sprintf("%s.%s", fileName, util.FormatHTML))
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	execCmdNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdWrongFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "invalidFormat")
	assert.Equal(t, err.Error(), "bad report format invalidFormat", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithWrongFilterFormat(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "result", "--scan-id", "MOCK", "--report-format", "sarif", "--filter", "limit40")
}

func TestRunGetResultsByScanIdWithMissingOrEmptyScanId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "result")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")

	err = execCmdNotNilAssertion(t, "result", "--scan-id", "")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithEmptyOutputPath(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "result", "--scan-id", "MOCK", "--output-path", "")
}
