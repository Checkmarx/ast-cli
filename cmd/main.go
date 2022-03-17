package main

import (
	"log"
	"os"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/viper"
)

const (
	successfulExitCode = 0
	failureExitCode    = 1
)

func main() {
	var err error
	bindKeysToEnvAndDefault()
	configuration.LoadConfiguration()
	scans := viper.GetString(params.ScansPathKey)
	groups := viper.GetString(params.GroupsPathKey)
	logs := viper.GetString(params.LogsPathKey)
	projects := viper.GetString(params.ProjectsPathKey)
	results := viper.GetString(params.ResultsPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	codebashing := viper.GetString(params.CodeBashingPathKey)
	bfl := viper.GetString(params.BflPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	groupsWrapper := wrappers.NewHTTPGroupsWrapper(groups)
	logsWrapper := wrappers.NewLogsWrapper(logs)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	resultsPredicatesWrapper := wrappers.NewResultsPredicatesHTTPWrapper()
	codeBashingWrapper := wrappers.NewCodeBashingHTTPWrapper(codebashing)
	gitHubWrapper := wrappers.NewGitHubWrapper()
	azureWrapper := wrappers.NewAzureWrapper()
	bflWrapper := wrappers.NewBflHTTPWrapper(bfl)

	astCli := commands.NewAstCLI(
		scansWrapper,
		resultsPredicatesWrapper,
		codeBashingWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		authWrapper,
		logsWrapper,
		groupsWrapper,
		gitHubWrapper,
		azureWrapper,
		bflWrapper,
	)

	err = astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		switch e := err.(type) {
		case *wrappers.AstError:
			log.Println(e.Err)
			os.Exit(e.Code)
		default:
			log.Println(e)
			os.Exit(failureExitCode)
		}
	}
}

func bindKeysToEnvAndDefault() {
	for _, b := range params.EnvVarsBinds {
		err := viper.BindEnv(b.Key, b.Env)
		if err != nil {
			exitIfError(err)
		}
		viper.SetDefault(b.Key, b.Default)
	}
}
