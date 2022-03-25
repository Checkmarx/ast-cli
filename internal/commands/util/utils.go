package util

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewUtilsCommand(gitHubWrapper wrappers.GitHubWrapper,
	azureWrapper wrappers.AzureWrapper,
	bitBucketWrapper wrappers.BitBucketWrapper,
	gitLabWrapper wrappers.GitLabWrapper) *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "Utility functions",
		Long:  "The utils command enables the ability to perform CxAST utility functions.",
		Example: heredoc.Doc(
			`
			$ cx utils env
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`,
			),
		},
	}
	envCheckCmd := NewEnvCheckCommand()

	completionCmd := NewCompletionCommand()

	utilsCmd.AddCommand(completionCmd, envCheckCmd, usercount.NewUserCountCommand(gitHubWrapper, azureWrapper, bitBucketWrapper, gitLabWrapper))

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
