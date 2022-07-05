package util

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	invalidFlag = "Value of %s is invalid"
)

func NewLearnMoreCommand(wrapper wrappers.LearnMoreWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn-more",
		Short: "Shows the descriptions and additional details for a query id",
		Example: heredoc.Doc(
			`
			$ cx utils learn-more
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`,
			),
		},
		RunE: runLearnMoreCmd(wrapper),
	}
	return cmd
}

func runLearnMoreCmd(wrapper wrappers.LearnMoreWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		queryID := viper.GetString(params.QueryIDFlag)
		if queryID == "" {
			return errors.Errorf(
				invalidFlag, params.QueryIDFlag)
		}
		LearnMoreResponse, errorModel, err := wrapper.GetLearnMoreDetails(queryID)
		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf("Failed getting additional details")
		}

		if LearnMoreResponse != nil {
			logger.PrintIfVerbose(fmt.Sprintf("Response from wrapper: %s", LearnMoreResponse))
		}

		return nil
	}
}
