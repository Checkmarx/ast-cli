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
	logFileFlag           = "log"
	configFileFlag        = "config"
	astInstallationFolder = "folder"
	failedInstallingAST   = "Failed installing AST"
)

var logrusFileLogger = logrus.New()

func NewSingleNodeCommand(scriptsWrapper wrappers.ScriptsWrapper) *cobra.Command {
	singleNodeCmd := &cobra.Command{
		Use:   "single-node",
		Short: "Single Node AST",
	}
	installSingleNodeCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Single Node AST",
		RunE:  runInstallSingleNodeCommand(scriptsWrapper),
	}
	startSingleNodeCmd := &cobra.Command{
		Use:   "start",
		Short: "Start AST",
		RunE:  runStartSingleNodeCommand(scriptsWrapper),
	}
	stopSingleNodeCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop AST",
		RunE:  runStopSingleNodeCommand(scriptsWrapper),
	}
	restartSingleNodeCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart AST",
		RunE:  runRestartSingleNodeCommand(scriptsWrapper),
	}
	healthSingleNodeCmd := &cobra.Command{
		Use:   "health",
		Short: "Show health information for AST",
		RunE:  runHealthSingleNodeCommand,
	}

	installSingleNodeCmd.PersistentFlags().String(logFileFlag, "./install.ast.log",
		"Installation log file path (optional)")
	installSingleNodeCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path to provide to the AST installation (optional)")
	startSingleNodeCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path for AST (optional)")
	startSingleNodeCmd.PersistentFlags().String(astInstallationFolder, "",
		"AST installation path")
	restartSingleNodeCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path for AST (optional)")
	singleNodeCmd.AddCommand(installSingleNodeCmd,
		startSingleNodeCmd,
		stopSingleNodeCmd,
		restartSingleNodeCmd,
		healthSingleNodeCmd)
	return singleNodeCmd
}

func runInstallSingleNodeCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		logFilePath, _ := cmd.Flags().GetString(logFileFlag)
		PrintIfVerbose(fmt.Sprintf("Log file path: %s", logFilePath))
		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to open installation log file", failedInstallingAST)
		}
		logrusFileLogger.SetOutput(logFile)
		logrusFileLogger.SetFormatter(&logrus.TextFormatter{
			DisableQuote: true,
		})

		installCmdStdOutputBuffer := bytes.NewBufferString("")
		installCmdStdErrorBuffer := bytes.NewBufferString("")
		upCmdStdOutputBuffer := bytes.NewBufferString("")
		upCmdStdErrorBuffer := bytes.NewBufferString("")

		installScriptPath := scriptsWrapper.GetInstallScriptPath()
		installationStarted := "Single node installation started"
		writeToInstallationLog(installationStarted)
		writeToStandardOutput(installationStarted)
		writeToInstallationLog(fmt.Sprintf("Running installation script from path %s", installScriptPath))

		err = runBashCommand(installScriptPath, installCmdStdOutputBuffer, installCmdStdErrorBuffer)
		installationScriptOutput := installCmdStdOutputBuffer.String()
		writeToInstallationLogIfNotEmpty(installationScriptOutput)
		writeToStandardOutputIfNotEmpty(installationScriptOutput)

		if err != nil {
			msg := fmt.Sprintf("%s: Failed to run install script", failedInstallingAST)
			logrusFileLogger.WithFields(logrus.Fields{
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
			msg := fmt.Sprintf("%s: Failed to start AST after installation", failedInstallingAST)
			logrusFileLogger.WithFields(logrus.Fields{
				"err": err,
			}).Println(msg)
			writeToInstallationLogIfNotEmpty(upCmdStdErrorBuffer.String())
			return errors.Wrapf(err, msg)
		}
		successfully := "Single node installation completed successfully"
		writeToStandardOutput("AST is up!")
		writeToInstallationLog(successfully)
		writeToStandardOutput(successfully)
		return nil
	}
}

func runStartSingleNodeCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		upCmdStdOutputBuffer := bytes.NewBufferString("")
		upCmdStdErrorBuffer := bytes.NewBufferString("")

		writeToStandardOutput("Trying to start AST...")
		err := runUpScript(cmd, scriptsWrapper, upCmdStdOutputBuffer, upCmdStdErrorBuffer)
		upScriptOutput := upCmdStdOutputBuffer.String()
		writeToStandardOutputIfNotEmpty(upScriptOutput)
		if err != nil {
			msg := fmt.Sprintf("Failed to start AST")
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is up!")
		return nil
	}
}

func runStopSingleNodeCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		downCmdStdOutputBuffer := bytes.NewBufferString("")
		downCmdStdErrorBuffer := bytes.NewBufferString("")

		writeToStandardOutput("Trying to stop AST...")
		err := runDownScript(scriptsWrapper, downCmdStdOutputBuffer, downCmdStdErrorBuffer)
		downScriptOutput := downCmdStdOutputBuffer.String()
		writeToInstallationLogIfNotEmpty(downScriptOutput)
		writeToStandardOutputIfNotEmpty(downScriptOutput)
		if err != nil {
			msg := fmt.Sprintf("Failed to stop AST")
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is down!")
		return nil
	}
}
func runRestartSingleNodeCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Trying to stop AST...")
		err := runStopSingleNodeCommand(scriptsWrapper)(cmd, args)
		if err != nil {
			return err
		}
		err = runStartSingleNodeCommand(scriptsWrapper)(cmd, args)
		if err != nil {
			return err
		}
		writeToStandardOutput("AST restarted successfully!")
		return nil
	}
}
func runHealthSingleNodeCommand(cmd *cobra.Command, args []string) error {
	return nil
}

func runUpScript(cmd *cobra.Command, scriptsWrapper wrappers.ScriptsWrapper,
	upCmdStdOutputBuffer, upCmdStdErrorBuffer io.Writer) error {
	var err error
	upScriptPath := scriptsWrapper.GetUpScriptPath()
	configFile, _ := cmd.Flags().GetString(configFileFlag)
	configuration := config.SingleNodeConfiguration{}

	if configFile != "" {
		var configInput []byte
		// Reading configuration from config file
		PrintIfVerbose(fmt.Sprintf("Reading configuration from file %s", configFile))
		configInput, err = ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "Failed to open config file")
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
	privateKeyPath := fmt.Sprintf("private_key_path=%s", configuration.Network.PrivateKeyPath)
	certificateFile := fmt.Sprintf("certificate_path=%s", configuration.Network.CertificatePath)
	fqdn := fmt.Sprintf("fqdn=%s", configuration.Network.FullyQualifiedDomainName)
	deployDB := fmt.Sprintf("deploy_DB=%t", configuration.Database.Host == "")
	deployTLS := fmt.Sprintf("deploy_TLS=%t", configuration.Network.CertificatePath != "")

	err = runBashCommand(upScriptPath, upCmdStdOutputBuffer, upCmdStdErrorBuffer,
		logMaxSize, logAgeDays, privateKeyPath, certificateFile, fqdn, deployTLS, deployDB)
	if err != nil {
		return errors.Wrapf(err, "Failed to run up script")
	}
	return nil
}

func runDownScript(scriptsWrapper wrappers.ScriptsWrapper,
	downCmdStdOutputBuffer, downCmdStdErrorBuffer io.Writer) error {
	var err error
	downScriptPath := scriptsWrapper.GetDownScriptPath()

	err = runBashCommand(downScriptPath, downCmdStdOutputBuffer, downCmdStdErrorBuffer)
	if err != nil {
		return errors.Wrapf(err, "Failed to run down script")
	}
	return nil
}

func writeToInstallationLog(msg string) {
	logrusFileLogger.Println(msg)
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
