package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/viper"
)

const (
	successfulExitCode = 0
	failureExitCode    = 1
	killCommand        = "kill"
)

func main() {
	var err error
	bindProxy()
	bindKeysToEnvAndDefault()
	err = configuration.LoadConfiguration()
	exitIfError(err)
	scans := viper.GetString(params.ScansPathKey)
	groups := viper.GetString(params.GroupsPathKey)
	logs := viper.GetString(params.LogsPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	applications := viper.GetString(params.ApplicationsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	scanSummary := viper.GetString(params.ScanSummaryPathKey)
	risksOverview := viper.GetString(params.RisksOverviewPathKey)
	riskManagement := viper.GetString(params.RiskManagementPathKey)
	scsScanOverview := viper.GetString(params.ScsScanOverviewPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	codebashing := viper.GetString(params.CodeBashingPathKey)
	bfl := viper.GetString(params.BflPathKey)
	prDecorationGithubPath := viper.GetString(params.PRDecorationGithubPathKey)
	prDecorationGitlabPath := viper.GetString(params.PRDecorationGitlabPathKey)
	bitbucketServerPath := viper.GetString(params.PRDecorationBitbucketServerPathKey)
	bitbucketCloudPath := viper.GetString(params.PRDecorationBitbucketCloudPathKey)
	prDecorationAzurePath := viper.GetString(params.PRDecorationAzurePathKey)
	descriptionsPath := viper.GetString(params.DescriptionsPathKey)
	tenantConfigurationPath := viper.GetString(params.TenantConfigurationPathKey)
	resultsPdfPath := viper.GetString(params.ResultsPdfReportPathKey)
	resultsJSONPath := viper.GetString(params.ResultsJSONReportPathKey)
	exportPath := viper.GetString(params.ExportPathKey)
	featureFlagsPath := viper.GetString(params.FeatureFlagsKey)
	policyEvaluationPath := viper.GetString(params.PolicyEvaluationPathKey)
	sastMetadataPath := viper.GetString(params.SastMetadataPathKey)
	accessManagementPath := viper.GetString(params.AccessManagementPathKey)
	byorPath := viper.GetString(params.ByorPathKey)
	realtimeScannerPath := viper.GetString(params.RealtimeScannerPathKey)

	customStatesWrapper := wrappers.NewCustomStatesHTTPWrapper()
	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	resultsPdfReportsWrapper := wrappers.NewResultsPdfReportsHTTPWrapper(resultsPdfPath)
	resultsJSONReportsWrapper := wrappers.NewResultsJSONReportsHTTPWrapper(resultsJSONPath)
	exportWrapper := wrappers.NewExportHTTPWrapper(exportPath)
	groupsWrapper := wrappers.NewHTTPGroupsWrapper(groups)
	logsWrapper := wrappers.NewLogsWrapper(logs)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	applicationsWrapper := wrappers.NewApplicationsHTTPWrapper(applications)
	risksOverviewWrapper := wrappers.NewHTTPRisksOverviewWrapper(risksOverview)
	riskManagementWrapper := wrappers.NewHTTPRiskManagementWrapper(riskManagement)
	scsScanOverviewWrapper := wrappers.NewHTTPScanOverviewWrapper(scsScanOverview)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, scanSummary)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	resultsPredicatesWrapper := wrappers.NewResultsPredicatesHTTPWrapper()
	codeBashingWrapper := wrappers.NewCodeBashingHTTPWrapper(codebashing)
	gitHubWrapper := wrappers.NewGitHubWrapper()
	azureWrapper := wrappers.NewAzureWrapper()
	bitBucketWrapper := wrappers.NewBitbucketWrapper()
	bitBucketServerWrapper := bitbucketserver.NewBitbucketServerWrapper()
	gitLabWrapper := wrappers.NewGitLabWrapper()
	bflWrapper := wrappers.NewBflHTTPWrapper(bfl)
	prWrapper := wrappers.NewHTTPPRWrapper(prDecorationGithubPath, prDecorationGitlabPath, bitbucketCloudPath, bitbucketServerPath, prDecorationAzurePath)
	learnMoreWrapper := wrappers.NewHTTPLearnMoreWrapper(descriptionsPath)
	tenantConfigurationWrapper := wrappers.NewHTTPTenantConfigurationWrapper(tenantConfigurationPath)
	jwtWrapper := wrappers.NewJwtWrapper()
	scaRealTimeWrapper := wrappers.NewHTTPScaRealTimeWrapper()
	chatWrapper := wrappers.NewChatWrapper()
	featureFlagsWrapper := wrappers.NewFeatureFlagsHTTPWrapper(featureFlagsPath)
	policyWrapper := wrappers.NewHTTPPolicyWrapper(policyEvaluationPath, featureFlagsWrapper)
	sastMetadataWrapper := wrappers.NewSastIncrementalHTTPWrapper(sastMetadataPath)
	accessManagementWrapper := wrappers.NewAccessManagementHTTPWrapper(accessManagementPath)
	byorWrapper := wrappers.NewByorHTTPWrapper(byorPath)
	containerResolverWrapper := wrappers.NewContainerResolverWrapper()
	realTimeWrapper := wrappers.NewRealtimeScannerHTTPWrapper(realtimeScannerPath, jwtWrapper, featureFlagsWrapper)
	telemetryWrapper := wrappers.NewHTTPTelemetryAIWrapper(realtimeScannerPath)

	astCli := commands.NewAstCLI(
		applicationsWrapper,
		scansWrapper,
		exportWrapper,
		resultsPdfReportsWrapper,
		resultsJSONReportsWrapper,
		resultsPredicatesWrapper,
		customStatesWrapper,
		codeBashingWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
		riskManagementWrapper,
		scsScanOverviewWrapper,
		authWrapper,
		logsWrapper,
		groupsWrapper,
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		bitBucketServerWrapper,
		gitLabWrapper,
		bflWrapper,
		prWrapper,
		learnMoreWrapper,
		tenantConfigurationWrapper,
		jwtWrapper,
		scaRealTimeWrapper,
		chatWrapper,
		featureFlagsWrapper,
		policyWrapper,
		sastMetadataWrapper,
		accessManagementWrapper,
		byorWrapper,
		containerResolverWrapper,
		realTimeWrapper,
		telemetryWrapper,
	)
	exitListener()
	err = astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		switch e := err.(type) {
		case *wrappers.AstError:
			fmt.Println(e.Err)
			os.Exit(e.Code)
		default:
			fmt.Println(e)
			os.Exit(failureExitCode)
		}
	}
}

func bindKeysToEnvAndDefault() {
	for _, b := range params.EnvVarsBinds {
		err := viper.BindEnv(b.Key, b.Env)
		if err != nil {
			exitIfError(err)
		}
		viper.SetDefault(b.Key, b.Default)
	}
}

func bindProxy() {
	err := viper.BindEnv(params.ProxyKey, params.CxProxyEnv, params.ProxyEnv)
	if err != nil {
		exitIfError(err)
	}
	viper.SetDefault(params.ProxyKey, "")
	err = os.Setenv(params.ProxyEnv, viper.GetString(params.ProxyKey))
	if err != nil {
		exitIfError(err)
	}
}

func exitListener() {
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(
		signalChanel,
		syscall.SIGTERM,
	)
	go signalHandler(signalChanel)
}

func signalHandler(signalChanel chan os.Signal) {
	kicsRunArgs := []string{
		killCommand,
		viper.GetString(params.KicsContainerNameKey),
	}
	for {
		s := <-signalChanel
		switch s {
		case syscall.SIGTERM:
			out, err := exec.Command("docker", "ps").CombinedOutput()
			if err != nil {
				os.Exit(failureExitCode)
			}
			logger.PrintIfVerbose(string(out))
			if strings.Contains(string(out), viper.GetString(params.KicsContainerNameKey)) {
				out, err = exec.Command("docker", kicsRunArgs...).CombinedOutput()
				logger.PrintIfVerbose(string(out))
				if err != nil {
					os.Exit(failureExitCode)
				}
			}
			os.Exit(successfulExitCode)
		// Should not get here since we only listen to SIGTERM, kept for safety
		default:
			os.Exit(failureExitCode)
		}
	}
}
