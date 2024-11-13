package util

import (
	"fmt"
	"log"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/policymanagement"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	failedCreatingGithubPrDecoration    = "Failed creating github PR Decoration"
	failedCreatingGitlabPrDecoration    = "Failed creating gitlab MR Decoration"
	failedCreatingBitbucketPrDecoration = "Failed creating bitbucket PR Decoration"
	errorCodeFormat                     = "%s: CODE: %d, %s\n"
	policyErrorFormat                   = "%s: Failed to get scanID policy information"
	waitDelayDefault                    = 5
	resultPolicyDefaultTimeout          = 1
	failedGettingScanError              = "Failed showing a scan"
	noPRDecorationCreated               = "A PR couldn't be created for this scan because it is still in progress."
	githubOnPremURLSuffix               = "/api/v3/repos/"
	gitlabOnPremURLSuffix               = "/api/v4/"
	githubCloudURL                      = "https://api.github.com/repos/"
	gitlabCloudURL                      = "https://gitlab.com" + gitlabOnPremURLSuffix
	bitbucketCloudURL                   = "bitbucket.org"
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
	prDecorationBitbucket := PRDecorationBitbucket(prWrapper, policyWrapper, scansWrapper)

	cmd.AddCommand(prDecorationGithub)
	cmd.AddCommand(prDecorationGitlab)
	cmd.AddCommand(prDecorationBitbucket)
	return cmd
}

func IsScanRunningOrQueued(scansWrapper wrappers.ScansWrapper, scanID string) (bool, error) {
	var scanResponseModel *wrappers.ScanResponseModel
	var errorModel *wrappers.ErrorModel
	var err error

	scanResponseModel, errorModel, err = scansWrapper.GetByID(scanID)

	if err != nil {
		log.Printf("%s: %v", failedGettingScanError, err)
		return false, err
	}

	if errorModel != nil {
		log.Printf("%s: CODE: %d, %s", failedGettingScanError, errorModel.Code, errorModel.Message)
		return false, errors.New(errorModel.Message)
	}

	if scanResponseModel == nil {
		return false, errors.New(failedGettingScanError)
	}

	logger.PrintfIfVerbose("scan status: %s", scanResponseModel.Status)

	if scanResponseModel.Status == wrappers.ScanRunning || scanResponseModel.Status == wrappers.ScanQueued {
		return true, nil
	}
	return false, nil
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

	prDecorationGithub.Flags().String(params.CodeRepositoryFlag, "", params.CodeRepositoryFlagUsage)
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
                --iid <pr iid> --gitlab-project <gitlab project ID> --code-repository-url <code-repository-url>
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

	prDecorationGitlab.Flags().String(params.CodeRepositoryFlag, "", params.CodeRepositoryFlagUsage)
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

func PRDecorationBitbucket(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) *cobra.Command {
	prDecorationBitbucket := &cobra.Command{
		Use:   "bitbucket ",
		Short: "Decorate bitbucket PR with vulnerabilities",
		Long:  "Decorate bitbucket PR with vulnerabilities",
		Example: heredoc.Doc(
			`
			$ cx utils pr bitbucket --scan-id <scan-id> --token <PAT> --namespace <username (required for cloud services)> --repo-name <repository-slug>
                --pr-id <pr number> --code-repository-url <bitbucket-server-url (required for self-hosted)>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
			`,
			),
		},
		RunE: runPRDecorationBitbucket(prWrapper, policyWrapper, scansWrapper),
	}

	prDecorationBitbucket.Flags().String(params.ScanIDFlag, "", "Scan ID to retrieve results from")
	prDecorationBitbucket.Flags().String(params.SCMTokenFlag, "", params.BitbucketTokenUsage)
	prDecorationBitbucket.Flags().String(params.NamespaceFlag, "", fmt.Sprintf(params.NamespaceFlagUsage, "Bitbucket"))
	prDecorationBitbucket.Flags().String(params.RepoNameFlag, "", fmt.Sprintf(params.RepoNameFlagUsage, "Bitbucket"))
	prDecorationBitbucket.Flags().Int(params.PRBBIDFlag, 0, params.PRBBIDFlagUsage)
	prDecorationBitbucket.Flags().String(params.ProjectKeyFlag, "", params.ProjectKeyFlagUsage)
	prDecorationBitbucket.Flags().String(params.CodeRepositoryFlag, "", params.CodeRepositoryFlagUsage)

	// Set the value for token to mask the scm token
	_ = viper.BindPFlag(params.SCMTokenFlag, prDecorationBitbucket.Flags().Lookup(params.SCMTokenFlag))

	// mark all fields as required\
	_ = prDecorationBitbucket.MarkFlagRequired(params.ScanIDFlag)
	_ = prDecorationBitbucket.MarkFlagRequired(params.SCMTokenFlag)
	_ = prDecorationBitbucket.MarkFlagRequired(params.RepoNameFlag)
	_ = prDecorationBitbucket.MarkFlagRequired(params.PRBBIDFlag)

	return prDecorationBitbucket
}

func runPRDecoration(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		prNumberFlag, _ := cmd.Flags().GetInt(params.PRNumberFlag)
		apiURL, _ := cmd.Flags().GetString(params.CodeRepositoryFlag)

		scanRunningOrQueued, err := IsScanRunningOrQueued(scansWrapper, scanID)

		if err != nil {
			return err
		}

		if scanRunningOrQueued {
			log.Println(noPRDecorationCreated)
			return nil
		}

		// Retrieve policies related to the scan and project to include in the PR decoration
		policies, policyError := getScanViolatedPolicies(scansWrapper, policyWrapper, scanID, cmd)
		if policyError != nil {
			return errors.Errorf(policyErrorFormat, failedCreatingGithubPrDecoration)
		}

		// Build and post the pr decoration
		updatedAPIURL := updateAPIURLForGithubOnPrem(apiURL)

		prModel := &wrappers.PRModel{
			ScanID:    scanID,
			ScmToken:  scmTokenFlag,
			Namespace: namespaceFlag,
			RepoName:  repoNameFlag,
			PrNumber:  prNumberFlag,
			Policies:  policies,
			APIURL:    updatedAPIURL,
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

func updateAPIURLForGithubOnPrem(apiURL string) string {
	if apiURL != "" {
		return apiURL + githubOnPremURLSuffix
	}
	return githubCloudURL
}

func updateAPIURLForGitlabOnPrem(apiURL string) string {
	if apiURL != "" {
		return apiURL + gitlabOnPremURLSuffix
	}
	return gitlabCloudURL
}

func runPRDecorationGitlab(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		iIDFlag, _ := cmd.Flags().GetInt(params.PRIidFlag)
		gitlabProjectIDFlag, _ := cmd.Flags().GetInt(params.PRGitlabProjectFlag)
		apiURL, _ := cmd.Flags().GetString(params.CodeRepositoryFlag)

		scanRunningOrQueued, err := IsScanRunningOrQueued(scansWrapper, scanID)

		if err != nil {
			return err
		}

		if scanRunningOrQueued {
			log.Println(noPRDecorationCreated)
			return nil
		}

		// Retrieve policies related to the scan and project to include in the PR decoration
		policies, policyError := getScanViolatedPolicies(scansWrapper, policyWrapper, scanID, cmd)
		if policyError != nil {
			return errors.Errorf(policyErrorFormat, failedCreatingGitlabPrDecoration)
		}

		// Build and post the mr decoration
		updatedAPIURL := updateAPIURLForGitlabOnPrem(apiURL)

		prModel := &wrappers.GitlabPRModel{
			ScanID:          scanID,
			ScmToken:        scmTokenFlag,
			Namespace:       namespaceFlag,
			RepoName:        repoNameFlag,
			IiD:             iIDFlag,
			GitlabProjectID: gitlabProjectIDFlag,
			Policies:        policies,
			APIURL:          updatedAPIURL,
		}

		prResponse, errorModel, err := prWrapper.PostPRDecoration(prModel)

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

func runPRDecorationBitbucket(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		repoNameFlag, _ := cmd.Flags().GetString(params.RepoNameFlag)
		prIDFlag, _ := cmd.Flags().GetInt(params.PRBBIDFlag)
		apiURL, _ := cmd.Flags().GetString(params.CodeRepositoryFlag)
		projectKey, _ := cmd.Flags().GetString(params.ProjectKeyFlag)

		isCloud, flagRequiredErr := checkIsCloudAndValidateFlag(apiURL, namespaceFlag, projectKey)
		if flagRequiredErr != nil {
			return flagRequiredErr
		}

		scanRunningOrQueued, err := IsScanRunningOrQueued(scansWrapper, scanID)

		if err != nil {
			return err
		}

		if scanRunningOrQueued {
			log.Println(noPRDecorationCreated)
			return nil
		}

		policies, policyError := getScanViolatedPolicies(scansWrapper, policyWrapper, scanID, cmd)
		if policyError != nil {
			return errors.Errorf(policyErrorFormat, failedCreatingBitbucketPrDecoration)
		}

		prModel := createPRModel(isCloud, scanID, scmTokenFlag, namespaceFlag, repoNameFlag, prIDFlag, apiURL, projectKey, policies)
		prResponse, errorModel, err := prWrapper.PostPRDecoration(prModel)

		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf(errorCodeFormat, failedCreatingBitbucketPrDecoration, errorModel.Code, errorModel.Message)
		}

		logger.Print(prResponse)
		return nil
	}
}

func repoSlugFormatBB(repoNameFlag string) string {
	repoSlug := strings.Replace(repoNameFlag, " ", "-", -1)
	return repoSlug
}

func checkIsCloudAndValidateFlag(apiURL, namespaceFlag, projectKey string) (bool, error) {
	isCloud := isBitbucketCloud(apiURL)
	flagRequiredErr := validateBitbucketFlags(isCloud, namespaceFlag, projectKey)
	return isCloud, flagRequiredErr
}

func validateBitbucketFlags(isCloud bool, namespaceFlag, projectKey string) error {
	if isCloud {
		if namespaceFlag == "" {
			return errors.New("namespace is required for Bitbucket Cloud")
		}
	} else {
		if projectKey == "" {
			return errors.New("project key is required for Bitbucket Server")
		}
	}
	return nil
}

func isBitbucketCloud(apiURL string) bool {
	if apiURL == "" || strings.Contains(apiURL, bitbucketCloudURL) {
		return true
	}
	return false
}

func createPRModel(isCloud bool, scanID, scmTokenFlag, namespaceFlag, repoNameFlag string, prIDFlag int, apiURL, projectKey string, policies []wrappers.PrPolicy) interface{} {

	repoSlugFormatBB := repoSlugFormatBB(repoNameFlag)

	if isCloud {
		return &wrappers.BitbucketCloudPRModel{
			ScanID:    scanID,
			ScmToken:  scmTokenFlag,
			Namespace: namespaceFlag,
			RepoName:  repoSlugFormatBB,
			PRID:      prIDFlag,
			Policies:  policies,
		}
	} else {
		return &wrappers.BitbucketServerPRModel{
			ScanID:     scanID,
			ScmToken:   scmTokenFlag,
			ProjectKey: projectKey,
			RepoName:   repoSlugFormatBB,
			PRID:       prIDFlag,
			Policies:   policies,
			ServerURL:  apiURL,
		}
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
	policyResponseModel, err := policymanagement.HandlePolicyWait(waitDelayDefault,
		resultPolicyDefaultTimeout,
		policyWrapper,
		scanID,
		scanResponseModel.ProjectID,
		cmd)
	if err != nil {
		return nil, err
	}
	// transform into the PR model for violated policies
	violatedPolicies := policiesToPrPolicies(policyResponseModel)
	return violatedPolicies, nil
}

func policiesToPrPolicies(policy *wrappers.PolicyResponseModel) []wrappers.PrPolicy {
	var prPolicies []wrappers.PrPolicy
	if policy != nil {
		for _, policy := range policy.Policies {
			if len(policy.RulesViolated) == 0 {
				continue
			}
			prPolicy := wrappers.PrPolicy{}
			prPolicy.Name = policy.Name
			prPolicy.BreakBuild = policy.BreakBuild
			prPolicy.RulesNames = policy.RulesViolated
			prPolicies = append(prPolicies, prPolicy)
		}
	}
	return prPolicies
}
