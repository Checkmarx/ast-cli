package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"cliversion"},
		Short:   "Print version information",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}