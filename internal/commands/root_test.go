package commands

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
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
	// Isolate credential storage: unit tests must never touch the real OS
	// keyring or the user's actual config file.
	testConfigDir, err := os.MkdirTemp("", "cx-test-config")
	if err != nil {
		log.Fatalf("failed to create test config dir: %v", err)
	}
	configPath := filepath.Join(testConfigDir, "checkmarxcli.yaml")
	if err := os.WriteFile(configPath, nil, 0o600); err != nil {
		log.Fatalf("failed to seed test config file: %v", err)
	}
	_ = os.Setenv(credentialstore.KeyringModeEnvVar, "disabled")
	_ = os.Setenv(params.ConfigFilePathEnv, configPath)
	credentialstore.ResetForTest()
	exitVal := m.Run()
	viper.SetDefault(resolverEnvVar, resolverEnvVarDefault)
	_ = os.RemoveAll(testConfigDir)
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
	scanSummaryMockWrapper := &mock.ScanSummaryMockWrapper{}
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
	telemetryWrapper := &mock.TelemetryMockWrapper{}
	dastEnvironmentsWrapper := &mock.DastEnvironmentsMockWrapper{}
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
		scanSummaryMockWrapper,
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
		telemetryWrapper,
		dastEnvironmentsWrapper,
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
	defer viper.Reset()
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

func TestOptionalFlag(t *testing.T) {
	baseArgs := []string{"scan", "create", "--project-name", "MOCK", "-s", githubDummyRepo, "-b", "dummy_branch", "--optional-flags", "asca-location=/optional/path"}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, baseArgs...)
	assert.NilError(t, err)
}

const (
	classicProxyAddress    = "http://classic.company.com:8080"
	invalidProxyAuthType   = "NotARealAuthType"
	invalidProxyAuthErrMsg = "invalid proxy authentication type"
	supportedProxyAuthMsg  = "basic, ntlm, kerberos, kerberos-native"
)

func TestProxyAuthType_SupportedValues_Pass(t *testing.T) {
	supported := []string{
		"basic", "ntlm", "kerberos", "kerberos-native",
		"BASIC", "NTLM", "Ntlm", "Kerberos", "KERBEROS-NATIVE",
	}
	for _, value := range supported {
		t.Run(value, func(t *testing.T) {
			execCmdNilAssertion(t, "utils", "env", "--proxy-auth-type", value)
		})
	}
}

func TestProxyAuthType_EmptyOrUnset_Pass(t *testing.T) {
	execCmdNilAssertion(t, "utils", "env")
	execCmdNilAssertion(t, "utils", "env", "--proxy-auth-type", "")
}

func TestProxyAuthType_InvalidFlagValue_Fails(t *testing.T) {
	err := execCmdNotNilAssertion(t, "utils", "env", "--proxy-auth-type", invalidProxyAuthType)
	assertError(t, err, invalidProxyAuthErrMsg)
	assertError(t, err, `"`+invalidProxyAuthType+`"`)
	assertError(t, err, supportedProxyAuthMsg)
	assertError(t, err, "--"+params.ProxyTypeFlag)
}

func TestProxyAuthType_InvalidFlagValue_WithIgnoreProxy_Fails(t *testing.T) {
	err := execCmdNotNilAssertion(t, "utils", "env", "--ignore-proxy", "--proxy-auth-type", invalidProxyAuthType)
	assertError(t, err, invalidProxyAuthErrMsg)
}

func TestProxyAuthType_ValidFlagValue_WithIgnoreProxy_Pass(t *testing.T) {
	execCmdNilAssertion(t, "utils", "env", "--ignore-proxy", "--proxy-auth-type", "NTLM")
}

func TestProxyAuthType_InvalidEnvVar_Fails(t *testing.T) {
	_ = viper.BindEnv(params.ProxyTypeKey, params.ProxyTypeEnv)
	t.Setenv(params.ProxyTypeEnv, invalidProxyAuthType)
	err := execCmdNotNilAssertion(t, "utils", "env")
	assertError(t, err, invalidProxyAuthErrMsg)
	assertError(t, err, params.ProxyTypeEnv)
}

func TestProxyAuthType_UppercaseEnvVar_Pass(t *testing.T) {
	_ = viper.BindEnv(params.ProxyTypeKey, params.ProxyTypeEnv)
	t.Setenv(params.ProxyTypeEnv, "NTLM")
	execCmdNilAssertion(t, "utils", "env")
}

func writeProxyAuthConfigFile(t *testing.T, value string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "checkmarxcli.yaml")
	content := fmt.Sprintf("%s: %s\n", params.ProxyTypeKey, value)
	assert.NilError(t, os.WriteFile(configPath, []byte(content), 0o600))
	return configPath
}

func TestProxyAuthType_InvalidConfigFileValue_Fails(t *testing.T) {
	configPath := writeProxyAuthConfigFile(t, invalidProxyAuthType)
	err := execCmdNotNilAssertion(t, "utils", "env", "--config-file-path", configPath)
	assertError(t, err, invalidProxyAuthErrMsg)
	assertError(t, err, params.ProxyTypeKey)
}

func TestProxyAuthType_UppercaseConfigFileValue_Pass(t *testing.T) {
	configPath := writeProxyAuthConfigFile(t, "KERBEROS")
	execCmdNilAssertion(t, "utils", "env", "--config-file-path", configPath)
}

func TestProxyAuthType_ProxyAddressIsNotValidated(t *testing.T) {
	addresses := []string{
		classicProxyAddress,
		"http://user:pass@proxy.example.com:3128",
		"proxy.example.com:8080",
	}
	for _, address := range addresses {
		t.Run(address, func(t *testing.T) {
			execCmdNilAssertion(t, "utils", "env", "--proxy", address)
			execCmdNilAssertion(t, "utils", "env", "--proxy", address, "--proxy-auth-type", "NTLM")
		})
	}
}

func TestProxyAuthType_InvalidTypeWithValidProxyAddress_ReportsAuthType(t *testing.T) {
	err := execCmdNotNilAssertion(t, "utils", "env", "--proxy", classicProxyAddress, "--proxy-auth-type", invalidProxyAuthType)
	assertError(t, err, invalidProxyAuthErrMsg)
	assert.Assert(t, !strings.Contains(err.Error(), "classic.company.com"), "error must not blame the proxy address")
}

func commandChain(names ...string) *cobra.Command {
	parent := &cobra.Command{Use: "cx"}
	for _, name := range names {
		child := &cobra.Command{Use: name}
		parent.AddCommand(child)
		parent = child
	}
	return parent
}

func TestSkipsProxyAuthTypeValidation(t *testing.T) {
	exempt := [][]string{
		{"configure"}, {"configure", "set"}, {"configure", "show"}, {"version"}, {"help"},
		{"completion"}, {"utils", "completion"}, {"__complete"}, {"__completeNoDesc"},
	}
	for _, names := range exempt {
		t.Run("exempt "+strings.Join(names, " "), func(t *testing.T) {
			assert.Assert(t, skipsProxyAuthTypeValidation(commandChain(names...)))
		})
	}

	validated := [][]string{
		{"scan", "create"}, {"auth", "validate"}, {"utils", "env"}, {"project", "list"}, {"results", "show"},
	}
	for _, names := range validated {
		t.Run("validated "+strings.Join(names, " "), func(t *testing.T) {
			assert.Assert(t, !skipsProxyAuthTypeValidation(commandChain(names...)))
		})
	}
}

func TestProxyAuthType_InvalidConfigFileValue_DoesNotBlockRecoveryCommands(t *testing.T) {
	configPath := writeProxyAuthConfigFile(t, invalidProxyAuthType)
	recoveryCommands := [][]string{{"version"}, {"help"}, {"configure", "show"}}
	for _, args := range recoveryCommands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			execCmdNilAssertion(t, append(args, "--config-file-path", configPath)...)
		})
	}
}

func TestProxyAuthType_InvalidEnvVar_DoesNotBlockVersion(t *testing.T) {
	_ = viper.BindEnv(params.ProxyTypeKey, params.ProxyTypeEnv)
	t.Setenv(params.ProxyTypeEnv, invalidProxyAuthType)
	execCmdNilAssertion(t, "version")
}

func TestProxyAuthType_ConfigureSetRepairsInvalidPersistedValue(t *testing.T) {
	configPath := writeProxyAuthConfigFile(t, invalidProxyAuthType)

	err := execCmdNotNilAssertion(t, "utils", "env", "--config-file-path", configPath)
	assertError(t, err, invalidProxyAuthErrMsg)

	execCmdNilAssertion(
		t, "configure", "set", "--prop-name", params.ProxyTypeKey, "--prop-value", "basic", "--config-file-path", configPath,
	)

	execCmdNilAssertion(t, "utils", "env", "--config-file-path", configPath)
}
