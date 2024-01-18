package util

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/policymanagement"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
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

func NewPRDecorationCommand(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Posts the comment with scan results on the Pull Request",
		Example: heredoc.Doc(
			`
			$ cx utils pr <scm-name>
		`,
		),
	}

	prDecorationGithub := PRDecorationGithub(prWrapper, policyWrapper, scansWrapper)
	prDecorationGitlab := PRDecorationGitlab(prWrapper, policyWrapper, scansWrapper)

	cmd.AddCommand(prDecorationGithub)
	cmd.AddCommand(prDecorationGitlab)
	return cmd
}

func PRDecorationGithub(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) *cobra.Command {
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
		RunE: runPRDecoration(prWrapper, policyWrapper, scansWrapper),
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

func PRDecorationGitlab(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) *cobra.Command {
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
		RunE: runPRDecorationGitlab(prWrapper, policyWrapper, scansWrapper),
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

func runPRDecoration(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		prNumberFlag, _ := cmd.Flags().GetInt(params.PRNumberFlag)

		// Retrieve policies related to the scan and project to include in the PR decoration
		policies, policyError := getScanViolatedPolicies(scansWrapper, policyWrapper, scanID, cmd)
		if policyError != nil {
			return policyError
		}

		// Build and post the pr decoration
		prModel := &wrappers.PRModel{
			ScanID:    scanID,
			ScmToken:  scmTokenFlag,
			Namespace: namespaceFlag,
			RepoName:  repoNameFlag,
			PrNumber:  prNumberFlag,
			Policies:  policies,
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

func runPRDecorationGitlab(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		iIDFlag, _ := cmd.Flags().GetInt(params.PRIidFlag)
		gitlabProjectIDFlag, _ := cmd.Flags().GetInt(params.PRGitlabProjectFlag)

		// Retrieve policies related to the scan and project to include in the PR decoration
		policies, policyError := getScanViolatedPolicies(scansWrapper, policyWrapper, scanID, cmd)
		if policyError != nil {
			return policyError
		}

		// Build and post the mr decoration
		prModel := &wrappers.GitlabPRModel{
			ScanID:          scanID,
			ScmToken:        scmTokenFlag,
			Namespace:       namespaceFlag,
			RepoName:        repoNameFlag,
			IiD:             iIDFlag,
			GitlabProjectID: gitlabProjectIDFlag,
			Policies:        policies,
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

func getScanViolatedPolicies(scansWrapper wrappers.ScansWrapper, policyWrapper wrappers.PolicyWrapper, scanID string, cmd *cobra.Command) ([]wrappers.PrPolicy, error) {
	// retrieve scan model to get the projectID
	scanResponseModel, errorScanModel, err := scansWrapper.GetByID(scanID)
	if err != nil {
		return nil, err
	}
	if errorScanModel != nil {
		return nil, err
	}
	// retrieve policy information to send to the PR service
	policyResponseModel := &wrappers.PolicyResponseModel{}
	policyResponseModel, err = policymanagement.HandlePolicyWait(commonParams.WaitDelayDefault, commonParams.ResultPolicyDefaultTimeout, policyWrapper, scanID, scanResponseModel.ProjectID, cmd)
	if err != nil {
		return nil, err
	}
	// transform into the PR model for violated policies
	violatedPolicies := policiesToPrPolicies(policyResponseModel.Policies)
	return violatedPolicies, nil
}

func policiesToPrPolicies(policies []wrappers.Policy) []wrappers.PrPolicy {
	var prPolicies []wrappers.PrPolicy
	for _, policy := range policies {
		prPolicy := wrappers.PrPolicy{}
		prPolicy.Name = policy.Name
		prPolicy.BreakBuild = policy.BreakBuild
		prPolicy.RulesNames = policy.RulesViolated
		prPolicies = append(prPolicies, prPolicy)
	}
	return prPolicies
}
