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
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

const (
	astURIEnv              = "AST_URI"
	scansPathEnv           = "SCANS_PATH"
	projectsPathEnv        = "PROJECTS_PATH"
	resultsPathEnv         = "RESULTS_PATH"
	uploadsPathEnv         = "UPLOADS_PATH"
	bflPathEnv             = "BFL_PATH"
	sastRmPathEnv          = "SAST_RM_PATH"
	letterBytes            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	credentialsFilePathEnv = "CREDENTIALS_FILE_PATH"
	tokenExpirySecondsEnv  = "TOKEN_EXPIRY_SECONDS"
	successfulExitCode     = 0
	failureExitCode        = 1
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
	astURIKey := strings.ToLower(astURIEnv)
	err := bindKeyToEnvAndDefault(astURIKey, astURIEnv, "http://localhost:80")
	assert.NilError(t, err)
	ast := viper.GetString(astURIKey)

	scansPathKey := strings.ToLower(scansPathEnv)
	err = bindKeyToEnvAndDefault(scansPathKey, scansPathEnv, "api/scans")
	assert.NilError(t, err)
	scans := viper.GetString(scansPathKey)

	projectsPathKey := strings.ToLower(projectsPathEnv)
	err = bindKeyToEnvAndDefault(projectsPathKey, projectsPathEnv, "api/projects")
	assert.NilError(t, err)
	projects := viper.GetString(projectsPathKey)

	resultsPathKey := strings.ToLower(resultsPathEnv)
	err = bindKeyToEnvAndDefault(resultsPathKey, resultsPathEnv, "api/results")
	assert.NilError(t, err)
	results := viper.GetString(resultsPathKey)

	uploadsPathKey := strings.ToLower(uploadsPathEnv)
	err = bindKeyToEnvAndDefault(uploadsPathKey, uploadsPathEnv, "api/uploads")
	assert.NilError(t, err)
	uploads := viper.GetString(uploadsPathKey)

	bflPathKey := strings.ToLower(bflPathEnv)
	err = bindKeyToEnvAndDefault(bflPathKey, bflPathEnv, "api/bfl")
	assert.NilError(t, err)
	bfl := viper.GetString(bflPathKey)

	sastRmPathKey := strings.ToLower(sastRmPathEnv)
	err = bindKeyToEnvAndDefault(sastRmPathKey, sastRmPathEnv, "api/sast-rm")
	assert.NilError(t, err)
	sastrm := viper.GetString(sastRmPathKey)

	err = bindKeyToEnvAndDefault(commands.AccessKeyIDConfigKey, commands.AccessKeyIDEnv, "")
	assert.NilError(t, err)
	err = bindKeyToEnvAndDefault(commands.AccessKeySecretConfigKey, commands.AccessKeySecretEnv, "")
	assert.NilError(t, err)
	err = bindKeyToEnvAndDefault(commands.AstAuthenticationURIConfigKey, commands.AstAuthenticationURIEnv, "")
	assert.NilError(t, err)

	credentialsFilePathKey := strings.ToLower(credentialsFilePathEnv)
	err = bindKeyToEnvAndDefault(credentialsFilePathKey, credentialsFilePathEnv, "credentials.ast")
	assert.NilError(t, err)

	tokenExpirySecondsKey := strings.ToLower(tokenExpirySecondsEnv)
	err = bindKeyToEnvAndDefault(tokenExpirySecondsKey, tokenExpirySecondsEnv, "300")
	assert.NilError(t, err)

	scansURL := fmt.Sprintf("%s/%s", ast, scans)
	uploadsURL := fmt.Sprintf("%s/%s", ast, uploads)
	projectsURL := fmt.Sprintf("%s/%s", ast, projects)
	resultsURL := fmt.Sprintf("%s/%s", ast, results)
	bflURL := fmt.Sprintf("%s/%s", ast, bfl)
	rmURL := fmt.Sprintf("%s/%s", ast, sastrm)

	scansWrapper := wrappers.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploadsURL)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projectsURL)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(resultsURL)
	bflWrapper := wrappers.NewHTTPBFLWrapper(bflURL)
	rmWrapper := wrappers.NewSastRmHTTPWrapper(rmURL)

	astCli := commands.NewAstCLI(scansWrapper, uploadsWrapper, projectsWrapper, resultsWrapper, bflWrapper, rmWrapper)
	return astCli
}

func execute(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}
