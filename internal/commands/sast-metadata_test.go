package commands

import (
	"bytes"
	"io/ioutil"
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
	outBuff := bytes.NewBufferString("")
	cmd.SetOut(outBuff)
	err := executeTestCommand(cmd, "-v", "sast-metadata", "engine-log", "1")
	assert.NilError(t, err)
	b, err := ioutil.ReadAll(outBuff)
	assert.NilError(t, err)
	assert.Assert(t, string(b) == wrappers.MockContent)
}

func TestRunDownloadEngineLogNoParam(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "sast-metadata", "engine-log")
	assert.Assert(t, err != nil)
}
