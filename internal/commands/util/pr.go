package util

import (
	"fmt"
	"log"
	"strconv"

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
	cmd.PersistentFlags().String(params.SCMTokenFlag, "", "SCM Token is needed")
	cmd.PersistentFlags().String(params.NamespaceFlag, "", "Namespace is needed")
	cmd.PersistentFlags().String(params.RepoNameFlag, "", "RepoName is needed")
	cmd.PersistentFlags().String(params.PRNumberFlag, "", "PR Number is needed")

	flagsMap := make(map[string]string)
	flagsMap[params.SCMTokenKey] = params.SCMTokenFlag
	flagsMap[params.CxScanKey] = params.ScanIDFlag
	flagsMap[params.OrgNamespaceKey] = params.NamespaceFlag
	flagsMap[params.OrgRepoNameKey] = params.RepoNameFlag
	flagsMap[params.PRNumberKey] = params.PRNumberFlag
	bindCmdVariablesToViperEnvVariables(flagsMap, cmd)

	return cmd
}

func bindCmdVariablesToViperEnvVariables(flagsMap map[string]string, cmd *cobra.Command) {
	for k, v := range flagsMap {
		err := viper.BindPFlag(k, cmd.PersistentFlags().Lookup(v))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func markFlagAsRequired(cmd *cobra.Command, flag string) {
	err := cmd.MarkPersistentFlagRequired(flag)
	if err != nil {
		log.Fatal(err)
	}
}

func runPRDecoration(prWrapper wrappers.PRWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID := viper.GetString(params.CxScanKey)
		if scanID == "" {
			return errors.Errorf(
				invalidFlag, params.ScanIDFlag)
		}
		scmToken := viper.GetString(params.SCMTokenKey)
		if scmToken == "" {
			return errors.Errorf(
				invalidFlag, params.SCMTokenFlag)
		}
		namespace := viper.GetString(params.OrgNamespaceKey)
		if namespace == "" {
			return errors.Errorf(
				invalidFlag, params.NamespaceFlag)
		}
		repoName := viper.GetString(params.OrgRepoNameKey)
		if repoName == "" {
			return errors.Errorf(
				invalidFlag, params.RepoNameFlag)
		}
		prNumberString := viper.GetString(params.PRNumberKey)
		if prNumberString == "" {
			return errors.Errorf(
				invalidFlag, params.PRNumberFlag)
		}
		prNumber, err := strconv.Atoi(prNumberString)

		if err != nil {
			return errors.Errorf(invalidFlag, params.PRNumberFlag)
		}

		// Set the value for token to mask the scm token
		viper.Set(params.SCMTokenFlag, scmToken)

		// mark all fields as required
		markFlagAsRequired(cmd, params.ScanIDFlag)
		markFlagAsRequired(cmd, params.SCMTokenFlag)
		markFlagAsRequired(cmd, params.NamespaceFlag)
		markFlagAsRequired(cmd, params.RepoNameFlag)
		markFlagAsRequired(cmd, params.PRNumberFlag)

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
			return errors.Errorf("Failed creating PR Decoration")
		}

		if prResponse != nil {
			logger.PrintIfVerbose(fmt.Sprintf("Response from wrapper: %s", prResponse.Message))
		}

		return nil

	}
}
