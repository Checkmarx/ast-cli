package util

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers/configuration"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedSettingProp = "Failed to set property"
	propNameFlag      = "prop-name"
	propValFlag       = "prop-value"
)

func NewConfigCommand() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "configure",
		Short: "Manage scan configurations",
		Run: func(cmd *cobra.Command, args []string) {
			configuration.PromptConfiguration()
		},
	}

	storValCmd := &cobra.Command{
		Use:   "show",
		Short: "Shows current profiles configuration",
		Run: func(cmd *cobra.Command, args []string) {
			configuration.ShowConfiguration()
		},
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration property (cx_base_uri, cx_base_auth_uri, cx_apikey, cx_tenant, cx_client_id, cx_client_secret, cx_http_proxy)",
		RunE:  runSetValue(),
	}
	setCmd.PersistentFlags().String(propNameFlag, "", "Name of property set")
	setCmd.PersistentFlags().String(propValFlag, "", "Value of property set")
	scanCmd.AddCommand(storValCmd, setCmd)
	return scanCmd
}

func runSetValue() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		propName, _ := cmd.Flags().GetString(propNameFlag)
		propValue, _ := cmd.Flags().GetString(propValFlag)
		if propName == params.AstAPIKey {
			configuration.SetConfigProperty(propName, propValue)
		} else if propName == params.BaseURIKey {
			configuration.SetConfigProperty(propName, propValue)
		} else if propName == params.BaseAuthURIKey {
			configuration.SetConfigProperty(propName, propValue)
		} else if strings.EqualFold(propName, params.ProxyKey) {
			configuration.SetConfigProperty(propName, propValue)
		} else if propName == params.AccessKeySecretConfigKey {
			configuration.SetConfigProperty(propName, propValue)
		} else if propName == params.AccessKeyIDConfigKey {
			configuration.SetConfigProperty(propName, propValue)
		} else {
			return errors.Errorf("%s: unknown property or bad value", failedSettingProp)
		}
		return nil
	}
}
