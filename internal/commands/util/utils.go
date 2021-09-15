package util

import (
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

	completionCmd := NewCompletionCommand()

	logsCmd := NewLogsCommand(logsWraper)

	utilsCmd.AddCommand(completionCmd, envCheckCmd, logsCmd)

	return utilsCmd
}
