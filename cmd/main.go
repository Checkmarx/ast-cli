package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/checkmarx/ast-cli/internal/commands"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/viper"
)

const (
	successfulExitCode = 0
	failureExitCode    = 1
	killCommand        = "kill"
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
	scaPackage := viper.GetString(params.ScaPackagePathKey)
	risksOverview := viper.GetString(params.RisksOverviewPathKey)
	uploads := viper.GetString(params.UploadsPathKey)
	codebashing := viper.GetString(params.CodeBashingPathKey)
	bfl := viper.GetString(params.BflPathKey)
	prDecorationGithubPath := viper.GetString(params.PRDecorationGithubPathKey)
	descriptionsPath := viper.GetString(params.DescriptionsPathKey)
	tenantConfigurationPath := viper.GetString(params.TenantConfigurationPathKey)

	scansWrapper := wrappers.NewHTTPScansWrapper(scans)
	groupsWrapper := wrappers.NewHTTPGroupsWrapper(groups)
	logsWrapper := wrappers.NewLogsWrapper(logs)
	uploadsWrapper := wrappers.NewUploadsHTTPWrapper(uploads)
	projectsWrapper := wrappers.NewHTTPProjectsWrapper(projects)
	risksOverviewWrapper := wrappers.NewHTTPRisksOverviewWrapper(risksOverview)
	resultsWrapper := wrappers.NewHTTPResultsWrapper(results, scaPackage)
	authWrapper := wrappers.NewAuthHTTPWrapper()
	resultsPredicatesWrapper := wrappers.NewResultsPredicatesHTTPWrapper()
	codeBashingWrapper := wrappers.NewCodeBashingHTTPWrapper(codebashing)
	gitHubWrapper := wrappers.NewGitHubWrapper()
	azureWrapper := wrappers.NewAzureWrapper()
	bitBucketWrapper := wrappers.NewBitbucketWrapper()
	bitBucketServerWrapper := bitbucketserver.NewBitbucketServerWrapper()
	gitLabWrapper := wrappers.NewGitLabWrapper()
	bflWrapper := wrappers.NewBflHTTPWrapper(bfl)
	prWrapper := wrappers.NewHTTPPRWrapper(prDecorationGithubPath)
	learnMoreWrapper := wrappers.NewHTTPLearnMoreWrapper(descriptionsPath)
	tenantConfigurationWrapper := wrappers.NewHTTPTenantConfigurationWrapper(tenantConfigurationPath)

	astCli := commands.NewAstCLI(
		scansWrapper,
		resultsPredicatesWrapper,
		codeBashingWrapper,
		uploadsWrapper,
		projectsWrapper,
		resultsWrapper,
		risksOverviewWrapper,
		authWrapper,
		logsWrapper,
		groupsWrapper,
		gitHubWrapper,
		azureWrapper,
		bitBucketWrapper,
		bitBucketServerWrapper,
		gitLabWrapper,
		bflWrapper,
		prWrapper,
		learnMoreWrapper,
		tenantConfigurationWrapper,
	)
	exitListener()
	err = astCli.Execute()
	exitIfError(err)
	os.Exit(successfulExitCode)
}

func exitIfError(err error) {
	if err != nil {
		switch e := err.(type) {
		case *wrappers.AstError:
			fmt.Println(e.Err)
			os.Exit(e.Code)
		default:
			fmt.Println(e)
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

func exitListener() {
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(
		signalChanel,
		syscall.SIGTERM,
	)
	go signalHandler(signalChanel)
}

func signalHandler(signalChanel chan os.Signal) {
	kicsRunArgs := []string{
		killCommand,
		viper.GetString(params.KicsContainerNameKey),
	}
	for {
		s := <-signalChanel
		switch s {
		case syscall.SIGTERM:
			out, err := exec.Command("docker", "ps").CombinedOutput()
			if err != nil {
				os.Exit(failureExitCode)
			}
			logger.PrintIfVerbose(string(out))
			if strings.Contains(string(out), viper.GetString(params.KicsContainerNameKey)) {
				out, err = exec.Command("docker", kicsRunArgs...).CombinedOutput()
				logger.PrintIfVerbose(string(out))
				if err != nil {
					os.Exit(failureExitCode)
				}
			}
			os.Exit(successfulExitCode)
		// Should not get here since we only listen to SIGTERM, kept for safety
		default:
			os.Exit(failureExitCode)
		}
	}
}
