package commands

import (
	"github.com/spf13/cobra"
)

func NewHealthCommand() *cobra.Command {
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Show health information for AST",
		RunE:  runHealthSingleNodeCommand,
	}
	return healthCmd
}

func runHealthSingleNodeCommand(cmd *cobra.Command, args []string) error {
	return nil
}
