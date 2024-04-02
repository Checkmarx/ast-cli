package commands

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

const (
	resolverEnvVar        = "SCA_RESOLVER"
	resolverEnvVarDefault = "./ScaResolver"
)

func TestMain(m *testing.M) {
	log.Println("Commands tests started")
	// Run all tests
	exitVal := m.Run()
	viper.SetDefault(resolverEnvVar, resolverEnvVarDefault)
	log.Println("Commands tests done")
	os.Exit(exitVal)
}

func createASTTestCommand() *cobra.Command {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	scansMockWrapper := &mock.ScansMockWrapper{}
	resultsSbomWrapper := &mock.ResultsSbomWrapper{}
	resultsPdfWrapper := &mock.ResultsPdfWrapper{}
	scansMockWrapper.Running = true
	resultsPredicatesMockWrapper := &mock.ResultsPredicatesMockWrapper{}
	groupsMockWrapper := &mock.GroupsMockWrapper{}
	uploadsMockWrapper := &mock.UploadsMockWrapper{}
	projectsMockWrapper := &mock.ProjectsMockWrapper{}
	resultsMockWrapper := &mock.ResultsMockWrapper{}
	risksOverviewMockWrapper := &mock.RisksOverviewMockWrapper{}
	scsScanOverviewMockWrapper := &mock.ScanOverviewMockWrapper{}
	authWrapper := &mock.AuthMockWrapper{}
	logsWrapper := &mock.LogsMockWrapper{}
	codeBashingWrapper := &mock.CodeBashingMockWrapper{}
	gitHubWrapper := &mock.GitHubMockWrapper{}
	azureWrapper := &mock.AzureMockWrapper{}
	bitBucketWrapper := &mock.BitBucketMockWrapper{}
	gitLabWrapper := &mock.GitLabMockWrapper{}
	bflMockWrapper := &mock.BflMockWrapper{}
	learnMoreMockWrapper := &mock.LearnMoreMockWrapper{}
	prMockWrapper := &mock.PRMockWrapper{}
	tenantConfigurationMockWrapper := &mock.TenantConfigurationMockWrapper{}
	jwtWrapper := &mock.JWTMockWrapper{}
	scaRealtimeMockWrapper := &mock.ScaRealTimeHTTPMockWrapper{}
	chatWrapper := &mock.ChatMockWrapper{}
	featureFlagsMockWrapper := &mock.FeatureFlagsMockWrapper{}
	policyWrapper := &mock.PolicyMockWrapper{}
	sastMetadataWrapper := &mock.SastMetadataMockWrapper{}
	accessManagementWrapper := &mock.AccessManagementMockWrapper{}

	return NewAstCLI(
		applicationWrapper,
		scansMockWrapper,
		resultsSbomWrapper,
		resultsPdfWrapper,
		resultsPredicatesMockWrapper,
		codeBashingWrapper,
		uploadsMockWrapper,
		projectsMockWrapper,
		resultsMockWrapper,
		risksOverviewMockWrapper,
		scsScanOverviewMockWrapper,
		authWrapper,
		logsWrapper,
		groupsMockWrapper,
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		nil,
		gitLabWrapper,
		bflMockWrapper,
		prMockWrapper,
		learnMoreMockWrapper,
		tenantConfigurationMockWrapper,
		jwtWrapper,
		scaRealtimeMockWrapper,
		chatWrapper,
		featureFlagsMockWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
	)
}

func createASTTestCommandWithScs(scsScanned, scsScanPartial, scorecardScanned bool) *cobra.Command {
	applicationWrapper := &mock.ApplicationsMockWrapper{}
	scansMockWrapper := &mock.ScansMockWrapper{HasSCS: scsScanned}
	resultsSbomWrapper := &mock.ResultsSbomWrapper{}
	resultsPdfWrapper := &mock.ResultsPdfWrapper{}
	scansMockWrapper.Running = true
	resultsPredicatesMockWrapper := &mock.ResultsPredicatesMockWrapper{}
	groupsMockWrapper := &mock.GroupsMockWrapper{}
	uploadsMockWrapper := &mock.UploadsMockWrapper{}
	projectsMockWrapper := &mock.ProjectsMockWrapper{}
	resultsMockWrapper := &mock.ResultsMockWrapper{}
	risksOverviewMockWrapper := &mock.RisksOverviewMockWrapper{}
	scsScanOverviewMockWrapper := &mock.ScanOverviewMockWrapper{ScanPartial: scsScanPartial, ScorecardScanned: scorecardScanned}
	authWrapper := &mock.AuthMockWrapper{}
	logsWrapper := &mock.LogsMockWrapper{}
	codeBashingWrapper := &mock.CodeBashingMockWrapper{}
	gitHubWrapper := &mock.GitHubMockWrapper{}
	azureWrapper := &mock.AzureMockWrapper{}
	bitBucketWrapper := &mock.BitBucketMockWrapper{}
	gitLabWrapper := &mock.GitLabMockWrapper{}
	bflMockWrapper := &mock.BflMockWrapper{}
	learnMoreMockWrapper := &mock.LearnMoreMockWrapper{}
	prMockWrapper := &mock.PRMockWrapper{}
	tenantConfigurationMockWrapper := &mock.TenantConfigurationMockWrapper{}
	jwtWrapper := &mock.JWTMockWrapper{}
	scaRealtimeMockWrapper := &mock.ScaRealTimeHTTPMockWrapper{}
	chatWrapper := &mock.ChatMockWrapper{}
	featureFlagsMockWrapper := &mock.FeatureFlagsMockWrapper{}
	policyWrapper := &mock.PolicyMockWrapper{}
	sastMetadataWrapper := &mock.SastMetadataMockWrapper{}
	accessManagementWrapper := &mock.AccessManagementMockWrapper{}

	return NewAstCLI(
		applicationWrapper,
		scansMockWrapper,
		resultsSbomWrapper,
		resultsPdfWrapper,
		resultsPredicatesMockWrapper,
		codeBashingWrapper,
		uploadsMockWrapper,
		projectsMockWrapper,
		resultsMockWrapper,
		risksOverviewMockWrapper,
		scsScanOverviewMockWrapper,
		authWrapper,
		logsWrapper,
		groupsMockWrapper,
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		nil,
		gitLabWrapper,
		bflMockWrapper,
		prMockWrapper,
		learnMoreMockWrapper,
		tenantConfigurationMockWrapper,
		jwtWrapper,
		scaRealtimeMockWrapper,
		chatWrapper,
		featureFlagsMockWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
	)
}

func TestRootHelp(t *testing.T) {
	cmd := createASTTestCommand()
	args := "--help"
	err := executeTestCommand(cmd, args)
	assert.NilError(t, err)
}

func TestRootVersion(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "version")
	assert.NilError(t, err)
}

func executeTestCommand(cmd *cobra.Command, args ...string) error {
	fmt.Println("Executing command with args ", args)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	return cmd.Execute()
}

func executeRedirectedTestCommand(args ...string) (*bytes.Buffer, error) {
	buffer := bytes.NewBufferString("")
	cmd := createASTTestCommand()
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SetOut(buffer)
	return buffer, cmd.Execute()
}

func executeRedirectedOsStdoutTestCommand(cmd *cobra.Command, args ...string) (bytes.Buffer, error) {
	// Writing os stdout to file
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	err := cmd.Execute()

	// Writing output to buffer
	w.Close()
	os.Stdout = old
	var buffer bytes.Buffer
	_, errCopy := io.Copy(&buffer, r)
	if errCopy != nil {
		return buffer, errCopy
	}
	return buffer, err
}

func execCmdNilAssertion(t *testing.T, args ...string) {
	err := executeTestCommand(createASTTestCommand(), args...)
	assert.NilError(t, err)
}

func execCmdNotNilAssertion(t *testing.T, args ...string) error {
	err := executeTestCommand(createASTTestCommand(), args...)
	assert.Assert(t, err != nil)
	return err
}

func assertError(t *testing.T, err error, expectedMessage string) {
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(strings.ToLower(err.Error()), strings.ToLower(expectedMessage)))
}
