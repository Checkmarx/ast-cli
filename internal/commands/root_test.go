package commands

import (
	"fmt"
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
	scansMockWrapper := &mock.ScansMockWrapper{}
	scansMockWrapper.Running = true
	resultsPredicatesMockWrapper := &mock.ResultsPredicatesMockWrapper{}
	groupsMockWrapper := &mock.GroupsMockWrapper{}
	uploadsMockWrapper := &mock.UploadsMockWrapper{}
	projectsMockWrapper := &mock.ProjectsMockWrapper{}
	resultsMockWrapper := &mock.ResultsMockWrapper{}
	risksOverviewMockWrapper := &mock.RisksOverviewMockWrapper{}
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

	return NewAstCLI(
		scansMockWrapper,
		resultsPredicatesMockWrapper,
		codeBashingWrapper,
		uploadsMockWrapper,
		projectsMockWrapper,
		resultsMockWrapper,
		risksOverviewMockWrapper,
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
