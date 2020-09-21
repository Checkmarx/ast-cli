package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	params "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const (
	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	err := bindKeyToEnvAndDefault(params.BaseURIKey, params.BaseURIEnv, "http://127.0.0.1:80")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.ScansPathKey, params.ScansPathEnv, "api/scans")
	exitIfError(err)
	scans := viper.GetString(params.ScansPathKey)

	err = bindKeyToEnvAndDefault(params.ProjectsPathKey, params.ProjectsPathEnv, "api/projects")
	exitIfError(err)
	projects := viper.GetString(params.ProjectsPathKey)

	err = bindKeyToEnvAndDefault(params.ResultsPathKey, params.ResultsPathEnv, "api/results")
	exitIfError(err)
	results := viper.GetString(params.ResultsPathKey)

	err = bindKeyToEnvAndDefault(params.BflPathKey, params.BflPathEnv, "api/bfl")
	exitIfError(err)
	bfl := viper.GetString(params.BflPathKey)

	err = bindKeyToEnvAndDefault(params.UploadsPathKey, params.UploadsPathEnv, "api/uploads")
	exitIfError(err)
	uploads := viper.GetString(params.UploadsPathKey)

	sastRmPathKey := strings.ToLower(params.SastRmPathEnv)
	err = bindKeyToEnvAndDefault(sastRmPathKey, params.SastRmPathEnv, "api/sast-rm")
	exitIfError(err)
	sastrm := viper.GetString(sastRmPathKey)

	err = bindKeyToEnvAndDefault(params.AstWebAppHealthCheckPathKey, params.AstWebAppHealthCheckPathEnv, "#/projects")
	exitIfError(err)
	webAppHlthChk := viper.GetString(params.AstWebAppHealthCheckPathKey)

	err = bindKeyToEnvAndDefault(params.AstKeycloakWebAppHealthCheckPathKey, params.AstKeycloakWebAppHealthCheckPathEnv,
		"auth")
	exitIfError(err)
	keyCloakWebAppHlthChk := viper.GetString(params.AstKeycloakWebAppHealthCheckPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckPathKey, params.HealthcheckPathEnv, "api/healthcheck")
	exitIfError(err)
	healthcheck := viper.GetString(params.HealthcheckPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckDBPathKey, params.HealthcheckDBPathEnv, "database")
	exitIfError(err)
	healthcheckDBPath := viper.GetString(params.HealthcheckDBPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckMessageQueuePathKey,
		params.HealthcheckMessageQueuePathEnv, "message-queue")
	exitIfError(err)
	healthcheckMessageQueuePath := viper.GetString(params.HealthcheckMessageQueuePathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckObjectStorePathKey,
		params.HealthcheckObjectStorePathEnv, "object-store")
	exitIfError(err)
	healthcheckObjectStorePath := viper.GetString(params.HealthcheckObjectStorePathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckInMemoryDBPathKey,
		params.HealthcheckInMemoryDBPathEnv, "in-memory-db")
	exitIfError(err)
	healthcheckInMemoryDBPath := viper.GetString(params.HealthcheckInMemoryDBPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckLoggingPathKey,
		params.HealthcheckLoggingPathEnv, "logging")
	exitIfError(err)
	healthcheckLoggingPath := viper.GetString(params.HealthcheckLoggingPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckScanFlowPathKey, params.HealthcheckScanFlowPathEnv,
		"scan-flow")
	exitIfError(err)
	healthcheckScanFlowPath := viper.GetString(params.HealthcheckScanFlowPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckSastEnginesPathKey, params.HealthcheckSastEnginesPathEnv,
		"sast-engines")
	exitIfError(err)
	healthcheckSastEnginesPath := viper.GetString(params.HealthcheckSastEnginesPathEnv)

	err = bindKeyToEnvAndDefault(params.QueriesPathKey, params.QueriesPathEnv, "api/queries")
	exitIfError(err)
	queries := viper.GetString(params.QueriesPathKey)

	err = bindKeyToEnvAndDefault(params.QueriesClonePathKey, params.QueriesCLonePathEnv, "clone")
	exitIfError(err)
	queriesClonePath := viper.GetString(params.QueriesClonePathKey)

	err = bindKeyToEnvAndDefault(params.AccessKeyIDConfigKey, params.AccessKeyIDEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.AccessKeySecretConfigKey, params.AccessKeySecretEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.AstAuthenticationURIConfigKey, params.AstAuthenticationURIEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.AstRoleKey, params.AstRoleEnv, params.ScaAgent)
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.CredentialsFilePathKey, params.CredentialsFilePathEnv, "credentials.json")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.TokenExpirySecondsKey, params.TokenExpirySecondsEnv, "300")
	exitIfError(err)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results)
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
	defaultConfigFileLocation := "/etc/conf/cx/config.yml"

	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		bflWrapper,
		rmWrapper,
		healthCheckWrapper,
		queriesWrapper,
		defaultConfigFileLocation,
	)

	err = astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		os.Exit(failureExitCode)
	}
}

func bindKeyToEnvAndDefault(key, env, defaultVal string) error {
	err := viper.BindEnv(key, env)
	viper.SetDefault(key, defaultVal)
	return err
}
