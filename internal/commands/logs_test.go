// +build !integration

package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestLogsHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "logs")
	assert.NilError(t, err)
}

func TestLogsNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "logs")
	assert.NilError(t, err)
}

/**
func TestRunDownloadLogs(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "logs", "download")
	assert.NilError(t, err)
	wd, _ := os.Getwd()
	mockFilePath := filepath.Join(wd, LogsDestFileName)
	mocklogsFile, err := os.Open(mockFilePath)
	assert.NilError(t, err, "failed to open logs mock file")
	defer os.Remove(mockFilePath)
	defer mocklogsFile.Close()
	data, err := ioutil.ReadAll(mocklogsFile)
	assert.NilError(t, err, "failed to read logs mock file")
	assert.Assert(t, string(data) == wrappers.MockContent)
}

func TestRunDownloadEngineLog(t *testing.T) {
	cmd := createASTTestCommand()
	outBuff := bytes.NewBufferString("")
	cmd.SetOut(outBuff)
	err := executeTestCommand(cmd, "-v", "utils", "logs", "sast", "--scan-id", "1")
	assert.NilError(t, err)
	b, err := ioutil.ReadAll(outBuff)
	assert.NilError(t, err)
	assert.Assert(t, string(b) == wrappers.MockContent)
}

func TestRunDownloadEngineLogNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "logs", "sast")
	assert.Assert(t, err != nil)
}
*/
