package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	"gotest.tools/assert"
)

func TestSSIHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "sast-metadata")
	assert.NilError(t, err)
}

func TestSSINoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-metadata")
	assert.NilError(t, err)
}

func TestRunDownloadEngineLog(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "sast-metadata", "engine-log", "1")
	assert.NilError(t, err)
	wd, _ := os.Getwd()
	mockFilePath := filepath.Join(wd, EngineLogDestFile)
	mockLogFile, err := os.Open(mockFilePath)
	assert.NilError(t, err, "failed to open log mock file")
	defer os.Remove(mockFilePath)
	defer mockLogFile.Close()
	bytes, err := ioutil.ReadAll(mockLogFile)
	assert.NilError(t, err, "failed to read log mock file")
	assert.Assert(t, string(bytes) == wrappers.MockContent)
}

func TestRunDownloadEngineLogNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-metadata", "engine-log")
	assert.Assert(t, err != nil)
}
