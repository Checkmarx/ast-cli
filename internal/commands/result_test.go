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
	err := executeTestCommand(cmd, "-v", "result", "--scan-id", "MOCK", "--report-format", "sarif")
	assert.NilError(t, err)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "result", "--scan-id", "MOCK", "--report-format", "json")
	assert.NilError(t, err)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "result", "--scan-id", "MOCK", "--report-format", "summaryHTML")
	assert.NilError(t, err)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "result", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)
}
