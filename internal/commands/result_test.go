// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestResultHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "result")
	assert.NilError(t, err)
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "sarif")
	assert.Assert(t, err != nil)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "json")
	assert.Assert(t, err != nil)
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "summaryHTML")
	assert.Assert(t, err != nil)
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "result", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.Assert(t, err != nil)
}
