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
	err := bindKeyToEnvAndDefault(params.AstURIKey, params.AstURIEnv, "http://localhost:80")
	assert.NilError(t, err)
	ast := viper.GetString(params.AstURIKey)

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

	sastRmPathKey := strings.ToLower(sastRmPathEnv)
	err = bindKeyToEnvAndDefault(sastRmPathKey, sastRmPathEnv, "api/sast-rm")
	assert.NilError(t, err)
	sastrm := viper.GetString(sastRmPathKey)

	err = bindKeyToEnvAndDefault(params.AccessKeyIDConfigKey, params.AccessKeyIDEnv, "")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.AccessKeySecretConfigKey, params.AccessKeySecretEnv, "")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.AstAuthenticationURIConfigKey, params.AstAuthenticationURIEnv, "")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.CredentialsFilePathKey, params.CredentialsFilePathEnv, "credentials.ast")
	assert.NilError(t, err)

	err = bindKeyToEnvAndDefault(params.TokenExpirySecondsKey, params.TokenExpirySecondsEnv, "300")
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
