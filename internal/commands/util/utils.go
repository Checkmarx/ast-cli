package util

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func NewUtilsCommand() *cobra.Command {
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

	completionCmd := NewCompletionCommand()

	utilsCmd.AddCommand(completionCmd, envCheckCmd)

	return utilsCmd
}
