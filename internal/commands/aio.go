package commands

import (
	"bytes"
	"fmt"
	"io"
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
		upCmdStdOutputBuffer := bytes.NewBufferString("")
		upCmdStdErrorBuffer := bytes.NewBufferString("")

		installScriptPath := scriptsWrapper.GetInstallScriptPath()
		installationStarted := "AIO installation started"
		writeToInstallationLog(installationStarted)
		writeToStandardOutput(installationStarted)
		writeToInstallationLog(fmt.Sprintf("Running installation script from path %s", installScriptPath))

		err = runBashCommand(installScriptPath, installCmdStdOutputBuffer, installCmdStdErrorBuffer)
		installationScriptOutput := installCmdStdOutputBuffer.String()
		writeToInstallationLogIfNotEmpty(installationScriptOutput)
		writeToStandardOutputIfNotEmpty(installationScriptOutput)

		if err != nil {
			msg := fmt.Sprintf("%s: Failed to run install script", failedInstallingAIO)
			logrus.WithFields(logrus.Fields{
				"err": err,
			}).Println(msg)

			writeToInstallationLogIfNotEmpty(installCmdStdErrorBuffer.String())
			return errors.Wrapf(err, msg)
		}

		// Run the up command after installation
		writeToStandardOutput("Trying to start AST...")
		err = runUpScript(cmd, scriptsWrapper, upCmdStdOutputBuffer, upCmdStdErrorBuffer)
		upScriptOutput := upCmdStdOutputBuffer.String()
		writeToInstallationLogIfNotEmpty(upScriptOutput)
		writeToStandardOutputIfNotEmpty(upScriptOutput)
		if err != nil {
			msg := fmt.Sprintf("%s: Failed to start AIO after installation", failedInstallingAIO)
			logrus.WithFields(logrus.Fields{
				"err": err,
			}).Println(msg)
			writeToInstallationLogIfNotEmpty(upCmdStdErrorBuffer.String())
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is up!")
		writeToInstallationLog("AIO installation completed successfully")
		writeToStandardOutput("AIO installation completed successfully")
		return nil
	}
}

func runStartAIOCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Run the up command after installation
		upCmdStdOutputBuffer := bytes.NewBufferString("")
		upCmdStdErrorBuffer := bytes.NewBufferString("")

		writeToStandardOutput("Trying to start AST...")
		err := runUpScript(cmd, scriptsWrapper, upCmdStdOutputBuffer, upCmdStdErrorBuffer)
		upScriptOutput := upCmdStdOutputBuffer.String()
		writeToInstallationLogIfNotEmpty(upScriptOutput)
		writeToStandardOutputIfNotEmpty(upScriptOutput)
		if err != nil {
			msg := fmt.Sprintf("Failed to start AST")
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is up!")
		return nil
	}
}

func runUpScript(cmd *cobra.Command, scriptsWrapper wrappers.ScriptsWrapper,
	upCmdStdOutputBuffer, upCmdStdErrorBuffer io.Writer) error {
	var err error
	upScriptPath := scriptsWrapper.GetUpScriptPath()
	configFile, _ := cmd.Flags().GetString(configFileFlag)
	configuration := config.AIOConfiguration{}

	if configFile != "" {
		var configInput []byte
		// Reading configuration from config file
		PrintIfVerbose(fmt.Sprintf("Reading configuration from file %s", configFile))
		configInput, err = ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to open config file", failedInstallingAIO)
		}

		err = yaml.Unmarshal(configInput, &configuration)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Unable to parse configuration file"))
		}

		err = mergeConfigurationWithEnv(&configuration, scriptsWrapper.GetDotEnvFilePath())
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("failed to merge configuration file with env file"))
		}
	}

	logMaxSize := fmt.Sprintf("log_rotation_size=%s", configuration.Log.Rotation.MaxSizeMB)
	logAgeDays := fmt.Sprintf("log_rotation_age_days=%s", configuration.Log.Rotation.MaxAgeDays)
	privateKeyFile := fmt.Sprintf("tls_private_key_file=%s", configuration.Network.PrivateKeyFile)
	certificateFile := fmt.Sprintf("tls_certificate_file=%s", configuration.Network.CertificateFile)

	err = runBashCommand(upScriptPath, upCmdStdOutputBuffer, upCmdStdErrorBuffer,
		logMaxSize, logAgeDays, privateKeyFile, certificateFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to run up script")
	}
	return nil
}

func writeToInstallationLog(msg string) {
	logrus.Println(msg)
}

func writeToInstallationLogIfNotEmpty(msg string) {
	if msg != "" {
		writeToInstallationLog(msg)
	}
}

func writeToStandardOutput(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}

func writeToStandardOutputIfNotEmpty(msg string) {
	if msg != "" {
		writeToStandardOutput(msg)
	}
}

func closeLogFile(logFile *os.File) func() {
	return func() {
		if logFile != nil {
			logFile.Close()
		}
	}
}
