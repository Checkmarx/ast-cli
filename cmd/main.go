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
	"github.com/spf13/viper"
)

const (
	astURIEnv              = "AST_URI"
	scansPathEnv           = "SCANS_PATH"
	projectsPathEnv        = "PROJECTS_PATH"
	resultsPathEnv         = "RESULTS_PATH"
	uploadsPathEnv         = "UPLOADS_PATH"
	bflPathEnv             = "BFL_PATH"
	sastRmPathEnv          = "SAST_RM_PATH"
	credentialsFilePathEnv = "CREDENTIALS_FILE_PATH"
	tokenExpirySecondsEnv  = "TOKEN_EXPIRY_SECONDS"

	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	// Key ast_uri will be bound to AST_URI
	astURIKey := strings.ToLower(astURIEnv)
	err := bindKeyToEnvAndDefault(astURIKey, astURIEnv, "http://localhost:80")
	exitIfError(err)
	ast := viper.GetString(astURIKey)

	scansPathKey := strings.ToLower(scansPathEnv)
	err = bindKeyToEnvAndDefault(scansPathKey, scansPathEnv, "api/scans")
	exitIfError(err)
	scans := viper.GetString(scansPathKey)

	projectsPathKey := strings.ToLower(projectsPathEnv)
	err = bindKeyToEnvAndDefault(projectsPathKey, projectsPathEnv, "api/projects")
	exitIfError(err)
	projects := viper.GetString(projectsPathKey)

	resultsPathKey := strings.ToLower(resultsPathEnv)
	err = bindKeyToEnvAndDefault(resultsPathKey, resultsPathEnv, "api/results")
	exitIfError(err)
	results := viper.GetString(resultsPathKey)

	bflPathKey := strings.ToLower(bflPathEnv)
	err = bindKeyToEnvAndDefault(bflPathKey, bflPathEnv, "api/bfl")
	exitIfError(err)
	bfl := viper.GetString(bflPathKey)

	uploadsPathKey := strings.ToLower(uploadsPathEnv)
	err = bindKeyToEnvAndDefault(uploadsPathKey, uploadsPathEnv, "api/uploads")
	exitIfError(err)
	uploads := viper.GetString(uploadsPathKey)

	sastRmPathKey := strings.ToLower(sastRmPathEnv)
	err = bindKeyToEnvAndDefault(sastRmPathKey, sastRmPathEnv, "api/sast-rm")
	exitIfError(err)
	sastrm := viper.GetString(sastRmPathKey)

	err = bindKeyToEnvAndDefault(commands.AccessKeyIDConfigKey, commands.AccessKeyIDEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AccessKeySecretConfigKey, commands.AccessKeySecretEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AstAuthenticationURIConfigKey, commands.AstAuthenticationURIEnv, "")
	exitIfError(err)

	credentialsFilePathKey := strings.ToLower(credentialsFilePathEnv)
	err = bindKeyToEnvAndDefault(credentialsFilePathKey, credentialsFilePathEnv, "credentials.ast")
	exitIfError(err)

	tokenExpirySecondsKey := strings.ToLower(tokenExpirySecondsEnv)
	err = bindKeyToEnvAndDefault(tokenExpirySecondsKey, tokenExpirySecondsEnv, "300")
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
