package util

import (
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	asserts "github.com/stretchr/testify/assert"
	"testing"

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
