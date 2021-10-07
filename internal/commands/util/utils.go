package util

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewUtilsCommand(logsWraper wrappers.LogsWrapper) *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "Utility functions",
		Long:  "The utils command enables the ability to perform CxAST utility functions.",
		Example: heredoc.Doc(`
			$ cx utils env
		`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`),
		},
	}
	envCheckCmd := NewEnvCheckCommand()

	logsCmd := NewLogsCommand(logsWraper)

	completionCmd := NewCompletionCommand()

	utilsCmd.AddCommand(completionCmd, envCheckCmd, logsCmd)

	return utilsCmd
}

/**
Tests if a string exists in the provided array
*/
func contains(array []string, val string) bool {
	for _, e := range array {
		if e == val {
			return true
		}
	}
	return false
}

func executeTestCommand(cmd *cobra.Command, args ...string) error {
	fmt.Println("Executing command with args ", args)
	cmd.SetArgs(args)
	cmd.SilenceUsage = false
	return cmd.Execute()
}
