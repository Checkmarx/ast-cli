package main

import (
	"fmt"
	"os"

	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/spf13/viper"
)

const (
	astSchema          = "AST_SCHEMA"
	astHost            = "AST_HOST"
	astPort            = "80"
	scansPath          = "SCANS_PATH"
	projectsPath       = "PROJECTS_PATH"
	resultsPath        = "RESULTS_PATH"
	uploadsPath        = "UPLOADS_PATH"
	logLevel           = "CLI_LOG_LEVEL"
	successfulExitCode = 0
	// failureExitCode    = 1
)

func main() {
	viper.AutomaticEnv()
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	viper.SetDefault(astSchema, "http")
	viper.SetDefault(astHost, "localhost")
	viper.SetDefault(astPort, "80")
	viper.SetDefault(scansPath, "scans")
	viper.SetDefault(projectsPath, "projects")
	viper.SetDefault(uploadsPath, "uploads")
	viper.SetDefault(resultsPath, "results")
	viper.SetDefault(logLevel, "DEBUG")

	schema := viper.GetString(astSchema)
	host := viper.GetString(astHost)
	port := viper.GetString(astPort)
	ast := fmt.Sprintf("%s://%s:%s/api", schema, host, port)

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
