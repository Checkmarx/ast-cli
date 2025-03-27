package usercount

import (
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/commands/util/usercount/bitbucketserver"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	bitbucketServerWrapper "github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/spf13/cobra"
)

const (
	UcCommand             = "contributor-count"
	UcShort               = "The contributor-count command enables the ability to count unique contributors from different SCM repositories, for the past 90 days."
	TotalContributorsName = "Total unique contributors"
)

var (
	ninetyDaysDate, format string
	hiddenFlags            = []string{
		params.AgentFlag,
		params.AstAPIKeyFlag,
		params.BaseAuthURIFlag,
		params.BaseURIFlag,
		params.AccessKeyIDFlag,
		params.AccessKeySecretFlag,
		params.InsecureFlag,
		params.RetryFlag,
		params.RetryDelayFlag,
		params.TenantFlag,
	}
)

func NewUserCountCommand(
	gitHubWrapper wrappers.GitHubWrapper,
	azureWrapper wrappers.AzureWrapper,
	bitBucketWrapper wrappers.BitBucketWrapper,
	bitBucketServerWrapper bitbucketServerWrapper.Wrapper,
	gitLabWrapper wrappers.GitLabWrapper,
) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:   UcCommand,
		Short: UcShort,
		Args:  cobra.NoArgs,
	}

	userCountCmd.SetHelpFunc(
		func(cmd *cobra.Command, args []string) {
			for _, f := range hiddenFlags {
				_ = cmd.Flags().MarkHidden(f)
			}

			userCountCmd.Parent().HelpFunc()(cmd, args)
		},
	)

	userCountCmd.AddCommand(newUserCountGithubCommand(gitHubWrapper))
	userCountCmd.AddCommand(newUserCountAzureCommand(azureWrapper))

	userCountCmd.AddCommand(newUserCountBitBucketCommand(bitBucketWrapper))

	userCountCmd.AddCommand(newUserCountGitLabCommand(gitLabWrapper))

	for _, cmd := range userCountCmd.Commands() {
		cmd.Flags().StringVar(
			&format,
			params.FormatFlag,
			printer.FormatTable,
			fmt.Sprintf(
				params.FormatFlagUsageFormat,
				[]string{printer.FormatTable, printer.FormatJSON, printer.FormatList},
			),
		)
	}

	userCountCmd.AddCommand(bitbucketserver.NewUserCountBitBucketServerCommand(bitBucketServerWrapper))

	// subtract ninety days from current date
	ninetyDaysDate = time.Now().UTC().Add(-90 * 24 * time.Hour).Format(time.RFC3339)

	return userCountCmd
}
