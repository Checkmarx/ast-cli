package usercount

import (
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	userCountFlag         = "user-count"
	userCountShort        = "Display SCM contributor counts for the past 90 days"
	TotalContributorsName = "Total unique contributors"
)

var (
	token, ninetyDaysDate, format string
	hiddenFlags                   = []string{
		params.AgentFlag,
		params.AstAPIKeyFlag,
		params.BaseAuthURIFlag,
		params.BaseURIFlag,
		params.AccessKeyIDFlag,
		params.AccessKeySecretFlag,
		params.InsecureFlag,
		params.ProfileFlag,
		params.RetryFlag,
		params.RetryDelayFlag,
		params.TenantFlag,
	}
)

func NewUserCountCommand(gitHubWrapper wrappers.GitHubWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:   userCountFlag,
		Short: userCountShort,
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

	for _, cmd := range userCountCmd.Commands() {
		cmd.Flags().StringVar(&token, params.SCMTokenFlag, "", params.SCMTokenUsage)
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

	// subtract ninety days from current date
	ninetyDaysDate = time.Now().UTC().Add(-90 * 24 * time.Hour).Format(time.RFC3339)

	return userCountCmd
}
