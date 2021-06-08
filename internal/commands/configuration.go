package commands

import (
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedSettingProp = "Failed to set property"
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
	scanCmd.AddCommand(storValCmd, setCmd)
	return scanCmd
}

func runSetValue() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		const minArgCnt = 2
		if len(args) < minArgCnt {
			return errors.Errorf("%s: Please provide a property to set and a value", failedDeleting)
		}
		if args[0] == params.AstAPIKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == params.BaseURIKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == params.BaseAuthURIKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == params.ProxyKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == params.AccessKeySecretConfigKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else if args[0] == params.AccessKeyIDConfigKey {
			wrappers.SetConfigProperty(args[0], args[1])
		} else {
			return errors.Errorf("%s: unknown property or bad value", failedSettingProp)
		}
		return nil
	}
}
