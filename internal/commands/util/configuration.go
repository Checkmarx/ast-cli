package util

import (
	"github.com/MakeNowJust/heredoc"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedSettingProp = "Failed to set property"
	propNameFlag      = "prop-name"
	propValFlag       = "prop-value"
)

func NewConfigCommand() *cobra.Command {
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure authentication and global properties",
		Long:  "The configure command is the fastest way to set up your AST CLI",
		Example: heredoc.Doc(`
			$ cx configure
			AST Base URI []:
			AST Base Auth URI (IAM) []:
			AST Tenant []:
			Do you want to use API Key authentication? (Y/N): Y
			AST API Key []:
		`),
		Run: func(cmd *cobra.Command, args []string) {
			configuration.PromptConfiguration()
		},
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(`
				See 'cx utils env' for the list of supported environment variables	
			`),
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/gwQRtw
			`),
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Shows effective profile configuration",
		Run: func(cmd *cobra.Command, args []string) {
			configuration.ShowConfiguration()
		},
		Example: heredoc.Doc(`
			$ cx configure show
			Current Effective Configuration
                     BaseURI: 
              BaseAuthURIKey: 
                  AST Tenant: 
                   Client ID:
               Client Secret:
                      APIKey: 
                       Proxy:`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/gwQRtw
			`),
		},
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration properties",
		Long:  "Set configuration properties\nRun 'cx utils env' for the list of supported environment variables",
		RunE:  runSetValue(),
		Example: heredoc.Doc(`
			$ cx configure set cx_base_uri <base_uri>
			Setting property [cx_base_uri] to value [<base_uri>]`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/gwQRtw
			`),
		},
	}
	setCmd.PersistentFlags().String(propNameFlag, "", "Name of property set")
	setCmd.PersistentFlags().String(propValFlag, "", "Value of property set")

	configureCmd.AddCommand(showCmd, setCmd)
	return configureCmd
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
		} else if propName == params.ProxyTypeKey {
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
