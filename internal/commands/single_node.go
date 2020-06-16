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
	astRoleFlagUsage = fmt.Sprintf("The runtime role. Available roles are: %s",
		strings.Join([]string{
			commonParams.ScaAgent,
			commonParams.SastALlInOne,
			commonParams.SastManager,
			commonParams.SastEngine}, ","))
)

func NewSingleNodeCommand(healthCheckWrapper wrappers.HealthCheckWrapper, defaultConfigFileLocation string) *cobra.Command {
	singleNodeCmd := &cobra.Command{
		Use:   "single-node",
		Short: "Single Node",
	}

	upSingleNodeCmd := &cobra.Command{
		Use:   "up",
		Short: "Start the system",
		RunE:  runUpSingleNodeCommand(defaultConfigFileLocation),
	}
	downSingleNodeCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop the system",
		RunE:  runDownSingleNodeCommand(),
	}

	updateSingleNodeCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the system",
		RunE:  runUpdateSingleNodeCommand(defaultConfigFileLocation),
	}

	healthSingleNodeCmd := NewHealthCheckCommand(healthCheckWrapper)

	installationConfigFileUsage := "Configuration file path (optional)"
	installationFolderUsage := "Installation folder path"
	installationFolderDefault := "./"

	upSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)
	upSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)
	upSingleNodeCmd.PersistentFlags().String(astRoleFlag, "", astRoleFlagUsage)
	// Binding the AST_ROLE env var to the --role flag
	_ = viper.BindPFlag(commonParams.AstRoleKey, upSingleNodeCmd.PersistentFlags().Lookup(astRoleFlag))

	downSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)

	updateSingleNodeCmd.PersistentFlags().String(astInstallationDir, installationFolderDefault, installationFolderUsage)
	updateSingleNodeCmd.PersistentFlags().String(configFileFlag, "", installationConfigFileUsage)

	healthSingleNodeCmd.PersistentFlags().String(astRoleFlag, commonParams.ScaAgent, astRoleFlagUsage)
	_ = viper.BindPFlag(commonParams.AstRoleKey, healthSingleNodeCmd.PersistentFlags().Lookup(astRoleFlag))

	singleNodeCmd.AddCommand(
		upSingleNodeCmd,
		downSingleNodeCmd,
		healthSingleNodeCmd,
		updateSingleNodeCmd)
	return singleNodeCmd
}

func runUpSingleNodeCommand(defaultConfigFileLocation string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Trying to start...")
		err := runUpScript(cmd, defaultConfigFileLocation)
		if err != nil {
			msg := "Failed to start"
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("System is up!")
		return nil
	}
}

func runDownSingleNodeCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Trying to stop...")
		err := runDownScript(cmd)

		if err != nil {
			msg := "Failed to stop"
			return errors.Wrapf(err, msg)
		}
		writeToStandardOutput("System is down!")
		return nil
	}
}

func runUpdateSingleNodeCommand(defaultConfigFileLocation string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := runDownSingleNodeCommand()(cmd, args)
		if err != nil {
			return err
		}
		err = runUpSingleNodeCommand(defaultConfigFileLocation)(cmd, args)
		if err != nil {
			return err
		}
		writeToStandardOutput("System updated successfully!")
		return nil
	}
}

func runUpScript(cmd *cobra.Command, defaultConfigFileLocation string) error {
	upScriptPath := getScriptPathRelativeToInstallation("up.sh", cmd)
	role := viper.GetString(commonParams.AstRoleKey)

	err := runWithConfig(cmd, upScriptPath, role, defaultConfigFileLocation)
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

func runWithConfig(cmd *cobra.Command, scriptPath, role, defaultConfigFileLocation string) error {
	var err error
	configuration := &config.SingleNodeConfiguration{}

	configFile, _ := cmd.Flags().GetString(configFileFlag)

	// Give precedence to the the config flag
	// We have been provided with a config file
	if configFile != "" {
		configuration, err = tryLoadConfiguration(configFile)
		if err != nil {
			return err
		}
	} else {
		// Try to run with the default config file
		if _, err = os.Stat(defaultConfigFileLocation); err == nil {
			PrintIfVerbose(fmt.Sprintf("Reading configuration from default location at %s", defaultConfigFileLocation))
			configuration, err = tryLoadConfiguration(defaultConfigFileLocation)
			if err != nil {
				return err
			}
		} else if os.IsNotExist(err) {
			PrintIfVerbose(fmt.Sprintf("No configuration file provided via flag and no configutaion file was found at %s. "+
				"Proceeding with default configuration.", defaultConfigFileLocation))
		}
	}

	installationFolder, _ := cmd.Flags().GetString(astInstallationDir)
	envVars := createEnvVarsForCommand(configuration, installationFolder, role)

	_, _, err = runBashCommand(scriptPath, envVars)
	return err
}

func tryLoadConfiguration(configFile string) (*config.SingleNodeConfiguration, error) {
	configuration := config.SingleNodeConfiguration{}

	PrintIfVerbose(fmt.Sprintf("Trying to load configuration from file %s", configFile))
	configInput, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load config file")
	}

	err = yaml.Unmarshal(configInput, &configuration)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to parse configuration file")
	}
	return &configuration, nil
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
