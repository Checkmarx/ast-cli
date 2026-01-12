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

func NewLicenseCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Shows the license details from the JWT token",
		Long:  "Returns license information extracted from the JWT token",
		Example: heredoc.Doc(
			`
			$ cx utils license
			$ cx utils license --format json
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runLicenseCmd(jwtWrapper),
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

func runLicenseCmd(jwtWrapper wrappers.JWTWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		jwtClaims, err := jwtWrapper.GetAllJwtClaims()
		if err != nil {
			return errors.Wrap(err, "Failed to get license details from JWT token")
		}

		if jwtClaims != nil {
			format, _ := cmd.Flags().GetString(params.FormatFlag)

			if format == "" {
				format = defaultFormat
			}
			err = printer.Print(cmd.OutOrStdout(), jwtClaims, format)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
