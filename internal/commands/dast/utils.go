package dast

import (
	"fmt"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func addFormatFlagToMultipleCommands(commands []*cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	for _, c := range commands {
		addFormatFlag(c, defaultFormat, otherAvailableFormats...)
	}
}

func addFormatFlag(cmd *cobra.Command, defaultFormat string, otherAvailableFormats ...string) {
	cmd.PersistentFlags().String(
		commonParams.FormatFlag, defaultFormat,
		fmt.Sprintf(commonParams.FormatFlagUsageFormat, append(otherAvailableFormats, defaultFormat)),
	)
}

func printByFormat(cmd *cobra.Command, view interface{}) error {
	f, _ := cmd.Flags().GetString(commonParams.FormatFlag)
	return printer.Print(cmd.OutOrStdout(), view, f)
}

func getFilters(cmd *cobra.Command) (map[string]string, error) {
	filters, _ := cmd.Flags().GetStringSlice(commonParams.FilterFlag)
	allFilters := make(map[string]string)
	for _, filter := range filters {
		filterKeyVal := strings.Split(filter, "=")
		if len(filterKeyVal) != commonParams.KeyValuePairSize {
			return nil, errors.New("invalid filters, filters should be in a KEY=VALUE format")
		}
		allFilters[filterKeyVal[0]] = strings.Replace(
			filterKeyVal[1], ";", ",",
			strings.Count(filterKeyVal[1], ";"),
		)
	}
	return allFilters, nil
}
