package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	asserts "github.com/stretchr/testify/assert"

	"gotest.tools/assert"
)

const (
	token = "token"
)

func TestNewGithubPRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationGithub(nil, nil, nil)
	assert.Assert(t, cmd != nil, "PR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}

func TestNewGitlabMRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationGitlab(nil, nil, nil)
	assert.Assert(t, cmd != nil, "MR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}

func TestNewAzurePRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationAzure(nil, nil, nil)
	assert.Assert(t, cmd != nil, "PR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}

func TestIsScanRunning_WhenScanRunning_ShouldReturnTrue(t *testing.T) {
	scansMockWrapper := &mock.ScansMockWrapper{Running: true}

	scanRunning, _ := IsScanRunningOrQueued(scansMockWrapper, "ScanRunning")
	asserts.True(t, scanRunning)
}

func TestIsScanRunning_WhenScanDone_ShouldReturnFalse(t *testing.T) {
	scansMockWrapper := &mock.ScansMockWrapper{Running: false}

	scanRunning, _ := IsScanRunningOrQueued(scansMockWrapper, "ScanNotRunning")
	asserts.False(t, scanRunning)
}

func TestPRDecorationGithub_WhenNoViolatedPolicies_ShouldNotReturnPolicy(t *testing.T) {
	prMockWrapper := &mock.PolicyMockWrapper{}
	policyResponse, _, _ := prMockWrapper.EvaluatePolicy(nil)
	prPolicy := policiesToPrPolicies(policyResponse)
	asserts.True(t, len(prPolicy) == 0)
}

func TestUpdateAPIURLForGithubOnPrem_whenAPIURLIsSet_ShouldUpdateAPIURL(t *testing.T) {
	selfHostedURL := "https://github.example.com"
	updatedAPIURL := updateAPIURLForGithubOnPrem(selfHostedURL)
	asserts.Equal(t, selfHostedURL+githubOnPremURLSuffix, updatedAPIURL)
}

func TestUpdateAPIURLForGithubOnPrem_whenAPIURLIsNotSet_ShouldReturnCloudAPIURL(t *testing.T) {
	cloudAPIURL := updateAPIURLForGithubOnPrem("")
	asserts.Equal(t, githubCloudURL, cloudAPIURL)
}

func TestUpdateAPIURLForGitlabOnPrem_whenAPIURLIsSet_ShouldUpdateAPIURL(t *testing.T) {
	selfHostedURL := "https://gitlab.example.com"
	updatedAPIURL := updateAPIURLForGitlabOnPrem(selfHostedURL)
	asserts.Equal(t, selfHostedURL+gitlabOnPremURLSuffix, updatedAPIURL)
}

func TestUpdateAPIURLForGitlabOnPrem_whenAPIURLIsNotSet_ShouldReturnCloudAPIURL(t *testing.T) {
	cloudAPIURL := updateAPIURLForGitlabOnPrem("")
	asserts.Equal(t, gitlabCloudURL, cloudAPIURL)
}

func TestCheckIsCloudAndValidateFlag(t *testing.T) {
	tests := []struct {
		name          string
		apiURL        string
		namespaceFlag string
		projectKey    string
		expectedCloud bool
		expectedError string
	}{
		{
			name:          "Bitbucket Cloud",
			apiURL:        "",
			namespaceFlag: "namespace",
			projectKey:    "",
			expectedCloud: true,
			expectedError: "",
		},
		{
			name:          "Bitbucket Cloud without https",
			apiURL:        "bitbucket.org",
			namespaceFlag: "namespace",
			projectKey:    "",
			expectedCloud: true,
			expectedError: "",
		},
		{
			name:          "Bitbucket Cloud with namespace",
			apiURL:        "https://bitbucket.org",
			namespaceFlag: "namespace",
			projectKey:    "",
			expectedCloud: true,
			expectedError: "",
		},
		{
			name:          "Bitbucket Cloud without namespace",
			apiURL:        "https://bitbucket.org",
			namespaceFlag: "",
			projectKey:    "",
			expectedCloud: true,
			expectedError: "namespace is required for Bitbucket Cloud",
		},
		{
			name:          "Bitbucket Server with project key and API URL",
			apiURL:        "https://bitbucket.example.com",
			namespaceFlag: "",
			projectKey:    "projectKey",
			expectedCloud: false,
			expectedError: "",
		},
		{
			name:          "Bitbucket Server without project key",
			apiURL:        "https://bitbucket.example.com",
			namespaceFlag: "",
			projectKey:    "",
			expectedCloud: false,
			expectedError: "project key is required for Bitbucket Server",
		},
		{
			name:          "Bitbucket Cloud with URL and project key",
			apiURL:        "https://bitbucket.org",
			namespaceFlag: "",
			projectKey:    "projectKey",
			expectedCloud: true,
			expectedError: "namespace is required for Bitbucket Cloud",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			isCloud, err := checkIsCloudAndValidateFlag(tt.apiURL, tt.namespaceFlag, tt.projectKey)
			asserts.Equal(t, tt.expectedCloud, isCloud)
			if tt.expectedError != "" {
				asserts.EqualError(t, err, tt.expectedError)
			} else {
				asserts.NoError(t, err)
			}
		})
	}
}

func TestRepoSlugFormatBB(t *testing.T) {
	tests := []struct {
		name         string
		repoNameFlag string
		expectedSlug string
	}{
		{
			name:         "Single word repo name",
			repoNameFlag: "repository",
			expectedSlug: "repository",
		},
		{
			name:         "Repo name with spaces",
			repoNameFlag: "my repository",
			expectedSlug: "my-repository",
		},
		{
			name:         "Repo name with multiple spaces",
			repoNameFlag: "my awesome repository",
			expectedSlug: "my-awesome-repository",
		},
		{
			name:         "Repo name with leading and trailing spaces",
			repoNameFlag: " my repository ",
			expectedSlug: "my-repository",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			slug := formatRepoNameSlugBB(tt.repoNameFlag)
			asserts.Equal(t, tt.expectedSlug, slug)
		})
	}
}

func TestGetAzureAPIURL_whenAPIURLIsSet_ShouldUpdateAPIURL(t *testing.T) {
	selfHostedURL := "https://azure.example.com"
	updatedAPIURL := getAzureAPIURL(selfHostedURL)
	asserts.Equal(t, selfHostedURL, updatedAPIURL)
}

func TestGetAzureAPIURL_whenAPIURLIsNotSet_ShouldReturnCloudAPIURL(t *testing.T) {
	cloudAPIURL := getAzureAPIURL("")
	asserts.Equal(t, azureCloudURL, cloudAPIURL)
}

func TestUpdateScmTokenForAzureOnPrem_whenUserNameIsSet_ShouldUpdateToken(t *testing.T) {
	username := "username"
	expectedToken := username + ":" + token
	updatedToken := updateScmTokenForAzure(token, username)
	asserts.Equal(t, expectedToken, updatedToken)
}

func TestUpdateScmTokenForAzureOnPrem_whenUserNameNotSet_ShouldNotUpdateToken(t *testing.T) {
	username := ""
	expectedToken := token
	updatedToken := updateScmTokenForAzure(token, username)
	asserts.Equal(t, expectedToken, updatedToken)
}

func TestCreateAzureNameSpace_ShouldCreateNamespace(t *testing.T) {
	azureNamespace := createAzureNameSpace("organization", "project")
	asserts.Equal(t, "organization/project", azureNamespace)
}

func TestValidateAzureOnPremParameters_WhenParametersAreValid_ShouldReturnNil(t *testing.T) {
	err := validateAzureOnPremParameters("https://azure.example.com", "username")
	asserts.Nil(t, err)
}

func TestValidateAzureOnPremParameters_WhenParametersAreNotValid_ShouldReturnError(t *testing.T) {
	err := validateAzureOnPremParameters("", "username")
	asserts.NotNil(t, err)
}

// ── policiesToPrPolicies (included branch) ──────────────────────────────────

func TestPoliciesToPrPolicies_IncludesViolatedPolicies(t *testing.T) {
	policy := &wrappers.PolicyResponseModel{
		Policies: []wrappers.Policy{
			{Name: "clean-policy", RulesViolated: []string{}},
			{Name: "violated-policy", BreakBuild: true, RulesViolated: []string{"rule-1", "rule-2"}},
		},
	}
	result := policiesToPrPolicies(policy)
	asserts.Len(t, result, 1)
	asserts.Equal(t, "violated-policy", result[0].Name)
	asserts.True(t, result[0].BreakBuild)
	asserts.Equal(t, []string{"rule-1", "rule-2"}, result[0].RulesNames)
}

// ── createBBPRModel ──────────────────────────────────────────────────────────

func TestCreateBBPRModel_Cloud_ReturnsCloudModel(t *testing.T) {
	model := createBBPRModel(true, "scan-1", "token", "my-namespace", "My Repo Name", 7, "", "", nil)
	cloudModel, ok := model.(*wrappers.BitbucketCloudPRModel)
	asserts.True(t, ok, "expected *wrappers.BitbucketCloudPRModel")
	asserts.Equal(t, "My-Repo-Name", cloudModel.RepoName)
	asserts.Equal(t, "my-namespace", cloudModel.Namespace)
	asserts.Equal(t, 7, cloudModel.PRID)
}

func TestCreateBBPRModel_Server_ReturnsServerModel(t *testing.T) {
	model := createBBPRModel(false, "scan-1", "token", "my-namespace", "My Repo", 9, "https://bb.example.com", "PROJ", nil)
	serverModel, ok := model.(*wrappers.BitbucketServerPRModel)
	asserts.True(t, ok, "expected *wrappers.BitbucketServerPRModel")
	asserts.Equal(t, "My-Repo", serverModel.RepoName)
	asserts.Equal(t, "PROJ", serverModel.ProjectKey)
	asserts.Equal(t, "https://bb.example.com", serverModel.ServerURL)
	asserts.Equal(t, 9, serverModel.PRID)
}

// ── getScanViolatedPolicies ──────────────────────────────────────────────────

func TestGetScanViolatedPolicies_ScanWrapperError_ReturnsError(t *testing.T) {
	cmd := &cobra.Command{}
	_, err := getScanViolatedPolicies(&mock.ScansMockWrapper{}, &mock.PolicyMockWrapper{}, "fake-error-id", cmd)
	asserts.Error(t, err, "fake error message")
}

// ── PR decoration commands: fast paths that never reach policy evaluation ──

func TestRunPRDecorationGithub_ScanRunning_SkipsDecoration(t *testing.T) {
	cmd := PRDecorationGithub(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRNumberFlag, "1"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunPRDecorationGithub_ScanWrapperError_ReturnsError(t *testing.T) {
	cmd := PRDecorationGithub(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "fake-error-id"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRNumberFlag, "1"))

	asserts.Error(t, cmd.RunE(cmd, nil), "fake error message")
}

func TestRunPRDecorationGithub_Success_PostsDecoration(t *testing.T) {
	cmd := PRDecorationGithub(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRNumberFlag, "1"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunPRDecorationGitlab_ScanRunning_SkipsDecoration(t *testing.T) {
	cmd := PRDecorationGitlab(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRIidFlag, "1"))
	asserts.NoError(t, cmd.Flags().Set(params.PRGitlabProjectFlag, "100"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunPRDecorationGitlab_Success_PostsDecoration(t *testing.T) {
	cmd := PRDecorationGitlab(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRIidFlag, "1"))
	asserts.NoError(t, cmd.Flags().Set(params.PRGitlabProjectFlag, "100"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunPRDecorationBitbucket_MissingNamespaceForCloud_ReturnsError(t *testing.T) {
	cmd := PRDecorationBitbucket(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRBBIDFlag, "1"))
	// namespace intentionally omitted, apiURL empty => cloud, requires namespace

	err := cmd.RunE(cmd, nil)
	asserts.Error(t, err, "namespace is required for Bitbucket Cloud")
}

func TestRunPRDecorationBitbucket_Success_PostsDecoration(t *testing.T) {
	cmd := PRDecorationBitbucket(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.RepoNameFlag, "repo"))
	asserts.NoError(t, cmd.Flags().Set(params.PRBBIDFlag, "1"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunPRDecorationAzure_OnPremParamsInvalid_ReturnsError(t *testing.T) {
	cmd := PRDecorationAzure(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.AzureProjectFlag, "proj"))
	asserts.NoError(t, cmd.Flags().Set(params.PRNumberFlag, "1"))
	// code-repository-username set without code-repository-url => invalid
	asserts.NoError(t, cmd.Flags().Set(params.CodeRespositoryUsernameFlag, "someuser"))

	err := cmd.RunE(cmd, nil)
	asserts.Error(t, err, errorAzureOnPremParams)
}

func TestRunPRDecorationAzure_Success_PostsDecoration(t *testing.T) {
	cmd := PRDecorationAzure(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	asserts.NoError(t, cmd.Flags().Set(params.ScanIDFlag, "ScanNotRunning"))
	asserts.NoError(t, cmd.Flags().Set(params.SCMTokenFlag, "tok"))
	asserts.NoError(t, cmd.Flags().Set(params.NamespaceFlag, "ns"))
	asserts.NoError(t, cmd.Flags().Set(params.AzureProjectFlag, "proj"))
	asserts.NoError(t, cmd.Flags().Set(params.PRNumberFlag, "1"))

	asserts.NoError(t, cmd.RunE(cmd, nil))
}

func TestNewPRDecorationCommand_HasAllSubcommands(t *testing.T) {
	cmd := NewPRDecorationCommand(&mock.PRMockWrapper{}, &mock.PolicyMockWrapper{}, &mock.ScansMockWrapper{})
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	asserts.True(t, names["github"])
	asserts.True(t, names["gitlab"])
	asserts.True(t, names["azure"])
}
