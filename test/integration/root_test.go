// +build integration

package integration

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
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

func bindKeyToEnvAndDefault(key, env, defaultVal string) error {
	err := viper.BindEnv(key, env)
	viper.SetDefault(key, defaultVal)
	return err
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
	// Run all tests
	exitVal := m.Run()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}

func createASTIntegrationTestCommand(t *testing.T) *cobra.Command {
	err := bindKeyToEnvAndDefault(params.BaseURIKey, params.BaseURIEnv, "http://localhost:80")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.ScansPathKey, params.ScansPathEnv, "api/scans")
	assert.NilError(t, err)
	scans := viper.GetString(params.ScansPathKey)

	err = bindKeyToEnvAndDefault(params.ProjectsPathKey, params.ProjectsPathEnv, "api/projects")
	assert.NilError(t, err)
	projects := viper.GetString(params.ProjectsPathKey)

	err = bindKeyToEnvAndDefault(params.ResultsPathKey, params.ResultsPathEnv, "api/results")
	assert.NilError(t, err)
	results := viper.GetString(params.ResultsPathKey)

	err = bindKeyToEnvAndDefault(params.BflPathKey, params.BflPathEnv, "api/bfl")
	assert.NilError(t, err)
	bfl := viper.GetString(params.BflPathKey)

	err = bindKeyToEnvAndDefault(params.UploadsPathKey, params.UploadsPathEnv, "api/uploads")
	assert.NilError(t, err)
	uploads := viper.GetString(params.UploadsPathKey)

	sastRmPathKey := strings.ToLower(params.SastRmPathEnv)
	err = bindKeyToEnvAndDefault(sastRmPathKey, params.SastRmPathEnv, "api/sast-rm")
	assert.NilError(t, err)
	sastrm := viper.GetString(sastRmPathKey)

	err = bindKeyToEnvAndDefault(params.AstWebAppHealthCheckPathKey, params.AstWebAppHealthCheckPathEnv, "#/projects")
	assert.NilError(t, err)
	webAppHlthChk := viper.GetString(params.AstWebAppHealthCheckPathKey)

	err = bindKeyToEnvAndDefault(params.AstKeycloakWebAppHealthCheckPathKey, params.AstKeycloakWebAppHealthCheckPathEnv,
		"auth/admin/organization/console/#/realms/organization/users")
	assert.NilError(t, err)
	keyCloakWebAppHlthChk := viper.GetString(params.AstKeycloakWebAppHealthCheckPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckPathKey, params.HealthcheckPathEnv, "api/healthcheck")
	assert.NilError(t, err)
	healthcheck := viper.GetString(params.HealthcheckPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckDBPathKey, params.HealthcheckDBPathEnv, "database")
	assert.NilError(t, err)
	healthcheckDBPath := viper.GetString(params.HealthcheckDBPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckMessageQueuePathKey,
		params.HealthcheckMessageQueuePathEnv, "message-queue")
	assert.NilError(t, err)
	healthcheckMessageQueuePath := viper.GetString(params.HealthcheckMessageQueuePathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckObjectStorePathKey,
		params.HealthcheckObjectStorePathEnv, "object-store")
	assert.NilError(t, err)
	healthcheckObjectStorePath := viper.GetString(params.HealthcheckObjectStorePathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckInMemoryDBPathKey,
		params.HealthcheckInMemoryDBPathEnv, "in-memory-db")
	assert.NilError(t, err)
	healthcheckInMemoryDBPath := viper.GetString(params.HealthcheckInMemoryDBPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckLoggingPathKey,
		params.HealthcheckLoggingPathEnv, "logging")
	assert.NilError(t, err)
	healthcheckLoggingPath := viper.GetString(params.HealthcheckLoggingPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckScanFlowPathKey, params.HealthcheckScanFlowPathEnv,
		"scan-flow")
	assert.NilError(t, err)
	healthcheckScanFlowPath := viper.GetString(params.HealthcheckScanFlowPathKey)

	err = bindKeyToEnvAndDefault(params.HealthcheckSastEnginesPathKey, params.HealthcheckSastEnginesPathEnv,
		"sast-engines")
	assert.NilError(t, err)
	healthcheckSastEnginesPath := viper.GetString(params.HealthcheckSastEnginesPathKey)

	err = bindKeyToEnvAndDefault(params.QueriesPathKey, params.QueriesPathEnv, "api/queries")
	assert.NilError(t, err)
	queries := viper.GetString(params.QueriesPathKey)

	err = bindKeyToEnvAndDefault(params.QueriesClonePathKey, params.QueriesCLonePathEnv, "clone")
	assert.NilError(t, err)
	queriesClone := viper.GetString(params.QueriesClonePathKey)

	err = bindKeyToEnvAndDefault(params.CreateOath2ClientPathKey, params.CreateOath2ClientPathEnv, "auth/realms/organization/pip/clients")
	assert.NilError(t, err)
	createClientPath := viper.GetString(params.CreateOath2ClientPathKey)

	err = bindKeyToEnvAndDefault(params.AccessKeyIDConfigKey, params.AccessKeyIDEnv, "")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.AccessKeySecretConfigKey, params.AccessKeySecretEnv, "")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.AstAuthenticationPathConfigKey, params.AstAuthenticationPathEnv,
		"auth/realms/organization/protocol/openid-connect/token")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.CredentialsFilePathKey, params.CredentialsFilePathEnv, "credentials.ast")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.TokenExpirySecondsKey, params.TokenExpirySecondsEnv, "300")
	assert.NilError(t, err)

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
		authWrapper,
		defaultConfigFileLocation,
	)
	return astCli
}

func execute(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}
