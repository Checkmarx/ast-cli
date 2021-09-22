// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

//goland:noinspection HttpUrlsUsage
const (
	ProxyUserEnv = "PROXY_USERNAME"
	ProxyPwEnv   = "PROXY_PASSWORD"
	ProxyPortEnv = "PROXY_PORT"
	ProxyHostEnv = "PROXY_HOST"
	ProxyURLTmpl = "http://%s:%s@%s:%d"
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
	uploads := viper.GetString(params.UploadsPathKey)
	logs := viper.GetString(params.LogsPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	logsWrapper := wrappers.NewLogsWrapper(logs)

	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		authWrapper,
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

	args = append(args, flag(commands.VerboseFlag))
	args = appendProxyArgs(args)
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

func appendProxyArgs(args []string) []string {
	proxyUser := viper.GetString(ProxyUserEnv)
	proxyPw := viper.GetString(ProxyPwEnv)
	proxyPort := viper.GetInt(ProxyPortEnv)
	proxyHost := viper.GetString(ProxyHostEnv)
	argsWithProxy := append(args, flag(commands.ProxyFlag))
	argsWithProxy = append(argsWithProxy, fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyPw, proxyHost, proxyPort))
	return argsWithProxy
}
