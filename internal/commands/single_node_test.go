package commands

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/config"

	"gotest.tools/assert"
)

func TestRunSingleNodeInstallAndRunCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "install",
		toFlag(configFileFlag), "./payloads/nonsense.json",
		toFlag(logFileFlag), fmt.Sprintf("%s.ast.log", t.Name()))
	assert.NilError(t, err)
}

func TestRunSingleNodeInstallAndRunCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "install",
		toFlag(configFileFlag), "./config_test.yml",
		toFlag(logFileFlag), fmt.Sprintf("%s.ast.log", t.Name()))
	assert.NilError(t, err)
}

func TestRunSingleNodeUpCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
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

func TestRunSingleNodeRestartCommandWithFileNotFound(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "restart",
		toFlag(configFileFlag), "./payloads/nonsense.json")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeRestartCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "restart",
		toFlag(configFileFlag), "./config_test.yml")
	assert.NilError(t, err)
}

func TestRunBashCommand(t *testing.T) {
	testConfig := config.SingleNodeConfiguration{
		Database: config.Database{
			Host:     "TEST_Host",
			Port:     "TEST_Port",
			Instance: "TEST_Instance",
			Username: "TEST_Username",
			Password: "TEST_Password",
		},
		Network: config.Network{
			EntrypointPort:           "TEST_EntrypointPort",
			EntrypointTLSPort:        "TEST_EntrypointTLSPort",
			FullyQualifiedDomainName: "TEST_FullyQualifiedDomainName",
			ExternalAccessIP:         "TEST_ExternalAccessIP",
			TLS: config.TLS{
				PrivateKeyPath:  "TEST_PrivateKeyPath",
				CertificatePath: "TEST_CertificatePath",
			},
		},
		Log: config.Log{
			Level: "TEST_Level",
			Rotation: config.LogRotation{
				MaxSizeMB:  "TEST_MaxSizeMB",
				MaxAgeDays: "TEST_MaxAgeDays",
			},
		},
	}
	cmd := createASTTestCommand()
	installCmdStdOutputBuffer := bytes.NewBufferString("")
	installCmdStdErrorBuffer := bytes.NewBufferString("")
	upCmdStdOutputBuffer := bytes.NewBufferString("")
	upCmdStdErrorBuffer := bytes.NewBufferString("")
	// TODO add down test

	installScriptPath := getScriptPathRelativeToInstallation("docker-install.sh", cmd)
	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)
	err := runBashCommand(installScriptPath, installCmdStdOutputBuffer, installCmdStdErrorBuffer, []string{})
	assert.NilError(t, err, "install command should succeed")

	installationFolder := "AST_TEST_INSTALLATION_FOLDER"
	envs := getEnvVarsForCommand(&testConfig, installationFolder)
	err = runBashCommand(upScriptPath, upCmdStdOutputBuffer, upCmdStdErrorBuffer, envs)
	assert.NilError(t, err, "up script should succeed")
	actual := upCmdStdOutputBuffer.String()
	expected := fmt.Sprintf("AST_INSTALLATION_PATH=%s,"+
		"DATABASE_HOST=%s,"+
		"DATABASE_PORT=%s,"+
		"DATABASE_USER=%s,"+
		"DATABASE_PASSWORD=%s,"+
		"DATABASE_INSTANCE=%s,"+
		"ENTRYPOINT_PORT=%s,"+
		"ENTRYPOINT_TLS_PORT=%s,"+
		"TLS_PRIVATE_KEY_PATH=%s,"+
		"TLS_CERTIFICATE_PATH=%s,"+
		"FQDN=%s,"+
		"LOG_LEVEL=%s,"+
		"LOG_ROTATION_AGE_DAYS=%s,"+
		"LOG_ROTATION_MAX_SIZE_MB=%s,"+
		"EXTERNAL_ACCESS_IP=%s\n",
		installationFolder,
		testConfig.Database.Host,
		testConfig.Database.Port,
		testConfig.Database.Username,
		testConfig.Database.Password,
		testConfig.Database.Instance,

		testConfig.Network.EntrypointPort,
		testConfig.Network.EntrypointTLSPort,
		testConfig.Network.TLS.PrivateKeyPath,
		testConfig.Network.TLS.CertificatePath,
		testConfig.Network.FullyQualifiedDomainName,

		testConfig.Log.Level,
		testConfig.Log.Rotation.MaxAgeDays,
		testConfig.Log.Rotation.MaxSizeMB,
		testConfig.Network.ExternalAccessIP)
	fmt.Println("EXPECTED FROM UP SCRIPT OUTPUT:")
	fmt.Println(expected)
	fmt.Println("ACTUAL FROM UP SCRIPT OUTPUT:")
	fmt.Println(actual)
	assert.Assert(t, expected == actual)
}
