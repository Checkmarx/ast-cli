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
	failedCreatingGithubPrDecoration = "Failed creating github PR Decoration"
	failedCreatingGitlabPrDecoration = "Failed creating gitlab MR Decoration"
	errorCodeFormat                  = "%s: CODE: %d, %s\n"
)

func NewPRDecorationCommand(prWrapper wrappers.PRWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Posts the comment with scan results on the Pull Request",
		Example: heredoc.Doc(
			`
			$ cx utils pr <scm-name>
		`,
		),
	}

	prDecorationGithub := PRDecorationGithub(prWrapper)
	prDecorationGitlab := PRDecorationGitlab(prWrapper)

	cmd.AddCommand(prDecorationGithub)
	cmd.AddCommand(prDecorationGitlab)
	return cmd
}

func PRDecorationGithub(prWrapper wrappers.PRWrapper) *cobra.Command {
	prDecorationGithub := &cobra.Command{
		Use:   "github",
		Short: "Decorate github PR with vulnerabilities",
		Long:  "Decorate github PR with vulnerabilities",
		Example: heredoc.Doc(
			`
			$ cx utils pr github --scan-id <scan-id> --token <PAT> --namespace <organization> --repo-name <repository>
                --pr-number <pr number>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html
			`,
			),
		},
		RunE: runPRDecoration(prWrapper),
	}

	prDecorationGithub.Flags().String(params.ScanIDFlag, "", "Scan ID to retrieve results from")
	prDecorationGithub.Flags().String(params.SCMTokenFlag, "", params.GithubTokenUsage)
	prDecorationGithub.Flags().String(params.NamespaceFlag, "", fmt.Sprintf(params.NamespaceFlagUsage, "Github"))
	prDecorationGithub.Flags().String(params.RepoNameFlag, "", fmt.Sprintf(params.RepoNameFlagUsage, "Github"))
	prDecorationGithub.Flags().Int(params.PRNumberFlag, 0, params.PRNumberFlagUsage)

	// Set the value for token to mask the scm token
	_ = viper.BindPFlag(params.SCMTokenFlag, prDecorationGithub.Flags().Lookup(params.SCMTokenFlag))

	// mark all fields as required\
	_ = prDecorationGithub.MarkFlagRequired(params.ScanIDFlag)
	_ = prDecorationGithub.MarkFlagRequired(params.SCMTokenFlag)
	_ = prDecorationGithub.MarkFlagRequired(params.NamespaceFlag)
	_ = prDecorationGithub.MarkFlagRequired(params.RepoNameFlag)
	_ = prDecorationGithub.MarkFlagRequired(params.PRNumberFlag)

	return prDecorationGithub
}

func PRDecorationGitlab(prWrapper wrappers.PRWrapper) *cobra.Command {
	prDecorationGitlab := &cobra.Command{
		Use:   "gitlab",
		Short: "Decorate gitlab PR with vulnerabilities",
		Long:  "Decorate gitlab PR with vulnerabilities",
		Example: heredoc.Doc(
			`
			$ cx utils pr gitlab --scan-id <scan-id> --token <PAT> --namespace <organization> --repo-name <repository>
                --iid <pr iid> --gitlab-project <gitlab project ID>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
			`,
			),
		},
		RunE: runPRDecorationGitlab(prWrapper),
	}

	prDecorationGitlab.Flags().String(params.ScanIDFlag, "", "Scan ID to retrieve results from")
	prDecorationGitlab.Flags().String(params.SCMTokenFlag, "", params.GitLabTokenUsage)
	prDecorationGitlab.Flags().String(params.NamespaceFlag, "", fmt.Sprintf(params.NamespaceFlagUsage, "Gitlab"))
	prDecorationGitlab.Flags().String(params.RepoNameFlag, "", fmt.Sprintf(params.RepoNameFlagUsage, "Gitlab"))
	prDecorationGitlab.Flags().Int(params.PRIidFlag, 0, params.PRIidFlagUsage)
	prDecorationGitlab.Flags().Int(params.PRGitlabProjectFlag, 0, params.PRGitlabProjectFlagUsage)

	// Set the value for token to mask the scm token
	_ = viper.BindPFlag(params.SCMTokenFlag, prDecorationGitlab.Flags().Lookup(params.SCMTokenFlag))

	// mark all fields as required\
	_ = prDecorationGitlab.MarkFlagRequired(params.ScanIDFlag)
	_ = prDecorationGitlab.MarkFlagRequired(params.SCMTokenFlag)
	_ = prDecorationGitlab.MarkFlagRequired(params.NamespaceFlag)
	_ = prDecorationGitlab.MarkFlagRequired(params.RepoNameFlag)
	_ = prDecorationGitlab.MarkFlagRequired(params.PRIidFlag)
	_ = prDecorationGitlab.MarkFlagRequired(params.PRGitlabProjectFlag)

	return prDecorationGitlab
}

func runPRDecoration(prWrapper wrappers.PRWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		prNumberFlag, _ := cmd.Flags().GetInt(params.PRNumberFlag)

		prModel := &wrappers.PRModel{
			ScanID:    scanID,
			ScmToken:  scmTokenFlag,
			Namespace: namespaceFlag,
			RepoName:  repoNameFlag,
			PrNumber:  prNumberFlag,
		}

		prResponse, errorModel, err := prWrapper.PostPRDecoration(prModel)

		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf(errorCodeFormat, failedCreatingGithubPrDecoration, errorModel.Code, errorModel.Message)
		}

		logger.Print(prResponse)

		return nil
	}
}

func runPRDecorationGitlab(prWrapper wrappers.PRWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		iIDFlag, _ := cmd.Flags().GetInt(params.PRIidFlag)
		gitlabProjectIDFlag, _ := cmd.Flags().GetInt(params.PRGitlabProjectFlag)

		prModel := &wrappers.GitlabPRModel{
			ScanID:          scanID,
			ScmToken:        scmTokenFlag,
			Namespace:       namespaceFlag,
			RepoName:        repoNameFlag,
			IiD:             iIDFlag,
			GitlabProjectID: gitlabProjectIDFlag,
		}

		prResponse, errorModel, err := prWrapper.PostGitlabPRDecoration(prModel)

		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf(errorCodeFormat, failedCreatingGitlabPrDecoration, errorModel.Code, errorModel.Message)
		}

		logger.Print(prResponse)

		return nil
	}
}
