package main

import (
	"fmt"
	commands "github.com/checkmarxDev/ast-cli/internal/commands"
	"github.com/spf13/viper"
	"os"
)

const (
	scansEp           = "SCANS_ENDPOINT"
	scansEpDisplay    = "Scans endpoint"
	projectsEp        = "PROJECTS_ENDPOINT"
	projectsEpDisplay = "Projects endpoint"
	resultsEp         = "RESULTS_ENDPOINT"
	resultsEpDisplay  = "Results endpoint"
	logLevel          = "CLI_LOG_LEVEL"
)

func main() {
	viper.AutomaticEnv()
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	viper.SetDefault(logLevel, "DEBUG")

	scans := viper.GetString(scansEp)
	if scans == "" {
		requiredErrAndExit(scansEp, scansEpDisplay)
	}
	projects := viper.GetString(projectsEp)
	if projects == "" {
		requiredErrAndExit(projectsEp, projectsEpDisplay)
	}
	results := viper.GetString(resultsEp)
	if results == "" {
		requiredErrAndExit(resultsEp, resultsEpDisplay)
	}

	astCli := commands.NewAstCLI(scans, projects, results)
	_ = astCli.Execute()

}

// When building an executable for Windows and providing a name,
// be sure to explicitly specify the .exe suffix when setting the executable’s name.
// env GOOS=windows GOARCH=amd64 go build -o ./bin/ast.exe ./cmd

func errorAndExit(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func requiredErrAndExit(key, displayName string) {
	errorAndExit(fmt.Sprintf("%s is required. Please specify it using the environment variable %s or in a config.env file.", displayName, key))
}
