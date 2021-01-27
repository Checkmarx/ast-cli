// +build integration

package integration

import (
	"bytes"
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
	// TODO: remove this after token support is available.
	authASTServer()
	fmt.Println("Starting a test!")
	// Run all tests
	exitVal := m.Run()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

func createASTIntegrationTestCommand(t *testing.T) *cobra.Command {
	// TODO: remove this test for nil. It's hack until token support is in place to
	// allow authASTServer() to work because there no test.T type available.
	if t != nil {
		bindKeysToEnvAndDefault(t)
	}
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

	fmt.Println("TODO: BaseURIEnv", viper.GetString(params.BaseURIEnv))
	fmt.Println("TODO: AccessKeyIDEnv", viper.GetString(params.AccessKeyIDEnv))
	fmt.Println("TODO: AccessKeySecretEnv", viper.GetString(params.AccessKeySecretEnv))

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
	args = append(args, "--key")
	args = append(args, accessKeyID)
	args = append(args, "--secret")
	args = append(args, accessKeySecret)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// TODO: The following logic allows the integration tests to authenticat against the AST
// server. This should be removed after token support is added
var accessKeyID = ""
var accessKeySecret = ""

func authASTServer() {
	fmt.Println("Authenticating to AST Server")
	for _, b := range params.EnvVarsBinds {
		viper.BindEnv(b.Key, b.Env)
		viper.SetDefault(b.Key, b.Default)
	}
	b := bytes.NewBufferString("")
	cmd := createASTIntegrationTestCommand(nil)
	cmd.SetOut(b)
	var args = []string{"auth", "register", "-u", "org_admin", "-p", "Cx123456"}
	cmd.SetArgs(args)
	accessKeyID = "ast-plugins-ad880520-10eb-4973-ba50-0f6f31f153a6"
	accessKeySecret = "384ae472-e660-4335-94df-2c3a1c674cca"
	/*
		if err := cmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(b)
		loginStr := string(b.String())
		fmt.Println(loginStr)
		fmt.Println("Logged into AST instance3")
	*/
	//fmt.Println(buf.String())
	/*
		var loginBytes []byte
		loginBytes, _ = ioutil.ReadAll(b)
		fmt.Println(loginBytes)
		loginStr := string(loginBytes)
		fmt.Println("Logged into AST instance2")
		fmt.Println(loginStr)
		authInfo := strings.Split(loginStr, "\n")
		for i, s := range authInfo {
			fmt.Println("Line")
			fmt.Println(i, s)
		}
	*/
}
