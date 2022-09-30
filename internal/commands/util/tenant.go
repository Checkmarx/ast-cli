package util

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewTenantConfigurationCommand(wrapper wrappers.TenantConfigurationWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Shows the tenant settings",
		Example: heredoc.Doc(
			`
			$ cx utils tenant
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runTenantCmd(wrapper),
	}
	cmd.PersistentFlags().String(
		params.FormatFlag,
		"",
		fmt.Sprintf(
			params.FormatFlagUsageFormat,
			[]string{printer.FormatTable, printer.FormatJSON, printer.FormatList},
		),
	)
	return cmd
}

func runTenantCmd(wrapper wrappers.TenantConfigurationWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		tenantConfigurationResponse, errorModel, err := wrapper.GetTenantConfiguration()
		if err != nil {
			return err
		}
		if errorModel != nil {
			return errors.Errorf("Failed getting tenant configuration: %v", errorModel)
		}
		if tenantConfigurationResponse != nil {
			format, _ := cmd.Flags().GetString(params.FormatFlag)
			tenantConfigurationResponseView := toTenantConfigurationResponseView(tenantConfigurationResponse)
			if format == "" {
				format = defaultFormat
			}
			err := printer.Print(cmd.OutOrStdout(), tenantConfigurationResponseView, format)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toTenantConfigurationResponseView(response *[]*wrappers.TenantConfigurationResponse) interface{} {
	var tenantConfigurationResponseView []*wrappers.TenantConfigurationResponse
	for _, resp := range *response {
		tenantConfigurationResponseView = append(
			tenantConfigurationResponseView, &wrappers.TenantConfigurationResponse{
				Key:   resp.Key,
				Value: resp.Value,
			},
		)
	}
	return tenantConfigurationResponseView
}
