package commands

import (
	"fmt"
	"io/ioutil"

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

	installAIOCmd.PersistentFlags().String(logFileFlag, "",
		"Installation log file path (optional)")
	installAIOCmd.PersistentFlags().String(configFileFlag, "",
		"Configuration file path to provide to the AIO installation (optional)")

	aioCmd.AddCommand(installAIOCmd)
	return aioCmd
}

func runInstallAIOCommand(scriptsWrapper wrappers.ScriptsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		logFile, _ := cmd.Flags().GetString(logFileFlag)
		PrintIfVerbose(fmt.Sprintf("%s: %s", logFileFlag, logFile))

		_, err := runBashCommand(scriptsWrapper.GetInstallScriptPath())
		if err != nil {
			return errors.Wrapf(err, "%s: Failed to run install command", failedInstallingAIO)
		}
		return up(cmd, scriptsWrapper)
	}
}

func up(cmd *cobra.Command, scriptsWrapper wrappers.ScriptsWrapper) error {
	configFile, _ := cmd.Flags().GetString(configFileFlag)
	configuration := config.AIOConfiguration{}

	if configFile != "" {
		// Reading configuration from config file
		PrintIfVerbose(fmt.Sprintf("Reading configuration from file %s", configFile))
		configInput, err := ioutil.ReadFile(configFile)
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

	_, err := runBashCommand(scriptsWrapper.GetUpScriptPath(),
		logMaxSize, logAgeDays, privateKeyFile, certificateFile)
	if err != nil {
		return errors.Wrapf(err, "%s: Failed to run up command", failedRunningAIO)
	}
	return nil
}
