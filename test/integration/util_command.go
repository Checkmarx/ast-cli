//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

//goland:noinspection HttpUrlsUsage
const (
	ProxyUserEnv = "PROXY_USERNAME"
	ProxyPwEnv   = "PROXY_PASSWORD"
	ProxyPortEnv = "PROXY_PORT"
	ProxyHostEnv = "PROXY_HOST"
	ProxyURLTmpl = "http://%s:%s@%s:%d"
	pat          = "PERSONAL_ACCESS_TOKEN"
)

// Bind environment vars and their defaults to viper
func bindKeysToEnvAndDefault(t *testing.T) {
	for _, b := range params.EnvVarsBinds {
		err := viper.BindEnv(b.Key, b.Env)
		if t != nil {
			assert.NilError(t, err)
		}
		viper.SetDefault(b.Key, b.Default)
	}
}

func bindProxy(t *testing.T) {
	err := viper.BindEnv(params.ProxyKey, params.CxProxyEnv, params.ProxyEnv)
	if err != nil {
		assert.NilError(t, err)
	}
	viper.SetDefault(params.ProxyKey, "")
	err = os.Setenv(params.ProxyEnv, viper.GetString(params.ProxyKey))
	if err != nil {
		assert.NilError(t, err)
	}
}

// Create a command to execute in tests
func createASTIntegrationTestCommand(t *testing.T) *cobra.Command {
	bindProxy(t)
	bindKeysToEnvAndDefault(t)
	_ = viper.BindEnv(pat)
	viper.AutomaticEnv()
	viper.Set("CX_TOKEN_EXPIRY_SECONDS", 2)
	scans := viper.GetString(params.ScansPathKey)
	applications := viper.GetString(params.ApplicationsPathKey)
	groups := viper.GetString(params.GroupsPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	scanSummmaryPath := viper.GetString(params.ScanSummaryPathKey)
	risksOverview := viper.GetString(params.RisksOverviewPathKey)
	scsScanOverviewPath := viper.GetString(params.ScsScanOverviewPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	logs := viper.GetString(params.LogsPathKey)
	codebashing := viper.GetString(params.CodeBashingPathKey)
	bfl := viper.GetString(params.BflPathKey)
	learnMore := viper.GetString(params.DescriptionsPathKey)
	prDecorationGithubPath := viper.GetString(params.PRDecorationGithubPathKey)
	prDecorationGitlabPath := viper.GetString(params.PRDecorationGitlabPathKey)
	prDecorationBitbucketCloudPath := viper.GetString(params.PRDecorationBitbucketCloudPathKey)
	prDecorationBitbucketServerPath := viper.GetString(params.PRDecorationBitbucketServerPathKey)
	prDecorationAzurePath := viper.GetString(params.PRDecorationAzurePathKey)
	tenantConfigurationPath := viper.GetString(params.TenantConfigurationPathKey)
	resultsPdfPath := viper.GetString(params.ResultsPdfReportPathKey)
	exportPath := viper.GetString(params.ExportPathKey)
	featureFlagsPath := viper.GetString(params.FeatureFlagsKey)
	policyEvaluationPath := viper.GetString(params.PolicyEvaluationPathKey)
	sastIncrementalPath := viper.GetString(params.SastMetadataPathKey)
	accessManagementPath := viper.GetString(params.AccessManagementPathKey)
	byorPath := viper.GetString(params.ByorPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	applicationsWrapper := wrappers.NewApplicationsHTTPWrapper(applications)
	resultsPdfReportsWrapper := wrappers.NewResultsPdfReportsHTTPWrapper(resultsPdfPath)
	exportWrapper := wrappers.NewExportHTTPWrapper(exportPath)

	resultsPredicatesWrapper := wrappers.NewResultsPredicatesHTTPWrapper()
	groupsWrapper := wrappers.NewHTTPGroupsWrapper(groups)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, scanSummmaryPath)
	risksOverviewWrapper := wrappers.NewHTTPRisksOverviewWrapper(risksOverview)
	scsScanOverviewWrapper := wrappers.NewHTTPScanOverviewWrapper(scsScanOverviewPath)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	logsWrapper := wrappers.NewLogsWrapper(logs)
	codeBashingWrapper := wrappers.NewCodeBashingHTTPWrapper(codebashing)
	gitHubWrapper := wrappers.NewGitHubWrapper()
	gitLabWrapper := wrappers.NewGitLabWrapper()
	azureWrapper := wrappers.NewAzureWrapper()
	bitBucketWrapper := wrappers.NewBitbucketWrapper()
	bflWrapper := wrappers.NewBflHTTPWrapper(bfl)
	learnMoreWrapper := wrappers.NewHTTPLearnMoreWrapper(learnMore)
	prWrapper := wrappers.NewHTTPPRWrapper(prDecorationGithubPath, prDecorationGitlabPath, prDecorationBitbucketCloudPath, prDecorationBitbucketServerPath, prDecorationAzurePath)
	tenantConfigurationWrapper := wrappers.NewHTTPTenantConfigurationWrapper(tenantConfigurationPath)
	jwtWrapper := wrappers.NewJwtWrapper()
	scaRealtimeWrapper := wrappers.NewHTTPScaRealTimeWrapper()
	chatWrapper := wrappers.NewChatWrapper()
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)
	policyWrapper := wrappers.NewHTTPPolicyWrapper(policyEvaluationPath)
	sastMetadataWrapper := wrappers.NewSastIncrementalHTTPWrapper(sastIncrementalPath)
	accessManagementWrapper := wrappers.NewAccessManagementHTTPWrapper(accessManagementPath)
	ByorWrapper := wrappers.NewByorHTTPWrapper(byorPath)
	containerResolverWrapper := wrappers.NewContainerResolverWrapper()

	astCli := commands.NewAstCLI(
		applicationsWrapper,
		scansWrapper,
		exportWrapper,
		resultsPdfReportsWrapper,
		resultsPredicatesWrapper,
		codeBashingWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
		scsScanOverviewWrapper,
		authWrapper,
		logsWrapper,
		groupsWrapper,
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		nil,
		gitLabWrapper,
		bflWrapper,
		prWrapper,
		learnMoreWrapper,
		tenantConfigurationWrapper,
		jwtWrapper,
		scaRealtimeWrapper,
		chatWrapper,
		featureFlagsWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
		ByorWrapper,
		containerResolverWrapper,
	)
	return astCli
}

// Create a test command by calling createASTIntegrationTestCommand
// Redirect stdout of the command to a buffer and return the buffer with the command
func createRedirectedTestCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	outputBuffer := bytes.NewBufferString("")
	cmd := createASTIntegrationTestCommand(t)
	cmd.SetOut(outputBuffer)
	return cmd, outputBuffer
}

/*
	Execute a previous created command. Used when there is a need to make changes in environment variables

Ex.: 1. Create a command
 2. Make changes in environment variables using viper
 3. Execute command
*/
func execute(cmd *cobra.Command, args ...string) error {
	return executeWithTimeout(cmd, 5*time.Minute, args...)
}

// Execute a CLI command expecting an error and buffer to execute post assertions
func executeCommand(t *testing.T, args ...string) (error, *bytes.Buffer) {

	cmd, buffer := createRedirectedTestCommand(t)

	err := executeWithTimeout(cmd, 5*time.Minute, args...)

	return err, buffer
}

// Execute a CLI command with nil error assertion
func executeCmdNilAssertion(t *testing.T, infoMsg string, args ...string) *bytes.Buffer {
	cmd, outputBuffer := createRedirectedTestCommand(t)

	err := execute(cmd, args...)
	assert.NilError(t, err, infoMsg)

	return outputBuffer
}

func executeCmdWithTimeOutNilAssertion(
	t *testing.T,
	infoMsg string,
	timeout time.Duration,
	args ...string,
) *bytes.Buffer {
	cmd, outputBuffer := createRedirectedTestCommand(t)
	err := executeWithTimeout(cmd, timeout, args...)
	assert.NilError(t, err, infoMsg)

	return outputBuffer
}

func executeWithTimeout(cmd *cobra.Command, timeout time.Duration, args ...string) error {

	args = append(args, flag(params.RetryFlag), "3", flag(params.RetryDelayFlag), "5")
	args = appendProxyArgs(args)
	cmd.SetArgs(args)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	execChannel := make(chan error)
	go func() {
		execChannel <- cmd.ExecuteContext(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-execChannel:
		return result
	}
}

func appendProxyArgs(args []string) []string {
	argsWithProxy := append(args, flag(params.ProxyFlag))
	argsWithProxy = append(argsWithProxy, buildProxyURL())
	return argsWithProxy
}

func buildProxyURL() string {
	proxyUser := viper.GetString(ProxyUserEnv)
	proxyPw := viper.GetString(ProxyPwEnv)
	proxyPort := viper.GetInt(ProxyPortEnv)
	proxyHost := viper.GetString(ProxyHostEnv)
	proxyURL := fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyPw, proxyHost, proxyPort)
	return proxyURL
}

// Assert error with expected message
func assertError(t *testing.T, err error, expectedMessage string) {
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), strings.ToLower(expectedMessage)))
}

// Assert error due missing a mandatory parameter
func assertRequiredParameter(t *testing.T, expectedMessage string, args ...string) {
	err := execute(createASTIntegrationTestCommand(t), args...)
	assertError(t, err, expectedMessage)
}
