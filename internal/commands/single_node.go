package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"

	"gopkg.in/yaml.v2"

	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	configFileFlag     = "config"
	astInstallationDir = "installation-dir"
	astRoleFlag        = "role"
)

var (
	astRoleFlagUsage = fmt.Sprintf("The AST runtime role. Available roles are: %s",
		strings.Join([]string{
			commonParams.ScaAgent,
			commonParams.SastALlInOne,
			commonParams.SastManager,
			commonParams.SastEngine}, ","))
)

func NewSingleNodeCommand(healthCheckWrapper wrappers.HealthCheckWrapper) *cobra.Command {
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
		RunE:  runUpdateSingleNodeCommand(),
	}

	healthSingleNodeCmd := NewHealthCheckCommand(healthCheckWrapper)

	installationConfigFileUsage := "Configuration file path for AST (optional)"
	installationFolderUsage := "AST installation folder path"
	installationFolderDefault := "./"

	upSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)
	upSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)
	upSingleNodeCmd.PersistentFlags().String(astRoleFlag, commonParams.ScaAgent, astRoleFlagUsage)
	// Binding the AST_ROLE env var to the --role flag
	_ = viper.BindPFlag(commonParams.AstRoleKey, upSingleNodeCmd.PersistentFlags().Lookup(astRoleFlag))

	downSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)
	downSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)

	updateSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)
	updateSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)

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
			msg := "Failed to start AST"
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
			msg := "Failed to stop AST"
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("AST is down!")
		return nil
	}
}

func runUpdateSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := runDownSingleNodeCommand()(cmd, args)
		if err != nil {
			return err
		}
		err = runUpSingleNodeCommand()(cmd, args)
		if err != nil {
			return err
		}
		writeToStandardOutput("AST updated successfully!")
		return nil
	}
}

func runUpScript(cmd *cobra.Command) error {
	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)
	role := viper.GetString(commonParams.AstRoleKey)

	err := runWithConfig(cmd, upScriptPath, role)
	if err != nil {
		return errors.Wrapf(err, "Failed to run up script")
	}
	return nil
}

func runDownScript(cmd *cobra.Command) error {
	downScriptPath := getScriptPathRelativeToInstallation("down.sh", cmd)
	role := viper.GetString(commonParams.AstRoleKey)

	err := runWithConfig(cmd, downScriptPath, role)
	if err != nil {
		return errors.Wrapf(err, "Failed to run down script")
	}
	return nil
}

func runWithConfig(cmd *cobra.Command, scriptPath, role string) error {
	var err error
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
			return errors.Wrapf(err, "Unable to parse configuration file")
		}
	}

	installationFolder, _ := cmd.Flags().GetString(astInstallationDir)
	envVars := createEnvVarsForCommand(&configuration, installationFolder, role)

	_, _, err = runBashCommand(scriptPath, envVars)
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
