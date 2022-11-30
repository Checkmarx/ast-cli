//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
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

// Create a command to execute in tests
func createASTIntegrationTestCommand(t *testing.T) *cobra.Command {
	bindKeysToEnvAndDefault(t)
	_ = viper.BindEnv(pat)
	viper.AutomaticEnv()
	viper.Set("CX_TOKEN_EXPIRY_SECONDS", 2)
	scans := viper.GetString(params.ScansPathKey)
	groups := viper.GetString(params.GroupsPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	scaPackage := viper.GetString(params.ScaPackagePathKey)
	risksOverview := viper.GetString(params.RisksOverviewPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	logs := viper.GetString(params.LogsPathKey)
	codebashing := viper.GetString(params.CodeBashingPathKey)
	bfl := viper.GetString(params.BflPathKey)
	learnMore := viper.GetString(params.DescriptionsPathKey)
	prDecorationGithubPath := viper.GetString(params.PRDecorationGithubPathKey)
	tenantConfigurationPath := viper.GetString(params.TenantConfigurationPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	resultsPredicatesWrapper := wrappers.NewResultsPredicatesHTTPWrapper()
	groupsWrapper := wrappers.NewHTTPGroupsWrapper(groups)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, scaPackage)
	risksOverviewWrapper := wrappers.NewHTTPRisksOverviewWrapper(risksOverview)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	logsWrapper := wrappers.NewLogsWrapper(logs)
	codeBashingWrapper := wrappers.NewCodeBashingHTTPWrapper(codebashing)
	gitHubWrapper := wrappers.NewGitHubWrapper()
	gitLabWrapper := wrappers.NewGitLabWrapper()
	azureWrapper := wrappers.NewAzureWrapper()
	bitBucketWrapper := wrappers.NewBitbucketWrapper()
	bflWrapper := wrappers.NewBflHTTPWrapper(bfl)
	learnMoreWrapper := wrappers.NewHTTPLearnMoreWrapper(learnMore)
	prWrapper := wrappers.NewHTTPPRWrapper(prDecorationGithubPath)
	tenantConfigurationWrapper := wrappers.NewHTTPTenantConfigurationWrapper(tenantConfigurationPath)

	astCli := commands.NewAstCLI(
		scansWrapper,
		resultsPredicatesWrapper,
		codeBashingWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
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

/*  Execute a previous created command. Used when there is a need to make changes in environment variables

Ex.: 1. Create a command
	 2. Make changes in environment variables using viper
	 3. Execute command
*/
func execute(cmd *cobra.Command, args ...string) error {
	return executeWithTimeout(cmd, time.Minute, args...)
}

// Execute a CLI command expecting an error and buffer to execute post assertions
func executeCommand(t *testing.T, args ...string) (error, *bytes.Buffer) {

	cmd, buffer := createRedirectedTestCommand(t)

	err := executeWithTimeout(cmd, time.Minute, args...)

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

	args = append(args, flag(params.DebugFlag), flag(params.RetryFlag), "3", flag(params.RetryDelayFlag), "5")
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
	proxyUser := viper.GetString(ProxyUserEnv)
	proxyPw := viper.GetString(ProxyPwEnv)
	proxyPort := viper.GetInt(ProxyPortEnv)
	proxyHost := viper.GetString(ProxyHostEnv)
	argsWithProxy := append(args, flag(params.ProxyFlag))
	argsWithProxy = append(argsWithProxy, fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyPw, proxyHost, proxyPort))
	return argsWithProxy
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
