package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
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
			wrappers.PromptConfiguration()
		},
	}

	storValCmd := &cobra.Command{
		Use:   "show",
		Short: "Shows current profiles configuration",
		Run: func(cmd *cobra.Command, args []string) {
			wrappers.ShowConfiguration()
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
			wrappers.SetConfigProperty(propName, propValue)
		} else if propName == params.BaseURIKey {
			wrappers.SetConfigProperty(propName, propValue)
		} else if propName == params.BaseAuthURIKey {
			wrappers.SetConfigProperty(propName, propValue)
		} else if propName == params.ProxyKey {
			wrappers.SetConfigProperty(propName, propValue)
		} else if propName == params.AccessKeySecretConfigKey {
			wrappers.SetConfigProperty(propName, propValue)
		} else if propName == params.AccessKeyIDConfigKey {
			wrappers.SetConfigProperty(propName, propValue)
		} else {
			return errors.Errorf("%s: unknown property or bad value", failedSettingProp)
		}
		return nil
	}
}
