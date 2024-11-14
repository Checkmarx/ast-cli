package util

import (
	"fmt"
	"log"

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
	failedCreatingGithubPrDecoration = "Failed creating github PR Decoration"
	failedCreatingAzurePrDecoration  = "Failed creating azure PR Decoration"
	failedCreatingGitlabPrDecoration = "Failed creating gitlab MR Decoration"
	errorCodeFormat                  = "%s: CODE: %d, %s\n"
	policyErrorFormat                = "%s: Failed to get scanID policy information"
	waitDelayDefault                 = 5
	resultPolicyDefaultTimeout       = 1
	failedGettingScanError           = "Failed showing a scan"
	noPRDecorationCreated            = "A PR couldn't be created for this scan because it is still in progress."
	githubOnPremURLSuffix            = "/api/v3/repos/"
	gitlabOnPremURLSuffix            = "/api/v4/"
	githubCloudURL                   = "https://api.github.com/repos/"
	gitlabCloudURL                   = "https://gitlab.com" + gitlabOnPremURLSuffix
	azureCloudURL                    = "https://dev.azure.com/"
	errorAzureOnPremParams           = "code-repository-url must be set when code-repository-username is set"
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
	prDecorationAzure := PRDecorationAzure(prWrapper, policyWrapper, scansWrapper)

	cmd.AddCommand(prDecorationGithub)
	cmd.AddCommand(prDecorationGitlab)
	cmd.AddCommand(prDecorationAzure)
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

func PRDecorationAzure(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) *cobra.Command {
	prDecorationAzure := &cobra.Command{
		Use:   "azure",
		Short: "Decorate azure PR with vulnerabilities",
		Long:  "Decorate azure PR with vulnerabilities",
		Example: heredoc.Doc(
			`
			$ cx utils pr azure --scan-id <scan-id> --token <AAD> --namespace <organization> --project <project-name or project-id>
                --pr-number <pr number> --code-repository-url <code-repository-url>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html
			`,
			),
		},
		RunE: runPRDecorationAzure(prWrapper, policyWrapper, scansWrapper),
	}

	prDecorationAzure.Flags().String(params.ScanIDFlag, "", "Scan ID to retrieve results from")
	prDecorationAzure.Flags().String(params.SCMTokenFlag, "", params.AzureTokenUsage)
	prDecorationAzure.Flags().String(params.NamespaceFlag, "", fmt.Sprintf(params.NamespaceFlagUsage, "Azure"))
	prDecorationAzure.Flags().String(params.AzureProjectFlag, "", fmt.Sprintf(params.AzureProjectFlagUsage))
	prDecorationAzure.Flags().Int(params.PRNumberFlag, 0, params.PRNumberFlagUsage)
	prDecorationAzure.Flags().String(params.CodeRepositoryFlag, "", params.CodeRepositoryFlagUsage)
	prDecorationAzure.Flags().String(params.CodeRespositoryUsernameFlag, "", fmt.Sprintf(params.CodeRespositoryUsernameFlagUsage))

	// Set the value for token to mask the scm token
	_ = viper.BindPFlag(params.SCMTokenFlag, prDecorationAzure.Flags().Lookup(params.SCMTokenFlag))

	// mark some fields as required\
	_ = prDecorationAzure.MarkFlagRequired(params.ScanIDFlag)
	_ = prDecorationAzure.MarkFlagRequired(params.SCMTokenFlag)
	_ = prDecorationAzure.MarkFlagRequired(params.NamespaceFlag)
	_ = prDecorationAzure.MarkFlagRequired(params.AzureProjectFlag)
	_ = prDecorationAzure.MarkFlagRequired(params.PRNumberFlag)

	return prDecorationAzure
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

func updateAPIURLForAzureOnPrem(apiURL string) string {
	if apiURL != "" {
		return apiURL
	}
	return azureCloudURL
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

func runPRDecorationAzure(prWrapper wrappers.PRWrapper, policyWrapper wrappers.PolicyWrapper, scansWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)
		scmTokenFlag, _ := cmd.Flags().GetString(params.SCMTokenFlag)
		namespaceFlag, _ := cmd.Flags().GetString(params.NamespaceFlag)
		projectNameFlag, _ := cmd.Flags().GetString(params.AzureProjectFlag)
		prNumberFlag, _ := cmd.Flags().GetInt(params.PRNumberFlag)
		apiURL, _ := cmd.Flags().GetString(params.CodeRepositoryFlag)
		codeRepositoryUserName, _ := cmd.Flags().GetString(params.CodeRespositoryUsernameFlag)

		errParams := validateAzureOnPremParameters(apiURL, codeRepositoryUserName)
		if errParams != nil {
			return errParams
		}

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
			return errors.Errorf(policyErrorFormat, failedCreatingAzurePrDecoration)
		}

		// Build and post the pr decoration
		updatedAPIURL := updateAPIURLForAzureOnPrem(apiURL)
		updatedScmToken := updateScmTokenForAzure(scmTokenFlag, codeRepositoryUserName)
		azureNameSpace := createAzureNameSpace(namespaceFlag, projectNameFlag)

		prModel := &wrappers.AzurePRModel{
			ScanID:    scanID,
			ScmToken:  updatedScmToken,
			Namespace: azureNameSpace,
			PrNumber:  prNumberFlag,
			Policies:  policies,
			APIURL:    updatedAPIURL,
		}
		prResponse, errorModel, err := prWrapper.PostAzurePRDecoration(prModel)
		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf(errorCodeFormat, failedCreatingAzurePrDecoration, errorModel.Code, errorModel.Message)
		}

		logger.Print(prResponse)

		return nil
	}
}

func validateAzureOnPremParameters(apiURL, codeRepositoryUserName string) error {
	if apiURL == "" && codeRepositoryUserName != "" {
		log.Println(errorAzureOnPremParams)
		return errors.New(errorAzureOnPremParams)
	}
	return nil
}

func createAzureNameSpace(namespaceFlag, projectNameFlag string) string {
	return fmt.Sprintf("%s/%s", namespaceFlag, projectNameFlag)
}

func updateScmTokenForAzure(scmTokenFlag, codeRepositoryUserName string) string {
	if codeRepositoryUserName != "" {
		return fmt.Sprintf("%s:%s", codeRepositoryUserName, scmTokenFlag)
	}
	return scmTokenFlag
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
