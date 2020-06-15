// +build !integration

package commands

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/config"

	"gotest.tools/assert"
)

func TestRunSingleNodeUpCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up", "-h")
	assert.NilError(t, err)
}

func TestRunSingleNodeUpCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(configFileFlag), "./config_test.yml")
	assert.NilError(t, err)
}

func TestRunSingleNodeUpCommandWithFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./non_existing_folder",
		toFlag(configFileFlag), "./config_test.yml")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeDownCommandWithFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "down",
		toFlag(astInstallationDir), "./non_existing_folder",
		toFlag(configFileFlag), "./config_test.yml")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandWithNoFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./non_existing_folder")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandWithDefaultConfiguration(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./")
	assert.NilError(t, err)
}

func TestRunSingleNodeUpCommandWithFileWithFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./")
	assert.NilError(t, err)
}

func TestRunSingleNodeUpdateCommandWithNoFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "update",
		toFlag(astInstallationDir), "./non_existing_folder")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpdateCommandWithFileWithFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "update",
		toFlag(astInstallationDir), "./")
	assert.NilError(t, err)
}

func TestRunSingleNodeDownCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "down", toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
	assert.Assert(t, err.Error() == "unknown flag: --config")
}

func TestRunSingleNodeDownCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "down")
	assert.NilError(t, err)
}

func TestRunSingleNodeUpdateCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "update",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpdateCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "update",
		toFlag(configFileFlag), "./config_test.yml")
	assert.NilError(t, err)
}

const (
	testhost               = "TEST_Host"
	testport               = "TEST_Port"
	testinstance           = "TEST_Instance"
	testusername           = "TEST_Username"
	testpassword           = "TEST_Password"
	testentryPoint         = "TEST_EntrypointPort"
	testexternalHost       = "TEST_ExternalHostname"
	testprivateKeyPath     = "TEST_PrivateKeyPath"
	testcertificatePath    = "TEST_CertificatePath"
	testlevel              = "TEST_Level"
	testlocation           = "TEST_Location"
	testrotationMaxSizeMB  = "TEST_MaxSizeMB"
	testformat             = "TEST_Format"
	testrotationCount      = "TEST_Count"
	testinstallationFolder = "AST_TEST_INSTALLATION_FOLDER"
	testrole               = "AST_TEST_ROLE"
	expected               = "AST_INSTALLATION_PATH=%s," +
		"AST_ROLE=%s," +
		"DATABASE_HOST=%s," +
		"DATABASE_PORT=%s," +
		"DATABASE_USER=%s," +
		"DATABASE_PASSWORD=%s," +
		"DATABASE_INSTANCE=%s," +
		"ENTRYPOINT_PORT=%s," +
		"TLS_PRIVATE_KEY_PATH=%s," +
		"TLS_CERTIFICATE_PATH=%s," +
		"LOG_LEVEL=%s," +
		"LOG_LOCATION=%s," +
		"LOG_ROTATION_COUNT=%s," +
		"LOG_ROTATION_MAX_SIZE_MB=%s," +
		"LOG_FORMAT=%s," +
		"EXTERNAL_HOSTNAME=%s\n"
)

func TestRunBashCommand(t *testing.T) {
	fmt.Println("**************************Testing running bash command*************")

	testConfig := config.SingleNodeConfiguration{
		Database: config.Database{
			Host:     testhost,
			Port:     testport,
			Instance: testinstance,
			Username: testusername,
			Password: testpassword,
		},
		Network: config.Network{
			EntrypointPort:   testentryPoint,
			ExternalHostname: testexternalHost,
			TLS: config.TLS{
				PrivateKeyPath:  testprivateKeyPath,
				CertificatePath: testcertificatePath,
			},
		},
		Log: config.Log{
			Level:    testlevel,
			Location: testlocation,
			Format:   testformat,
			Rotation: config.LogRotation{
				MaxSizeMB: testrotationMaxSizeMB,
				Count:     testrotationCount,
			},
		},
	}
	cmd := createASTTestCommand()

	var actualOut *bytes.Buffer
	var err error

	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)

	envs := createEnvVarsForCommand(&testConfig, testinstallationFolder, testrole)

	actualOut, _, err = runBashCommand(upScriptPath, envs)
	assert.NilError(t, err, "up script should succeed")

	expected := fmt.Sprintf(
		expected,
		testinstallationFolder,
		testrole,
		testhost,
		testport,
		testusername,
		testpassword,
		testinstance,
		testentryPoint,
		testprivateKeyPath,
		testcertificatePath,
		testlevel,
		testlocation,
		testrotationCount,
		testrotationMaxSizeMB,
		testformat,
		testexternalHost)
	fmt.Println()
	fmt.Println("EXPECTED from UP script:")
	fmt.Println(expected)
	fmt.Println()
	fmt.Println("ACTUAL from UP script:")
	fmt.Println(actualOut.String())
	assert.Assert(t, expected == actualOut.String())

	downScriptPath := getScriptPathRelativeToInstallation("down.sh", cmd)

	actualOut, _, err = runBashCommand(downScriptPath, envs)
	assert.NilError(t, err, "down script should succeed")
	fmt.Println()
	fmt.Println("EXPECTED from DOWN script:")
	fmt.Println(expected)
	fmt.Println()
	fmt.Println("ACTUAL from DOWN script:")
	fmt.Println(actualOut.String())
	assert.Assert(t, expected == actualOut.String())
}

func TestRunBashCommandDefaultRole(t *testing.T) {
	fmt.Println("**************************Testing running bash command*************")

	testConfig := config.SingleNodeConfiguration{
		Database: config.Database{
			Host:     testhost,
			Port:     testport,
			Instance: testinstance,
			Username: testusername,
			Password: testpassword,
		},
		Network: config.Network{
			EntrypointPort:   testentryPoint,
			ExternalHostname: testexternalHost,
			TLS: config.TLS{
				PrivateKeyPath:  testprivateKeyPath,
				CertificatePath: testcertificatePath,
			},
		},
		Log: config.Log{
			Level:    testlevel,
			Location: testlocation,
			Format:   testformat,
			Rotation: config.LogRotation{
				MaxSizeMB: testrotationMaxSizeMB,
				Count:     testrotationCount,
			},
		},
	}
	cmd := createASTTestCommand()

	var actualOut *bytes.Buffer
	var err error

	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)

	envs := createEnvVarsForCommand(&testConfig, testinstallationFolder, params.ScaAgent)
	actualOut, _, err = runBashCommand(upScriptPath, envs)
	assert.NilError(t, err, "up script should succeed")

	expected := fmt.Sprintf(
		expected,
		testinstallationFolder,
		params.ScaAgent,
		testhost,
		testport,
		testusername,
		testpassword,
		testinstance,
		testentryPoint,
		testprivateKeyPath,
		testcertificatePath,
		testlevel,
		testlocation,
		testrotationCount,
		testrotationMaxSizeMB,
		testformat,
		testexternalHost)
	fmt.Println()
	fmt.Println("EXPECTED from UP script:")
	fmt.Println(expected)
	fmt.Println()
	fmt.Println("ACTUAL from UP script:")
	fmt.Println(actualOut.String())
	assert.Assert(t, expected == actualOut.String())

	downScriptPath := getScriptPathRelativeToInstallation("down.sh", cmd)

	actualOut, _, err = runBashCommand(downScriptPath, envs)
	assert.NilError(t, err, "down script should succeed")
	fmt.Println()
	fmt.Println("EXPECTED from DOWN script:")
	fmt.Println(expected)
	fmt.Println()
	fmt.Println("ACTUAL from DOWN script:")
	fmt.Println(actualOut.String())
	assert.Assert(t, expected == actualOut.String())
}
