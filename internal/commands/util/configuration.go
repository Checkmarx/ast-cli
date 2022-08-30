package util

import (
	"strings"

	"github.com/MakeNowJust/heredoc"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedSettingProp = "Failed to set property"
	propNameFlag      = "prop-name"
	propValFlag       = "prop-value"
)

var Properties = map[string]bool{
	params.BaseURIKey:               true,
	params.BaseAuthURIKey:           true,
	params.TenantKey:                true,
	params.ProxyKey:                 true,
	params.ProxyTypeKey:             true,
	params.AccessKeyIDConfigKey:     true,
	params.AccessKeySecretConfigKey: true,
	params.AstAPIKey:                true,
	params.BranchKey:                true,
	params.ClientTimeoutKey:         true,
}

func NewConfigCommand() *cobra.Command {
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure authentication and global properties",
		Long:  "The configure command is the fastest way to set up your AST CLI",
		Example: heredoc.Doc(
			`
			$ cx configure 
			AST Base URI []: https://ast.checkmarx.net/
			AST Base Auth URI (IAM) []: https://iam.checkmarx.net/
			AST Tenant []: organization
			Do you want to use API Key authentication? (Y/N): Y
			AST API Key []: myapikey
		`,
		),
		Run: func(cmd *cobra.Command, args []string) {
			configuration.PromptConfiguration()
		},
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(
				`
				See 'cx utils env' for the list of supported environment variables	
			`,
			),
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68630-configure.html
			`,
			),
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Shows effective profile configuration",
		Run: func(cmd *cobra.Command, args []string) {
			configuration.ShowConfiguration()
		},
		Example: heredoc.Doc(
			`
			$ cx configure show
			Current Effective Configuration
                     BaseURI: 
              BaseAuthURIKey: 
                  AST Tenant: 
                   Client ID:
               Client Secret:
                      APIKey: 
                       Proxy:`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68621-checkmarx-one-cli-quick-start-guide.html
			`,
			),
		},
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration properties",
		Long:  "Set configuration properties\nRun 'cx utils env' for the list of supported environment variables",
		RunE:  runSetValue(),
		Example: heredoc.Doc(
			`
			$ cx configure set cx_base_uri <base_uri>
			Setting property [cx_base_uri] to value [<base_uri>]`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68630-configure.html#UUID-44d2b9af-ae5d-3be9-5e3e-0fda0ab85e05
			`,
			),
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
		if Properties[strings.ToLower(propName)] {
			configuration.SetConfigProperty(propName, propValue)
		} else {
			return errors.Errorf("%s: unknown property or bad value", failedSettingProp)
		}
		return nil
	}
}
