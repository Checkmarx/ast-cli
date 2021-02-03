// +build !integration

package commands

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	"gotest.tools/assert"
)

func TestSastMetadataHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "sast-metadata")
	assert.NilError(t, err)
}

func TestSastMetadataNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "sast-metadata")
	assert.NilError(t, err)
}

func TestRunDownloadEngineLog(t *testing.T) {
	cmd := createASTTestCommand()
	outBuff := bytes.NewBufferString("")
	cmd.SetOut(outBuff)
	err := executeTestCommand(cmd, "-v", "utils", "sast-metadata", "engine-log", "1")
	assert.NilError(t, err)
	b, err := ioutil.ReadAll(outBuff)
	assert.NilError(t, err)
	assert.Assert(t, string(b) == wrappers.MockContent)
}

func TestRunDownloadEngineLogNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "sast-metadata", "engine-log")
	assert.Assert(t, err != nil)
}

func TestRunScanInfo(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "sast-metadata", "scan-info", "1")
	assert.NilError(t, err)
}

func TestRunScanInfoNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "sast-metadata", "scan-info")
	assert.Assert(t, err != nil)
}

func TestRunMetrics(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "sast-metadata", "metrics", "1")
	assert.NilError(t, err)
}

func TestRunScanMetricsNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "sast-metadata", "metrics")
	assert.Assert(t, err != nil)
}
