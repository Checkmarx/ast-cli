// +build integration

package integration

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/checkmarxDev/ast-cli/internal/commands"

	params "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func bindKeysToEnvAndDefault(t *testing.T) {
	for _, b := range params.EnvVarsBinds {
		err := viper.BindEnv(b.Key, b.Env)
		assert.NilError(t, err)
		viper.SetDefault(b.Key, b.Default)
	}
}

func RandomizeString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[seededRand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	fmt.Println("Starting a test!")
	// Run all tests
	exitVal := m.Run()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

func createASTIntegrationTestCommand(t *testing.T) *cobra.Command {
	bindKeysToEnvAndDefault(t)
	scans := viper.GetString(params.ScansPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
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
	createClientPath := viper.GetString(params.CreateOath2ClientPathKey)
	sastScanInc := viper.GetString(params.SastScanIncPathKey)
	sastScanIncEngineLogPath := viper.GetString(params.SastScanIncEngineLogPathKey)
	sastScanIncMetricsPath := viper.GetString(params.SastScanIncMetricsPathKey)
	logs := viper.GetString(params.LogsPathKey)

	// Tests variables
	viper.SetDefault("TEST_FULL_SCAN_WAIT_COMPLETED_SECONDS", 400)
	viper.SetDefault("TEST_INC_SCAN_WAIT_COMPLETED_SECONDS", 60)

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
	queriesWrapper := wrappers.NewQueriesHTTPWrapper(queries, fmt.Sprintf("%s/%s", queries, queriesClone))
	authWrapper := wrappers.NewAuthHTTPWrapper(createClientPath)
	sastMetadataWrapper := wrappers.NewSastMetadataHTTPWrapper(
		sastScanInc,
		fmt.Sprintf("%s/%s", sastScanInc, sastScanIncEngineLogPath),
		fmt.Sprintf("%s/%s", sastScanInc, sastScanIncMetricsPath),
	)
	logsWrapper := wrappers.NewLogsWrapper(logs)

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

func execute(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}
