package usercount

import (
	"log"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RepositoryView struct {
	Name               string `json:"name"`
	UniqueContributors uint64 `json:"unique_contributors"`
}
type UserView struct {
	Name                       string `json:"name"`
	UniqueContributorsUsername string `json:"unique_contributors_username"`
}

const (
	GithubCommand  = "github"
	githubShort    = "The github command presents the unique contributors for the provided GitHub repositories or organizations."
	ReposFlag      = "repos"
	reposFlagUsage = "List of repositories to scan for contributors"
	OrgsFlag       = "orgs"
	orgsFlagUsage  = "List of organizations to scan for contributors"
	githubAPIURL   = "https://api.github.com"
	sinceParam     = "since"
	missingArgs    = "provide at least one repository or organization"
	missingOrg     = "an organization is required for your repositories"
	tooManyOrgs    = "a single organization should be provided for specific repositories"
	botType        = "Bot"
)

var (
	repos, orgs []string
)

func newUserCountGithubCommand(gitHubWrapper wrappers.GitHubWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     GithubCommand,
		Short:   githubShort,
		PreRunE: preRunGithubUserCount,
		RunE:    createRunGitHubUserCountFunc(gitHubWrapper),
	}
	userCountCmd.Flags().String(params.SCMTokenFlag, "", params.GithubTokenUsage)
	userCountCmd.Flags().StringSliceVar(&repos, ReposFlag, []string{}, reposFlagUsage)
	userCountCmd.Flags().StringSliceVar(&orgs, OrgsFlag, []string{}, orgsFlagUsage)
	userCountCmd.Flags().String(params.URLFlag, githubAPIURL, params.URLFlagUsage)

	_ = viper.BindPFlag(params.URLFlag, userCountCmd.Flags().Lookup(params.URLFlag))

	return userCountCmd
}

func preRunGithubUserCount(*cobra.Command, []string) error {
	if len(repos) == 0 && len(orgs) == 0 {
		return errors.New(missingArgs)
	}

	if len(repos) > 0 && len(orgs) == 0 {
		return errors.New(missingOrg)
	}

	if len(repos) > 0 && len(orgs) > 1 {
		return errors.New(tooManyOrgs)
	}

	return nil
}

func createRunGitHubUserCountFunc(gitHubWrapper wrappers.GitHubWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		var totalCommits []wrappers.CommitRoot
		var views []RepositoryView
		var viewsUsers []UserView

		_ = viper.BindPFlag(params.SCMTokenFlag, cmd.Flags().Lookup(params.SCMTokenFlag))

		if len(repos) > 0 {
			totalCommits, views, viewsUsers, err = collectFromRepos(gitHubWrapper)
		} else {
			totalCommits, views, viewsUsers, err = collectFromOrgs(gitHubWrapper)
		}
		if err != nil {
			return err
		}

		uniqueContributorsMap := getUniqueContributors(totalCommits)

		views = append(
			views,
			RepositoryView{
				Name:               TotalContributorsName,
				UniqueContributors: uint64(len(uniqueContributorsMap)),
			},
		)

		err = printer.Print(cmd.OutOrStdout(), views, format)

		// Only print user count information if in debug mode
		if viper.GetBool(params.DebugFlag) {
			err = printer.Print(cmd.OutOrStdout(), viewsUsers, format)
		}

		log.Println(params.BotCount)

		return err
	}
}

func collectFromRepos(gitHubWrapper wrappers.GitHubWrapper) ([]wrappers.CommitRoot, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.CommitRoot
	var views []RepositoryView
	var viewsUsers []UserView
	for _, repo := range repos {
		repository, err := gitHubWrapper.GetRepository(orgs[0], repo)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		commits, err := gitHubWrapper.GetCommits(repository, map[string]string{sinceParam: ninetyDaysDate})
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		totalCommits = append(totalCommits, commits...)

		uniqueContributorsMap := getUniqueContributors(commits)

		views = append(
			views,
			RepositoryView{
				Name:               repository.FullName,
				UniqueContributors: uint64(len(uniqueContributorsMap)),
			},
		)
		for name := range uniqueContributorsMap {
			viewsUsers = append(
				viewsUsers,
				UserView{
					Name:                       repository.FullName,
					UniqueContributorsUsername: name,
				},
			)
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromOrgs(gitHubWrapper wrappers.GitHubWrapper) ([]wrappers.CommitRoot, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.CommitRoot
	var views []RepositoryView
	var viewsUsers []UserView

	for _, org := range orgs {
		organization, err := gitHubWrapper.GetOrganization(org)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		repositories, err := gitHubWrapper.GetRepositories(organization)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		for _, repository := range repositories {
			commits, err := gitHubWrapper.GetCommits(repository, map[string]string{sinceParam: ninetyDaysDate})
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}

			totalCommits = append(totalCommits, commits...)

			uniqueContributorsMap := getUniqueContributors(commits)

			views = append(
				views,
				RepositoryView{
					Name:               repository.FullName,
					UniqueContributors: uint64(len(uniqueContributorsMap)),
				},
			)
			for name := range uniqueContributorsMap {
				viewsUsers = append(
					viewsUsers,
					UserView{
						Name:                       repository.FullName,
						UniqueContributorsUsername: name,
					},
				)
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func getUniqueContributors(commits []wrappers.CommitRoot) map[string]bool {
	var contributors = map[string]bool{}
	for _, commit := range commits {
		name := commit.Commit.CommitAuthor.Name
		if !contributors[name] && isNotBot(commit) {
			contributors[name] = true
		}
	}
	return contributors
}

func isNotBot(commit wrappers.CommitRoot) bool {
	return commit.Author == nil || commit.Author.Type != botType
}
