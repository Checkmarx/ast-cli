package util

import (
	"github.com/spf13/cobra"
)

func NewUtilsCommand() *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "AST Utility functions",
	}
	envCheckCmd := NewEnvCheckCommand()

	completionCmd := NewCompletionCommand()

	utilsCmd.AddCommand(completionCmd, envCheckCmd)

	return utilsCmd
}
