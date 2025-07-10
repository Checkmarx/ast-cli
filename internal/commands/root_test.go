package commands

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

const (
	resolverEnvVar        = "SCA_RESOLVER"
	resolverEnvVarDefault = "./ScaResolver"
	githubDummyRepo       = "https://github.com/dummyuser/dummy_project.git"
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
	exportWrapper := &mock.ExportMockWrapper{}
	resultsPdfWrapper := &mock.ResultsPdfWrapper{}
	resultsJSONWrapper := &mock.ResultsJSONWrapper{}
	scansMockWrapper.Running = true
	resultsPredicatesMockWrapper := &mock.ResultsPredicatesMockWrapper{}
	groupsMockWrapper := &mock.GroupsMockWrapper{}
	uploadsMockWrapper := &mock.UploadsMockWrapper{}
	projectsMockWrapper := &mock.ProjectsMockWrapper{}
	resultsMockWrapper := &mock.ResultsMockWrapper{}
	risksOverviewMockWrapper := &mock.RisksOverviewMockWrapper{}
	riskManagementMockWrapper := &mock.RiskManagementMockWrapper{}
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
	byorWrapper := &mock.ByorMockWrapper{}
	containerResolverMockWrapper := &mock.ContainerResolverMockWrapper{}
	customStatesMockWrapper := &mock.CustomStatesMockWrapper{}
	realTimeWrapper := &mock.RealtimeScannerMockWrapper{}
	return NewAstCLI(
		applicationWrapper,
		scansMockWrapper,
		exportWrapper,
		resultsPdfWrapper,
		resultsJSONWrapper,
		resultsPredicatesMockWrapper,
		customStatesMockWrapper,
		codeBashingWrapper,
		uploadsMockWrapper,
		projectsMockWrapper,
		resultsMockWrapper,
		risksOverviewMockWrapper,
		riskManagementMockWrapper,
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
		byorWrapper,
		containerResolverMockWrapper,
		realTimeWrapper,
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

func TestFilterTag(t *testing.T) {
	stateValues := "state=exclude_not_exploitable"
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", githubDummyRepo, "-b", "dummy_branch", "--filter", stateValues}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestFilterTagAllStateValues(t *testing.T) {
	stateValues := "state=exclude_not_exploitable;TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED;URGENT"
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", githubDummyRepo, "-b", "dummy_branch", "--filter", stateValues}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestFilterTagStateAndSeverityValues(t *testing.T) {
	stateAndSeverityValues := "state=exclude_not_exploitable;TO_VERIFY,severity=High"
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", githubDummyRepo, "-b", "dummy_branch", "--filter", stateAndSeverityValues}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

func TestCreateCommand_WithInvalidFlag_ShouldReturnExitCode1(t *testing.T) {
	args := []string{"g"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, args...)
	assert.Error(t, err, "unknown command \"g\" for \"cx\"")
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

func clearFlags() {
	mock.Flags = wrappers.FeatureFlagsResponseModel{}
	mock.Flag = wrappers.FeatureFlagResponseModel{}
	mock.FFErr = nil
	wrappers.ClearCache()
}

func Test_stateExclude_not_exploitableRepalceForAllStatesExceptNot_exploitable(t *testing.T) {
	type args struct {
		filterKeyVal []string
	}
	tests := []struct {
		filterName      string
		extraFilterName args
		replaceValue    []string
		expectedLog     string
	}{
		{
			filterName:      "State has some values and exclude-not-exploitable",
			extraFilterName: args{filterKeyVal: []string{"state", "exclude_not_exploitable;TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED"}},
			replaceValue:    []string{"state", "TO_VERIFY;PROPOSED_NOT_EXPLOITABLE;CONFIRMED;URGENT"},
			expectedLog:     "",
		},
		{
			filterName:      "State has no values ",
			extraFilterName: args{filterKeyVal: []string{"state", ""}},
			replaceValue:    []string{"state", ""},
			expectedLog:     "",
		},
		{
			filterName:      "State has Only exclude-not-exploitable value",
			extraFilterName: args{filterKeyVal: []string{"state", "exclude_not_exploitable"}},
			replaceValue:    []string{"state", ""},
			expectedLog:     "",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.filterName, func(t *testing.T) { // Reset the log output for each test
			if got := validateExtraFilters(test.extraFilterName.filterKeyVal); !reflect.DeepEqual(got, test.replaceValue) {
				assert.Assert(t, strings.Contains(strings.ToLower(got[1]), strings.ToLower(test.replaceValue[1])))
			}
		})
	}
}
func TestSetLogOutputFromFlag_InvalidDir1(t *testing.T) {
	err := setLogOutputFromFlag(params.LogFileFlag, "/custom/path")
	assert.ErrorContains(t, err, "the specified directory path does not exist.")
}

func TestSetLogOutputFromFlag_EmptyDirPath(t *testing.T) {
	err := setLogOutputFromFlag(params.LogFileFlag, "")
	assert.ErrorContains(t, err, "flag needs an argument")
}

func TestSetLogOutputFromFlag_DirPathIsFilePath(t *testing.T) {
	tempFile, _ := os.CreateTemp("", "ast-cli.txt")
	defer func() {
		if err := os.RemoveAll(tempFile.Name()); err != nil {
			fmt.Printf("Warning: failed to clean up temp directory %s: %v\n", tempFile.Name(), err)
		}
	}()
	err := setLogOutputFromFlag(params.LogFileFlag, tempFile.Name())
	assert.ErrorContains(t, err, "expected a directory path but got a file")
}

func TestSetLogOutputFromFlag_DirPathPermissionDenied(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "tempdir")
	_ = os.Chmod(tempDir, 0000)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	err := setLogOutputFromFlag(params.LogFileFlag, tempDir)
	assert.ErrorContains(t, err, "permission denied: cannot write to directory")
}

func TestSetLogOutputFromFlag_DirPath_Success(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "tempdir")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: failed to clean up temp directory %s: %v\n", tempDir, err)
		}
	}()
	err := setLogOutputFromFlag(params.LogFileFlag, tempDir)
	assert.NilError(t, err)
}

func TestSetLogOutputFromFlag_DirPath_Console_Success(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "tempdir")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: failed to clean up temp directory %s: %v\n", tempDir, err)
		}
	}()
	err := setLogOutputFromFlag(params.LogFileConsoleFlag, tempDir)
	assert.NilError(t, err)
}
