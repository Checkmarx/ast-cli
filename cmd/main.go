package main

import (
	"fmt"
	"os"

	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/spf13/viper"
)

const (
	astEp              = "AST_ENDPOINT"
	astEpDisplay       = "AST endpoint"
	scansPath          = "SCANS_PATH"
	projectsPath       = "PROJECTS_PATH"
	resultsPath        = "RESULTS_PATH"
	uploadsPath        = "UPLOADS_PATH"
	logLevel           = "CLI_LOG_LEVEL"
	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	viper.AutomaticEnv()
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	viper.SetDefault(scansPath, "scans")
	viper.SetDefault(projectsPath, "projects")
	viper.SetDefault(uploadsPath, "uploads")
	viper.SetDefault(resultsPath, "results")
	viper.SetDefault(logLevel, "DEBUG")

	ast := viper.GetString(astEp)
	if ast == "" {
		requiredErrAndExit(astEp, astEpDisplay)
	}
	scans := viper.GetString(scansPath)
	projects := viper.GetString(projectsPath)
	uploads := viper.GetString(uploadsPath)
	results := viper.GetString(resultsPath)

	astCli := commands.NewAstCLI(ast, scans, projects, uploads, results)
	_ = astCli.Execute()
	os.Exit(successfulExitCode)
}

// When building an executable for Windows and providing a name,
// be sure to explicitly specify the .exe suffix when setting the executable’s name.
// env GOOS=windows GOARCH=amd64 go build -o ./bin/ast.exe ./cmd
// "bin/ast.exe" scan create  --sources c:\CODE\ast\example\sources.zip  --inputFile  ./payloads/uploads.json
// "bin/ast.exe" scan get  --id 4d9a9189-ddcc-4aa0-ba2f-9d6d7f92eceb

func errorAndExit(msg string) {
	fmt.Println(msg)
	os.Exit(failureExitCode)
}

func requiredErrAndExit(key, displayName string) {
	errorAndExit(fmt.Sprintf("%s is required. Please specify it using the environment variable %s or in a config.env file.", displayName, key))
}
