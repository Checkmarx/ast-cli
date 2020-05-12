package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
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
	err := bindKeyToEnvAndDefault(params.AstURIKey, params.AstURIEnv, "http://localhost:80")
	exitIfError(err)
	ast := viper.GetString(params.AstURIKey)

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

	err = bindKeyToEnvAndDefault(params.AccessKeyIDConfigKey, params.AccessKeyIDEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.AccessKeySecretConfigKey, params.AccessKeySecretEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.AstAuthenticationURIConfigKey, params.AstAuthenticationURIEnv, "")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.CredentialsFilePathKey, params.CredentialsFilePathEnv, "credentials.ast")
	exitIfError(err)

	err = bindKeyToEnvAndDefault(params.CredentialsFilePathKey, params.TokenExpirySecondsEnv, "300")
	exitIfError(err)

	scansURL := fmt.Sprintf("%s/%s", ast, scans)
	uploadsURL := fmt.Sprintf("%s/%s", ast, uploads)
	projectsURL := fmt.Sprintf("%s/%s", ast, projects)
	resultsURL := fmt.Sprintf("%s/%s", ast, results)
	sastrmURL := fmt.Sprintf("%s/%s", ast, sastrm)
	bflURL := fmt.Sprintf("%s/%s", ast, bfl)

	scansWrapper := wrappers.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploadsURL)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projectsURL)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(resultsURL)
	bflWrapper := wrappers.NewHTTPBFLWrapper(bflURL)
	rmWrapper := wrappers.NewSastRmHTTPWrapper(sastrmURL)

	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error finding executable path: %v", err)
	}
	executableDir := filepath.Dir(executablePath)
	dotEnvFilePath := path.Join(executableDir, ".env")
	scriptsDir := "./.scripts-test"
	installFilePath := "install.sh"
	upFilePath := "up.sh"
	downFilePath := "down.sh"

	scriptsWrapper := wrappers.NewScriptsFolderWrapper(dotEnvFilePath, scriptsDir, installFilePath, upFilePath, downFilePath)

	astCli := commands.NewAstCLI(
		scansWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		bflWrapper,
		rmWrapper,
		scriptsWrapper,
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
