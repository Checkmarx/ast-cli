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
		toFlag(astInstallationFolder), "./non_existing_folder",
		toFlag(configFileFlag), "./config_test.yml")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandWithNoFileNoFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationFolder), "./non_existing_folder")
	assert.Assert(t, err != nil)
}

func TestRunSingleNodeUpCommandWithFileWithFolder(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "single-node", "up",
		toFlag(astInstallationFolder), "./")
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
		Execution: config.Execution{
			Type: "TEST_Type",
		},
		Database: config.Database{
			Host:     "TEST_Host",
			Port:     "TEST_Port",
			Name:     "TEST_Name",
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
		ObjectStore: config.ObjectStore{
			AccessKeyID:     "TEST_AccessKeyID",
			SecretAccessKey: "TEST_SecretAccessKey",
		},
		MessageQueue: config.MessageQueue{
			Username: "TEST_Username",
			Password: "TEST_Password",
		},
		AccessControl: config.AccessControl{
			Username: "TEST_Username",
			Password: "TEST_Password",
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

	installScriptPath := getScriptPathRelativeToInstallation("install.sh", cmd)
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
		"DATABASE_DB=%s,"+
		"TRAEFIK_PORT=%s,"+
		"TRAEFIK_SSL_PORT=%s,"+
		"TLS_PRIVATE_KEY_PATH=%s,"+
		"TLS_CERTIFICATE_PATH=%s,"+
		"FQDN=%s,"+
		"OBJECT_STORE_ACCESS_KEY_ID=%s,"+
		"OBJECT_STORE_SECRET_ACCESS_KEY=%s,"+
		"NATS_USERNAME=%s,"+
		"NATS_PASSWORD=%s,"+
		"KEYCLOAK_USER=%s,"+
		"KEYCLOAK_PASSWORD=%s,"+
		"LOG_LEVEL=%s,"+
		"LOG_ROTATION_AGE_DAYS=%s,"+
		"LOG_ROTATION_MAX_SIZE_MB=%s,"+
		"EXTERNAL_ACCESS_IP=%s,"+
		"EXECUTION_TYPE=%s\n",
		installationFolder,
		testConfig.Database.Host,
		testConfig.Database.Port,
		testConfig.Database.Username,
		testConfig.Database.Password,
		testConfig.Database.Name,

		testConfig.Network.EntrypointPort,
		testConfig.Network.EntrypointTLSPort,
		testConfig.Network.TLS.PrivateKeyPath,
		testConfig.Network.TLS.CertificatePath,
		testConfig.Network.FullyQualifiedDomainName,

		testConfig.ObjectStore.AccessKeyID,
		testConfig.ObjectStore.SecretAccessKey,

		testConfig.MessageQueue.Username,
		testConfig.MessageQueue.Password,

		testConfig.AccessControl.Username,
		testConfig.AccessControl.Password,

		testConfig.Log.Level,
		testConfig.Log.Rotation.MaxAgeDays,
		testConfig.Log.Rotation.MaxSizeMB,
		testConfig.Network.ExternalAccessIP,
		testConfig.Execution.Type)
	fmt.Println("EXPECTED FROM UP SCRIPT OUTPUT:")
	fmt.Println(expected)
	fmt.Println("ACTUAL FROM UP SCRIPT OUTPUT:")
	fmt.Println(actual)
	assert.Assert(t, expected == actual)
}
