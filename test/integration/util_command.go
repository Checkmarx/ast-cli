// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
	"testing"
	"time"
)

//goland:noinspection HttpUrlsUsage
const (
	ProxyUserEnv = "PROXY_USERNAME"
	ProxyPwEnv   = "PROXY_PASSWORD"
	ProxyURLTmpl = "http://%s:%s@localhost:3128"
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
	viper.AutomaticEnv()
	scans := viper.GetString(params.ScansPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	sastResults := viper.GetString(params.SastResultsPathKey)
	kicsResults := viper.GetString(params.KicsResultsPathKey)
	bfl := viper.GetString(params.BflPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	sastrm := viper.GetString(params.SastRmPathKey)
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
	queriesClone := viper.GetString(params.QueriesClonePathKey)
	sastScanInc := viper.GetString(params.SastMetadataPathKey)
	sastScanIncMetricsPath := viper.GetString(params.SastMetadataMetricsPathKey)
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
	queriesWrapper := wrappers.NewQueriesHTTPWrapper(queries, fmt.Sprintf("%s/%s", queries, queriesClone))
	authWrapper := wrappers.NewAuthHTTPWrapper()
	sastMetadataWrapper := wrappers.NewSastMetadataHTTPWrapper(sastScanInc,
		fmt.Sprintf("%s/%s", sastScanInc, sastScanIncMetricsPath),
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

func execute(cmd *cobra.Command, args ...string) error {
	return executeWithTimeout(cmd, time.Minute, args...)
}

func executeWithTimeout(cmd *cobra.Command, timeout time.Duration, args ...string) error {
	proxyUser := viper.GetString(ProxyUserEnv)
	proxyPw := viper.GetString(ProxyPwEnv)

	args = append(args, flag(commands.VerboseFlag))
	args = append(args, flag(commands.ProxyFlag))
	args = append(args, fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyPw))
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
