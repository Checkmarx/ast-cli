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
	astSchemaEnv       = "AST_SCHEMA"
	astHostEnv         = "AST_HOST"
	astPortEnv         = "AST_PORT"
	scansPathEnv       = "SCANS_PATH"
	projectsPathEnv    = "PROJECTS_PATH"
	resultsPathEnv     = "RESULTS_PATH"
	uploadsPathEnv     = "UPLOADS_PATH"
	queuePathEnv       = "QUEUE_PATH"
	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	// Key ast_schema will be bound to AST_SCHEMA
	astSchemaKey := strings.ToLower(astSchemaEnv)
	err := bindKeyToEnvAndDefault(astSchemaKey, astSchemaEnv, "http")
	exitIfError(err)
	schema := viper.GetString(astSchemaKey)

	astHostKey := strings.ToLower(astHostEnv)
	err = bindKeyToEnvAndDefault(astHostKey, astHostEnv, "localhost")
	exitIfError(err)
	host := viper.GetString(astHostKey)

	astPortKey := strings.ToLower(astPortEnv)
	err = bindKeyToEnvAndDefault(astPortKey, astPortEnv, "80")
	exitIfError(err)
	port := viper.GetString(astPortKey)

	scansPathKey := strings.ToLower(scansPathEnv)
	err = bindKeyToEnvAndDefault(scansPathKey, scansPathEnv, "scans")
	exitIfError(err)
	scans := viper.GetString(scansPathKey)

	projectsPathKey := strings.ToLower(projectsPathEnv)
	err = bindKeyToEnvAndDefault(projectsPathKey, projectsPathEnv, "projects")
	exitIfError(err)
	projects := viper.GetString(projectsPathKey)

	resultsPathKey := strings.ToLower(resultsPathEnv)
	err = bindKeyToEnvAndDefault(resultsPathKey, resultsPathEnv, "results")
	exitIfError(err)
	results := viper.GetString(resultsPathKey)

	uploadsPathKey := strings.ToLower(uploadsPathEnv)
	err = bindKeyToEnvAndDefault(uploadsPathKey, uploadsPathEnv, "uploads")
	exitIfError(err)
	uploads := viper.GetString(uploadsPathKey)

	queuePathKey := strings.ToLower(queuePathEnv)
	err = bindKeyToEnvAndDefault(queuePathKey, queuePathEnv, "sast-rm")
	exitIfError(err)
	queue := viper.GetString(queuePathKey)

	err = bindKeyToEnvAndDefault(commands.AccessKeyIDConfigKey, commands.AccessKeyIDEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AccessKeySecretConfigKey, commands.AccessKeySecretEnv, "")
	exitIfError(err)
	err = bindKeyToEnvAndDefault(commands.AstAuthenticationHostConfigKey, commands.AstAuthenticationHostEnv, "")
	exitIfError(err)

	ast := fmt.Sprintf("%s://%s:%s/api", schema, host, port)

	scansURL := fmt.Sprintf("%s/%s", ast, scans)
	uploadsURL := fmt.Sprintf("%s/%s", ast, uploads)
	projectsURL := fmt.Sprintf("%s/%s", ast, projects)
	resultsURL := fmt.Sprintf("%s/%s", ast, results)
	queueURL := fmt.Sprintf("%s/%s", ast, queue)

	scansWrapper := wrappers.NewHTTPScansWrapper(scansURL)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploadsURL)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projectsURL)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(resultsURL)
	queueWrapper := wrappers.NewQueueHTTPWrapper(queueURL)

	astCli := commands.NewAstCLI(scansWrapper, uploadsWrapper, projectsWrapper, resultsWrapper, queueWrapper)

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
// "bin/ast.exe" -v scan create   --inputFile  ./internal/commands/payloads/uploads.json --sources ./internal/commands/payloads/sources.zip
// "bin/ast.exe" scan get  --id 4d9a9189-ddcc-4aa0-ba2f-9d6d7f92eceb
