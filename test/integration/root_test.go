// +build integration

package integration

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	astSchemaEnv       = "AST_SCHEMA"
	astHostEnv         = "AST_HOST"
	astPortEnv         = "AST_PORT"
	scansPathEnv       = "SCANS_PATH"
	projectsPathEnv    = "PROJECTS_PATH"
	resultsPathEnv     = "RESULTS_PATH"
	uploadsPathEnv     = "UPLOADS_PATH"
	letterBytes        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	successfulExitCode = 0
	failureExitCode    = 1
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
	astSchemaKey := strings.ToLower(astSchemaEnv)
	err := bindKeyToEnvAndDefault(astSchemaKey, astSchemaEnv, "http")
	assert.NilError(t, err)
	schema := viper.GetString(astSchemaKey)

	astHostKey := strings.ToLower(astHostEnv)
	err = bindKeyToEnvAndDefault(astHostKey, astHostEnv, "localhost")
	assert.NilError(t, err)
	host := viper.GetString(astHostKey)

	astPortKey := strings.ToLower(astPortEnv)
	err = bindKeyToEnvAndDefault(astPortKey, astPortEnv, "80")
	assert.NilError(t, err)
	port := viper.GetString(astPortKey)

	scansPathKey := strings.ToLower(scansPathEnv)
	err = bindKeyToEnvAndDefault(scansPathKey, scansPathEnv, "scans")
	assert.NilError(t, err)
	scans := viper.GetString(scansPathKey)

	projectsPathKey := strings.ToLower(projectsPathEnv)
	err = bindKeyToEnvAndDefault(projectsPathKey, projectsPathEnv, "projects")
	assert.NilError(t, err)
	projects := viper.GetString(projectsPathKey)

	resultsPathKey := strings.ToLower(resultsPathEnv)
	err = bindKeyToEnvAndDefault(resultsPathKey, resultsPathEnv, "results")
	assert.NilError(t, err)
	results := viper.GetString(resultsPathKey)

	uploadsPathKey := strings.ToLower(uploadsPathEnv)
	err = bindKeyToEnvAndDefault(uploadsPathKey, uploadsPathEnv, "uploads")
	assert.NilError(t, err)
	uploads := viper.GetString(uploadsPathKey)

	err = bindKeyToEnvAndDefault(commands.AccessKeyIDConfigKey, commands.AccessKeyIDEnv, "")
	assert.NilError(t, err)
	err = bindKeyToEnvAndDefault(commands.AccessKeySecretConfigKey, commands.AccessKeySecretEnv, "")
	assert.NilError(t, err)
	err = bindKeyToEnvAndDefault(commands.AstAuthenticationURIConfigKey, commands.AstAuthenticationURIEnv, "")
	assert.NilError(t, err)

	ast := fmt.Sprintf("%s://%s:%s/api", schema, host, port)

	scansURL := fmt.Sprintf("%s/%s", ast, scans)
	uploadsURL := fmt.Sprintf("%s/%s", ast, uploads)
	projectsURL := fmt.Sprintf("%s/%s", ast, projects)
	resultsURL := fmt.Sprintf("%s/%s", ast, results)

	scansWrapper := wrappers.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploadsURL)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projectsURL)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(resultsURL)

	astCli := commands.NewAstCLI(scansWrapper, uploadsWrapper, projectsWrapper, resultsWrapper)
	return astCli
}

func execute(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}
