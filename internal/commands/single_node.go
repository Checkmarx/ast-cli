package commands

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v2"

	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	logFileFlag         = "log"
	configFileFlag      = "config"
	astInstallationDir  = "installation-dir"
	failedInstallingAST = "Failed installing AST"
)

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
	installationConfigFileUsage := "Configuration file path for AST (optional)"
	installationFolderUsage := "AST installation folder path"
	installationFolderDefault := "./"

	installSingleNodeCmd.PersistentFlags().String(logFileFlag, "./install.ast.log",
		"Installation log file path (optional)")
	installSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)

	upSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)
	upSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)

	downSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)

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

		var _, err = os.Stat(logFilePath)
		// create file if not exists
		if os.IsExist(err) {
			err = os.Remove(logFilePath)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to delete current installation log file", failedInstallingAST)
			}
		}

		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to open installation log file", failedInstallingAST)
		}

		installScriptPath := getScriptPathRelativeToInstallation("docker-install.sh", cmd)

		installationStarted := "Single node installation started"
		_ = writeToInstallationLog(logFile, installationStarted)
		_ = writeToInstallationLog(logFile, fmt.Sprintf("Running installation script from path %s", installScriptPath))
		writeToStandardOutput(installationStarted)

		var stdOut *bytes.Buffer
		var stdErr *bytes.Buffer
		stdOut, stdErr, err = runBashCommand(installScriptPath, []string{})
		_ = writeToInstallationLog(logFile, stdOut.String())
		_ = writeToInstallationLog(logFile, stdErr.String())

		if err != nil {
			msg := fmt.Sprintf("%s: Failed to run install script", failedInstallingAST)
			writeToStandardOutput(failedInstallingAST)
			writeToStandardOutput(fmt.Sprintf("For more information, read the installation log located at %s",
				logFilePath))
			_ = writeToInstallationLog(logFile, msg)
			return errors.Wrapf(err, msg)
		}
		successfully := "Single node installation completed successfully"
		_ = writeToInstallationLog(logFile, successfully)
		writeToStandardOutput(successfully)
		return nil
	}
}

func runUpSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Trying to start AST...")
		err := runUpScript(cmd)
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
		writeToStandardOutput("Trying to stop AST...")
		err := runDownScript(cmd)

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

func runUpScript(cmd *cobra.Command) error {
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

	installationFolder, _ := cmd.Flags().GetString(astInstallationDir)
	envVars := getEnvVarsForCommand(&configuration, installationFolder)

	_, _, err = runBashCommand(upScriptPath, envVars)

	if err != nil {
		return errors.Wrapf(err, "Failed to run up script")
	}
	return nil
}

func runDownScript(cmd *cobra.Command) error {
	var err error
	downScriptPath := getScriptPathRelativeToInstallation("down.sh", cmd)

	installationDir, _ := cmd.Flags().GetString(astInstallationDir)
	envs := []string{
		envKeyAndValue(astInstallationPathEnv, installationDir),
	}

	_, _, err = runBashCommand(downScriptPath, envs)
	if err != nil {
		return errors.Wrapf(err, "Failed to run down script")
	}
	return nil
}

func writeToInstallationLog(logFile io.StringWriter, msg string) error {
	_, err := logFile.WriteString(msg)
	if err != nil {
		return err
	}
	_, err = logFile.WriteString("\n")
	return err
}

func writeToStandardOutput(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}

func getPathRelativeToInstallation(filePath string, cmd *cobra.Command) string {
	installationFolder, _ := cmd.Flags().GetString(astInstallationDir)
	return path.Join(installationFolder, filePath)
}

func getScriptPathRelativeToInstallation(scriptFile string, cmd *cobra.Command) string {
	scriptsDir := ".scripts"
	return getPathRelativeToInstallation(path.Join(scriptsDir, scriptFile), cmd)
}
