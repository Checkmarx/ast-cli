// +build !integration

package commands

import (
	"bytes"
	"fmt"
	"testing"

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

func TestRunSingleNodeUpCommandWithNoFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./non_existing_folder")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandWithFileWithFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationDir), "./")
	assert.NilError(t, err)
}

func TestRunSingleNodeDownCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "down", "./payloads/nonsense.json")
	assert.NilError(t, err)
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

func TestRunBashCommand(t *testing.T) {
	fmt.Println("**************************Testing running bash command*************")
	testConfig := config.SingleNodeConfiguration{
		Database: config.Database{
			Host:     "TEST_Host",
			Port:     "TEST_Port",
			Instance: "TEST_Instance",
			Username: "TEST_Username",
			Password: "TEST_Password",
		},
		Network: config.Network{
			EntrypointPort:   "TEST_EntrypointPort",
			ExternalHostname: "TEST_ExternalHostname",
			TLS: config.TLS{
				PrivateKeyPath:  "TEST_PrivateKeyPath",
				CertificatePath: "TEST_CertificatePath",
			},
		},
		Log: config.Log{
			Level:    "TEST_Level",
			Location: "TEST_Location",
			Rotation: config.LogRotation{
				MaxSizeMB: "TEST_MaxSizeMB",
				Count:     "TEST_Count",
			},
		},
	}
	cmd := createASTTestCommand()

	var actualOut *bytes.Buffer
	var err error

	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)

	installationFolder := "AST_TEST_INSTALLATION_FOLDER"
	role := "AST_TEST_ROLE"
	envs := createEnvVarsForCommand(&testConfig, installationFolder, role)

	actualOut, _, err = runBashCommand(upScriptPath, envs)
	assert.NilError(t, err, "up script should succeed")

	expected := fmt.Sprintf(
		"AST_INSTALLATION_PATH=%s,"+
			"AST_ROLE=%s,"+
			"DATABASE_HOST=%s,"+
			"DATABASE_PORT=%s,"+
			"DATABASE_USER=%s,"+
			"DATABASE_PASSWORD=%s,"+
			"DATABASE_INSTANCE=%s,"+
			"ENTRYPOINT_PORT=%s,"+
			"TLS_PRIVATE_KEY_PATH=%s,"+
			"TLS_CERTIFICATE_PATH=%s,"+
			"LOG_LEVEL=%s,"+
			"LOG_LOCATION=%s,"+
			"LOG_ROTATION_COUNT=%s,"+
			"LOG_ROTATION_MAX_SIZE_MB=%s,"+
			"EXTERNAL_HOSTNAME=%s\n",
		installationFolder,
		role,
		testConfig.Database.Host,
		testConfig.Database.Port,
		testConfig.Database.Username,
		testConfig.Database.Password,
		testConfig.Database.Instance,

		testConfig.Network.EntrypointPort,
		testConfig.Network.TLS.PrivateKeyPath,
		testConfig.Network.TLS.CertificatePath,

		testConfig.Log.Level,
		testConfig.Log.Location,
		testConfig.Log.Rotation.Count,
		testConfig.Log.Rotation.MaxSizeMB,
		testConfig.Network.ExternalHostname)
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
