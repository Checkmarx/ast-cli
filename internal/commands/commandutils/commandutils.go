package commandutils

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
)

func AddFormatFlagToMultipleCommands(commands []*cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	for _, c := range commands {
		AddFormatFlag(c, defaultFormat, otherAvailableFormats...)
	}
}

func AddFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(
		params.FormatFlag, defaultFormat,
		fmt.Sprintf(params.FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)),
	)
}

func PrintByFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(params.FormatFlag)
	return printer.Print(cmd.OutOrStdout(), view, f)
}
