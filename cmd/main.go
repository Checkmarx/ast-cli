package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/spf13/viper"
)

const (
	astURIEnv       = "AST_URI"
	scansPathEnv    = "SCANS_PATH"
	projectsPathEnv = "PROJECTS_PATH"
	resultsPathEnv  = "RESULTS_PATH"
	uploadsPathEnv  = "UPLOADS_PATH"

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

	uploadsPathKey := strings.ToLower(uploadsPathEnv)
	err = bindKeyToEnvAndDefault(uploadsPathKey, uploadsPathEnv, "api/uploads")
	exitIfError(err)
	uploads := viper.GetString(uploadsPathKey)

	err = bindKeyToEnvAndDefault(commands.AccessKeyIDConfigKey, commands.AccessKeyIDEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AccessKeySecretConfigKey, commands.AccessKeySecretEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AstAuthenticationURIConfigKey, commands.AstAuthenticationURIEnv, "")
	exitIfError(err)

	scansURL := fmt.Sprintf("%s/%s", ast, scans)
	uploadsURL := fmt.Sprintf("%s/%s", ast, uploads)
	projectsURL := fmt.Sprintf("%s/%s", ast, projects)
	resultsURL := fmt.Sprintf("%s/%s", ast, results)

	scansWrapper := wrappers.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploadsURL)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projectsURL)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(resultsURL)

	astCli := commands.NewAstCLI(scansWrapper, uploadsWrapper, projectsWrapper, resultsWrapper)

	err = astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(failureExitCode)
	}
}

func bindKeyToEnvAndDefault(key, env, defaultVal string) error {
	err := viper.BindEnv(key, env)
	viper.SetDefault(key, defaultVal)
	return err
}

// When building an executable for Windows and providing a name,
// be sure to explicitly specify the .exe suffix when setting the executable’s name.
// env GOOS=windows GOARCH=amd64 go build -o ./bin/ast.exe ./cmd
// "bin/ast.exe" -v scan create   --input-file  ./internal/commands/payloads/uploads.json --sources ./internal/commands/payloads/sources.zip
// "bin/ast.exe" scan list   4d9a9189-ddcc-4aa0-ba2f-9d6d7f92eceb
