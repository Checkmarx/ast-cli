package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v2"

	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	configFileFlag     = "config"
	astInstallationDir = "installation-dir"
)

func NewSingleNodeCommand() *cobra.Command {
	singleNodeCmd := &cobra.Command{
		Use:   "single-node",
		Short: "Single Node AST",
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

	upSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)
	upSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)

	downSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)

	singleNodeCmd.AddCommand(
		upSingleNodeCmd,
		downSingleNodeCmd,
		healthSingleNodeCmd,
		updateSingleNodeCmd)
	return singleNodeCmd
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
