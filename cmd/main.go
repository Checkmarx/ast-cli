package main

import (
	"fmt"
	"os"

	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/params"
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
	uploads := viper.GetString(params.UploadsPathKey)
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
	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, sastResults, kicsResults, scans)
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
	authWrapper := wrappers.NewAuthHTTPWrapper()
	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		healthCheckWrapper,
		authWrapper,
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
