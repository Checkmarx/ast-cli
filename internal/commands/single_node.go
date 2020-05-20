package commands

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/sirupsen/logrus"

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

func NewSingleNodeCommand() *cobra.Command {
	singleNodeCmd := &cobra.Command{
		Use:   "single-node",
		Short: "Single Node AST",
	}
	installSingleNodeCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Single Node AST",
		RunE:  runInstallSingleNodeCommand(),
	}
	upSingleNodeCmd := &cobra.Command{
		Use:   "up",
		Short: "Start AST",
		RunE:  runUpSingleNodeCommand(),
	}
	downSingleNodeCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop AST",
		RunE:  runDownSingleNodeCommand(),
	}
	restartSingleNodeCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart AST",
		RunE:  runRestartSingleNodeCommand(),
	}

	updateSingleNodeCmd := &cobra.Command{
		Use:   "update",
		Short: "Update AST",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
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
	upSingleNodeCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path for AST (optional)")
	upSingleNodeCmd.PersistentFlags().String(astInstallationFolder, "./",
		"AST installation folder path")
	restartSingleNodeCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path for AST (optional)")
	singleNodeCmd.AddCommand(installSingleNodeCmd,
		upSingleNodeCmd,
		downSingleNodeCmd,
		restartSingleNodeCmd,
		healthSingleNodeCmd,
		updateSingleNodeCmd)
	return singleNodeCmd
}

func runInstallSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
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

		installScriptPath := getScriptPathRelativeToInstallation("install.sh", cmd)
		installationStarted := "Single node installation started"
		writeToInstallationLog(installationStarted)
		writeToStandardOutput(installationStarted)
		writeToInstallationLog(fmt.Sprintf("Running installation script from path %s", installScriptPath))

		err = runBashCommand(installScriptPath, installCmdStdOutputBuffer, installCmdStdErrorBuffer, []string{})
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
		successfully := "Single node installation completed successfully"
		writeToInstallationLog(successfully)
		writeToStandardOutput(successfully)
		return nil
	}
}

func runUpSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		upCmdStdOutputBuffer := bytes.NewBufferString("")
		upCmdStdErrorBuffer := bytes.NewBufferString("")

		writeToStandardOutput("Trying to start AST...")
		err := runUpScript(cmd, upCmdStdOutputBuffer, upCmdStdErrorBuffer)
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

func runDownSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		downCmdStdOutputBuffer := bytes.NewBufferString("")
		downCmdStdErrorBuffer := bytes.NewBufferString("")

		writeToStandardOutput("Trying to stop AST...")
		err := runDownScript(cmd, downCmdStdOutputBuffer, downCmdStdErrorBuffer)
		downScriptOutput := downCmdStdOutputBuffer.String()
		writeToStandardOutputIfNotEmpty(downScriptOutput)
		if err != nil {
			msg := fmt.Sprintf("Failed to stop AST")
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is down!")
		return nil
	}
}
func runRestartSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Trying to stop AST...")
		err := runDownSingleNodeCommand()(cmd, args)
		if err != nil {
			return err
		}
		err = runUpSingleNodeCommand()(cmd, args)
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

func runUpScript(cmd *cobra.Command, upCmdStdOutputBuffer, upCmdStdErrorBuffer io.Writer) error {
	var err error
	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)
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
	}

	installationFolder, _ := cmd.Flags().GetString(astInstallationFolder)
	envVars := getEnvVarsForCommand(&configuration, installationFolder)
	err = runBashCommand(upScriptPath, upCmdStdOutputBuffer, upCmdStdErrorBuffer, envVars)

	if err != nil {
		return errors.Wrapf(err, "Failed to run up script")
	}
	return nil
}

func runDownScript(cmd *cobra.Command, downCmdStdOutputBuffer, downCmdStdErrorBuffer io.Writer) error {
	var err error
	downScriptPath := getScriptPathRelativeToInstallation("down.sh", cmd)

	err = runBashCommand(downScriptPath, downCmdStdOutputBuffer, downCmdStdErrorBuffer, []string{})
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

func getPathRelativeToInstallation(filePath string, cmd *cobra.Command) string {
	installationFolder, _ := cmd.Flags().GetString(astInstallationFolder)
	return path.Join(installationFolder, filePath)
}

func getScriptPathRelativeToInstallation(scriptFile string, cmd *cobra.Command) string {
	scriptsDir := ".scripts"
	return getPathRelativeToInstallation(path.Join(scriptsDir, scriptFile), cmd)
}
