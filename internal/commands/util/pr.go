package util

import (
	"fmt"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"strconv"

	"github.com/checkmarx/ast-cli/internal/params"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedCreatingPRDecoration = "failed creating PR Decoration"
	pleaseProvideFlag          = "%s: Please provide %s flag"
	invalidFlag                = "Value of %s is invalid"
)

func NewPRDecorationCommand(prWrapper wrappers.PRWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Posts the comment with scan results on the Pull Request",
		Example: heredoc.Doc(
			`
			$ cx utils pr
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`,
			),
		},
		RunE: runPRDecoration(prWrapper),
	}
	cmd.PersistentFlags().String(params.ScanIDFlag, "", "Scan ID is needed")
	cmd.PersistentFlags().String(params.ScmTokenRef, "", "SCM Token is needed")
	cmd.PersistentFlags().String(params.NamespaceFlag, "", "Namespace is needed")
	cmd.PersistentFlags().String(params.RepoNameFlag, "", "RepoName is needed")
	cmd.PersistentFlags().String(params.PRNumberFlag, "", "PR Number is needed")

	return cmd
}

func runPRDecoration(prWrapper wrappers.PRWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		if scanID == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingPRDecoration, params.ScanIDFlag)
		}
		scmToken, _ := cmd.Flags().GetString(params.ScmTokenRef)
		if scmToken == "" {
			scmToken = viper.GetString(commonParams.SCMTokenKey)
		}
		if scmToken == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingPRDecoration, params.ScmTokenRef)
		}
		viper.Set(params.ScmTokenRef, scmToken)
		namespace, _ := cmd.Flags().GetString(params.NamespaceFlag)
		if namespace == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingPRDecoration, params.NamespaceFlag)
		}
		repoName, _ := cmd.Flags().GetString(params.RepoNameFlag)
		if repoName == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingPRDecoration, params.RepoNameFlag)
		}
		prNumberString, _ := cmd.Flags().GetString(params.PRNumberFlag)
		if prNumberString == "" {
			return errors.Errorf(pleaseProvideFlag, failedCreatingPRDecoration, params.PRNumberFlag)
		}
		prNumber, err := strconv.Atoi(prNumberString)

		if err != nil {
			return errors.Errorf(invalidFlag, params.PRNumberFlag)
		}

		prModel := &wrappers.PRModel{
			ScanID:    scanID,
			ScmToken:  scmToken,
			Namespace: namespace,
			RepoName:  repoName,
			PrNumber:  prNumber,
		}

		prResponse, errorModel, err := prWrapper.PostPRDecoration(prModel)

		if err != nil {
			return err
		}

		if errorModel != nil {
			errors.Errorf("Failed creating PR Decoration")
		}

		if prResponse != nil {
			logger.PrintIfVerbose(fmt.Sprintf("Response from wrapper: %s", prResponse.Message))
		}

		return nil

	}
}
