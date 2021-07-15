package main

import (
	"fmt"
	"os"

	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	params "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
)

const (
	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	bindKeysToEnvAndDefault()
	wrappers.LoadConfiguration()
	scans := viper.GetString(params.ScansPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	sastResults := viper.GetString(params.SastResultsPathKey)
	kicsResults := viper.GetString(params.KicsResultsPathKey)
	bfl := viper.GetString(params.BflPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	sastrm := viper.GetString(params.SastRmPathEnv)
	webAppHlthChk := viper.GetString(params.AstWebAppHealthCheckPathKey)
	keyCloakWebAppHlthChk := viper.GetString(params.AstKeycloakWebAppHealthCheckPathKey)
	healthcheck := viper.GetString(params.HealthcheckPathKey)
	healthcheckDBPath := viper.GetString(params.HealthcheckDBPathKey)
	healthcheckMessageQueuePath := viper.GetString(params.HealthcheckMessageQueuePathKey)
	healthcheckObjectStorePath := viper.GetString(params.HealthcheckObjectStorePathKey)
	healthcheckInMemoryDBPath := viper.GetString(params.HealthcheckInMemoryDBPathKey)
	healthcheckLoggingPath := viper.GetString(params.HealthcheckLoggingPathKey)
	healthcheckScanFlowPath := viper.GetString(params.HealthcheckScanFlowPathKey)
	healthcheckSastEnginesPath := viper.GetString(params.HealthcheckSastEnginesPathKey)
	queries := viper.GetString(params.QueriesPathKey)
	queriesClonePath := viper.GetString(params.QueriesClonePathKey)
	sastMetadata := viper.GetString(params.SastMetadataPathKey)
	sastMetadataMetricsPath := viper.GetString(params.SastMetadataMetricsPathKey)
	logs := viper.GetString(params.LogsPathKey)
	logsEngineLogPath := viper.GetString(params.LogsEngineLogPathKey)
	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, sastResults, kicsResults, scans)
	bflWrapper := wrappers.NewHTTPBFLWrapper(bfl)
	rmWrapper := wrappers.NewSastRmHTTPWrapper(sastrm)
	healthCheckWrapper := wrappers.NewHealthCheckHTTPWrapper(
		webAppHlthChk,
		keyCloakWebAppHlthChk,
		fmt.Sprintf("%s/%s", healthcheck, healthcheckDBPath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckMessageQueuePath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckObjectStorePath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckInMemoryDBPath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckLoggingPath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckScanFlowPath),
		fmt.Sprintf("%s/%s", healthcheck, healthcheckSastEnginesPath),
	)
	queriesCloneURIPath := fmt.Sprintf("%s/%s", queries, queriesClonePath)
	queriesWrapper := wrappers.NewQueriesHTTPWrapper(queries, queriesCloneURIPath)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	sastMetadataWrapper := wrappers.NewSastMetadataHTTPWrapper(sastMetadata,
		fmt.Sprintf("%s/%s", sastMetadata, sastMetadataMetricsPath),
	)
	logsWrapper := wrappers.NewLogsWrapper(logs,
		fmt.Sprintf("%s/%s", logs, logsEngineLogPath))
	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		bflWrapper,
		rmWrapper,
		healthCheckWrapper,
		queriesWrapper,
		authWrapper,
		sastMetadataWrapper,
		logsWrapper,
	)
	err := astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		os.Exit(failureExitCode)
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
