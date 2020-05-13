package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	"gopkg.in/yaml.v2"

	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	logFileFlag         = "log"
	configFileFlag      = "config"
	failedInstallingAIO = "Failed installing All-In-One"
	failedRunningAIO    = "Failed running All-In-One"
)

func NewAIOCommand(scriptsWrapper wrappers.ScriptsWrapper) *cobra.Command {
	aioCmd := &cobra.Command{
		Use:   "aio",
		Short: "All-In-One AST",
	}

	installAIOCmd := &cobra.Command{
		Use:   "install",
		Short: "Install All-In-One AST",
		RunE:  runInstallAIOCommand(scriptsWrapper),
	}

	startAIOCmd := &cobra.Command{
		Use:   "start",
		Short: "Start All-In-One AST",
		RunE:  runStartAIOCommand(scriptsWrapper),
	}

	installAIOCmd.PersistentFlags().String(logFileFlag, "./install.ast.log",
		"Installation log file path (optional)")
	installAIOCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path to provide to the AST installation (optional)")
	startAIOCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path for AST (optional)")

	aioCmd.AddCommand(installAIOCmd, startAIOCmd)
	return aioCmd
}

func runInstallAIOCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		logFilePath, _ := cmd.Flags().GetString(logFileFlag)
		PrintIfVerbose(fmt.Sprintf("Log file path: %s", logFilePath))
		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to open installation log file", failedInstallingAIO)
		}
		logrus.SetOutput(logFile)
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableQuote: true,
		})
		logrus.RegisterExitHandler(closeLogFile(logFile))

		installCmdStdOutputBuffer := bytes.NewBufferString("")
		installCmdStdErrorBuffer := bytes.NewBufferString("")
		var upCmdOutput []byte

		installScriptPath := scriptsWrapper.GetInstallScriptPath()
		writeToInstallationLog("AIO installation started")
		fmt.Println("AIO installation started")
		writeToInstallationLog("Running installation script from path", installScriptPath)
		err = runBashCommand(installScriptPath, installCmdStdOutputBuffer, installCmdStdErrorBuffer)
		if err != nil {
			msg := fmt.Sprintf("%s: Failed to run install command", failedInstallingAIO)
			logrus.WithFields(logrus.Fields{
				"err": err,
			}).Println(msg)
			writeToInstallationLog("OUTPUT FROM INSTALL COMMAND START")
			writeToInstallationLog(installCmdStdOutputBuffer.String())
			writeToInstallationLog("OUTPUT FROM INSTALL COMMAND END")
			writeToInstallationLog("ERROR FROM INSTALL COMMAND START")
			writeToInstallationLog(installCmdStdErrorBuffer.String())
			writeToInstallationLog(err.Error())
			writeToInstallationLog("ERROR FROM INSTALL COMMAND END")

			return errors.Wrapf(err, msg)
		}
		writeToInstallationLog(installCmdStdOutputBuffer.String())

		// Run the up command after installation
		upCmdOutput, err = up(cmd, scriptsWrapper)
		if err != nil {
			msg := fmt.Sprintf("%s: Failed to start AIO after installation", failedInstallingAIO)
			logrus.WithFields(logrus.Fields{
				"err": err,
			}).Println(msg)
			writeToInstallationLog(string(upCmdOutput))
			return errors.Wrapf(err, msg)
		}
		writeToInstallationLog(string(upCmdOutput))
		writeToInstallationLog("AIO installation completed successfully")
		fmt.Println("AIO installation completed successfully")
		return nil
	}
}

func runStartAIOCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Run the up command after installation
		upCmdOutput, err := up(cmd, scriptsWrapper)
		if err != nil {
			msg := fmt.Sprintf("Failed to start AST")
			fmt.Println(string(upCmdOutput))
			return errors.Wrapf(err, msg)
		}
		fmt.Println("AIO started successfully")
		return nil
	}
}

func up(cmd *cobra.Command, scriptsWrapper wrappers.ScriptsWrapper) ([]byte, error) {
	var cmdOutput []byte
	var err error
	writeToInstallationLog("Trying to start AST...")
	upScriptPath := scriptsWrapper.GetUpScriptPath()
	writeToInstallationLog("Running up script from path", upScriptPath)
	configFile, _ := cmd.Flags().GetString(configFileFlag)
	configuration := config.AIOConfiguration{}

	if configFile != "" {
		var configInput []byte
		// Reading configuration from config file
		PrintIfVerbose(fmt.Sprintf("Reading configuration from file %s", configFile))
		configInput, err = ioutil.ReadFile(configFile)
		if err != nil {
			return nil, errors.Wrapf(err, "%s: Failed to open config file", failedInstallingAIO)
		}

		err = yaml.Unmarshal(configInput, &configuration)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Unable to parse configuration file"))
		}

		err = mergeConfigurationWithEnv(&configuration, scriptsWrapper.GetDotEnvFilePath())
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("failed to merge configuration file with env file"))
		}
	}

	logMaxSize := fmt.Sprintf("log_rotation_size=%s", configuration.Log.Rotation.MaxSizeMB)
	logAgeDays := fmt.Sprintf("log_rotation_age_days=%s", configuration.Log.Rotation.MaxAgeDays)
	privateKeyFile := fmt.Sprintf("tls_private_key_file=%s", configuration.Network.PrivateKeyFile)
	certificateFile := fmt.Sprintf("tls_certificate_file=%s", configuration.Network.CertificateFile)
	upCmdStdOutputBuffer := bytes.NewBufferString("")
	upCmdStdErrorBuffer := bytes.NewBufferString("")
	err = runBashCommand(upScriptPath, upCmdStdOutputBuffer, upCmdStdErrorBuffer,
		logMaxSize, logAgeDays, privateKeyFile, certificateFile)
	if err != nil {
		return cmdOutput, errors.Wrapf(err, "Failed to run up command")
	}
	return cmdOutput, nil
}

func writeToInstallationLog(msg ...string) {
	logrus.Println(msg)
}

func closeLogFile(logFile *os.File) func() {
	return func() {
		if logFile != nil {
			logFile.Close()
		}
	}
}
