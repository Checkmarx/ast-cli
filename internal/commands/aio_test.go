package commands

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestRunAIOInstallAndRunCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "aio", "install", toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunAIOInstallAndRunCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "aio", "install", toFlag(configFileFlag), "./payloads/config.yml")
	assert.NilError(t, err)
}

func TestRunBashCommand(t *testing.T) {
	scriptsWrapper := createTestScriptWrapper()
	logMaxSize := fmt.Sprintf("log_rotation_size=%s", "test_log_rotation_size")
	logAgeDays := fmt.Sprintf("log_rotation_age_days=%s", "test_log_rotation_age_days")
	privateKeyFile := fmt.Sprintf("tls_private_key_file=%s", "test_tls_private_key_file")
	certificateFile := fmt.Sprintf("tls_certificate_file=%s", "test_tls_certificate_file")
	var upOutput []byte
	_, err := runBashCommand(scriptsWrapper.GetInstallScriptPath())
	assert.NilError(t, err, "install command should succeed")
	upOutput, err = runBashCommand(scriptsWrapper.GetUpScriptPath(),
		logMaxSize, logAgeDays, privateKeyFile, certificateFile)
	assert.NilError(t, err, "up command should succeed")
	fmt.Println("****UP COMMAND OUTPUT*******")
	actual := string(upOutput)
	expected := fmt.Sprintf("test_log_rotation_size#test_log_rotation_age_days#" +
		"test_tls_private_key_file#test_tls_certificate_file\n")
	fmt.Println("EXPECTED:")
	fmt.Println(expected)
	fmt.Println("ACTUAL:")
	fmt.Println(actual)
	assert.Assert(t, expected == actual)
}
