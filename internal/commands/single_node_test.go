package commands

import (
	"bytes"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestRunSingleNodeInstallAndRunCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "install",
		toFlag(configFileFlag), "./payloads/nonsense.json",
		toFlag(logFileFlag), fmt.Sprintf("%s.ast.log", t.Name()))
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeInstallAndRunCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "install",
		toFlag(configFileFlag), "./payloads/config.yml",
		toFlag(logFileFlag), fmt.Sprintf("%s.ast.log", t.Name()))
	assert.NilError(t, err)
}

func TestRunSingleNodeStartCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "start",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeStartCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "start",
		toFlag(configFileFlag), "./payloads/config.yml")
	assert.NilError(t, err)
}

func TestRunSingleNodeStopCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "stop", "./payloads/nonsense.json")
	assert.NilError(t, err)
}

func TestRunSingleNodeStopCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "stop")
	assert.NilError(t, err)
}

func TestRunSingleNodeRestartCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "restart",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeRestartCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "restart",
		toFlag(configFileFlag), "./payloads/config.yml")
	assert.NilError(t, err)
}

func TestRunBashCommand(t *testing.T) {
	scriptsWrapper := createTestScriptWrapper()
	logMaxSize := fmt.Sprintf("log_rotation_size=%s", "test_log_rotation_size")
	logAgeDays := fmt.Sprintf("log_rotation_age_days=%s", "test_log_rotation_age_days")
	privateKeyPath := fmt.Sprintf("private_key_path=%s", "test_private_key_path")
	certificateFile := fmt.Sprintf("certificate_path=%s", "test_certificate_path")
	deployDB := fmt.Sprintf("deploy_DB=%s", "1")
	installCmdStdOutputBuffer := bytes.NewBufferString("")
	installCmdStdErrorBuffer := bytes.NewBufferString("")
	upCmdStdOutputBuffer := bytes.NewBufferString("")
	upCmdStdErrorBuffer := bytes.NewBufferString("")
	err := runBashCommand(scriptsWrapper.GetInstallScriptPath(), installCmdStdOutputBuffer, installCmdStdErrorBuffer)
	assert.NilError(t, err, "install command should succeed")
	err = runBashCommand(scriptsWrapper.GetUpScriptPath(), upCmdStdOutputBuffer, upCmdStdErrorBuffer,
		logMaxSize, logAgeDays, privateKeyPath, certificateFile, deployDB)
	assert.NilError(t, err, "up script should succeed")
	fmt.Println("****UP COMMAND OUTPUT*******")
	actual := upCmdStdOutputBuffer.String()
	expected := fmt.Sprintf("test_log_rotation_size#test_log_rotation_age_days#" +
		"test_private_key_path#test_certificate_path#0\n")
	fmt.Println("EXPECTED:")
	fmt.Println(expected)
	fmt.Println("ACTUAL:")
	fmt.Println(actual)
	assert.Assert(t, expected == actual)
}
