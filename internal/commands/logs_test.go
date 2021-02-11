// +build !integration

package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
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
	bytes, err := ioutil.ReadAll(mocklogsFile)
	assert.NilError(t, err, "failed to read logs mock file")
	assert.Assert(t, string(bytes) == wrappers.MockContent)
}
