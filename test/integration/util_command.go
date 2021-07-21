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
	sastResults := viper.GetString(params.SastResultsPathKey)
	kicsResults := viper.GetString(params.KicsResultsPathKey)
	uploads := viper.GetString(params.UploadsPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, sastResults, kicsResults, scans)
	authWrapper := wrappers.NewAuthHTTPWrapper()

	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		authWrapper,
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
	proxyPort := viper.GetInt(ProxyPortEnv)
	proxyHost := viper.GetString(ProxyHostEnv)

	args = append(args, flag(commands.VerboseFlag))
	args = append(args, flag(commands.ProxyFlag))
	args = append(args, fmt.Sprintf(ProxyURLTmpl, proxyUser, proxyPw, proxyHost, proxyPort))
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
