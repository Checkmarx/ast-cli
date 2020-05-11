package commands

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/checkmarxDev/ast-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	logFileFlag         = "log"
	configFileFlag      = "config"
	failedInstallingAIO = "Failed installing All-In-One"
)

func NewAIOCommand() *cobra.Command {
	aioCmd := &cobra.Command{
		Use:   "aio",
		Short: "All-In-One AST",
	}

	installAIOCmd := &cobra.Command{
		Use:   "install",
		Short: "Install All-In-One AST",
		RunE:  runInstallAIOCommand,
	}

	installAIOCmd.PersistentFlags().String(logFileFlag, "",
		"Installation log file path (optional)")
	installAIOCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path to provide to the AIO installation (optional)")

	aioCmd.AddCommand(installAIOCmd)
	return aioCmd
}

func runInstallAIOCommand(cmd *cobra.Command, args []string) error {
	var err error
	logFile, _ := cmd.Flags().GetString(logFileFlag)
	configFile, _ := cmd.Flags().GetString(configFileFlag)
	PrintIfVerbose(fmt.Sprintf("%s: %s", logFileFlag, logFile))
	PrintIfVerbose(fmt.Sprintf("%s: %s", configFileFlag, configFile))

	if configFile != "" {
		// Reading configuration from config file
		PrintIfVerbose(fmt.Sprintf("Reading configuration from file %s", configFile))
		configInput, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to open config file", failedInstallingAIO)
		}
		configuration := config.AIOConfiguration{}
		err = yaml.Unmarshal(configInput, &configuration)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Unable to parse configuration file"))
		}
		err = mergeConfigurationWithEnv(&configuration)
	}

	err = runBashCommand("echo")
	if err != nil {
		return errors.Wrapf(err, "%s: Failed to run install command", failedInstallingAIO)
	}
	return nil
}
