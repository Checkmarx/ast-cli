package usercount

import (
	"log"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AzureCommand        = "azure"
	azureShort          = "The azure command presents the unique contributors for the provided Azure repositories or organizations."
	projectFlag         = "projects"
	url                 = "url-azure"
	projectFlagUsage    = "List of projects to scan for contributors"
	azureAPIURL         = "https://dev.azure.com/"
	missingOrganization = "Provide at least one organization"
	missingProject      = "Provide at least one project"
	azureBot            = "[bot]"
	azureManyOrgsOnRepo = "You must provide a single org for repo counting"
)

var (
	AzureRepos   []string
	AzureOrgs    []string
	AzureProject []string
	AzureURL     *string
	AzureToken   *string
)

func newUserCountAzureCommand(azureWrapper wrappers.AzureWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     AzureCommand,
		Short:   azureShort,
		PreRunE: preRunAzureUserCount,
		RunE:    createRunAzureUserCountFunc(azureWrapper),
	}

	userCountCmd.PersistentFlags().StringSliceVar(&AzureOrgs, OrgsFlag, []string{}, orgsFlagUsage)
	userCountCmd.Flags().StringSliceVar(&AzureProject, projectFlag, []string{}, projectFlagUsage)
	userCountCmd.Flags().StringSliceVar(&AzureRepos, ReposFlag, []string{}, reposFlagUsage)
	AzureURL = userCountCmd.Flags().String(url, azureAPIURL, params.URLFlagUsage)
	AzureToken = userCountCmd.Flags().String(params.SCMTokenFlag, "", params.AzureTokenUsage)

	return userCountCmd
}

func preRunAzureUserCount(*cobra.Command, []string) error {
	if len(AzureOrgs) == 0 {
		return errors.New(missingOrganization)
	}
	if len(AzureRepos) > 0 && len(AzureProject) == 0 {
		return errors.New(missingProject)
	}
	if len(AzureRepos) > 0 && len(AzureOrgs) > 1 {
		return errors.New(azureManyOrgsOnRepo)
	}
	return nil
}

func createRunAzureUserCountFunc(azureWrapper wrappers.AzureWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var totalCommits []wrappers.AzureRootCommit
		var err error
		var totalContrib uint64 = 0
		var views []RepositoryView
		var viewsUsers []UserView

		// In case there is the repos flag
		if len(AzureRepos) > 0 {
			totalCommits, views, viewsUsers, err = collectFromAzureRepos(azureWrapper)
		} else {
			// In case there is not the repos flag and there is the project flag
			if len(AzureProject) > 0 {
				totalCommits, views, viewsUsers, err = collectFromAzureProject(azureWrapper)
				// In case there is only the orgs flag
			} else {
				totalCommits, views, viewsUsers, err = collectFromAzureOrg(azureWrapper)
			}
		}

		if err != nil {
			return err
		}

		for _, commits := range totalCommits {
			totalContrib += uint64(len(getUniqueContributorsAzure(commits)))
		}

		views = append(
			views,
			RepositoryView{
				Name:               TotalContributorsName,
				UniqueContributors: totalContrib,
			},
		)

		err = printer.Print(cmd.OutOrStdout(), views, format)

		// Only print user count information if in debug mode
		if viper.GetBool(params.DebugFlag) {
			err = printer.Print(cmd.OutOrStdout(), viewsUsers, format)
		}

		log.Println("Note: dependabot is not counted but other bots might be considered as contributors.")

		return err
	}
}

func collectFromAzureRepos(azureWrapper wrappers.AzureWrapper) ([]wrappers.AzureRootCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.AzureRootCommit
	var views []RepositoryView
	var viewsUsers []UserView
	for _, org := range AzureOrgs {
		for _, project := range AzureProject {
			for _, repo := range AzureRepos {
				commits, err := azureWrapper.GetCommits(*AzureURL, org, project, repo, *AzureToken)
				if err != nil {
					return totalCommits, views, viewsUsers, err
				}
				totalCommits = append(totalCommits, commits)
				uniqueContributors := getUniqueContributorsAzure(commits)

				// Case there is no organization, project, commits or repos inside the organization
				if uint64(len(uniqueContributors)) > 0 {
					views = append(
						views,
						RepositoryView{
							Name:               buildCountPath(org, project, repo),
							UniqueContributors: uint64(len(uniqueContributors)),
						},
					)

					for name := range uniqueContributors {
						viewsUsers = append(
							viewsUsers,
							UserView{
								Name:                       buildCountPath(org, project, repo),
								UniqueContributorsUsername: name,
							},
						)
					}
				}
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromAzureProject(azureWrapper wrappers.AzureWrapper) ([]wrappers.AzureRootCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.AzureRootCommit
	var views []RepositoryView
	var viewsUsers []UserView
	for _, org := range AzureOrgs {
		for _, project := range AzureProject {
			// Fetch all the repos within the project
			repos, err := azureWrapper.GetRepositories(*AzureURL, org, project, *AzureToken)
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}
			// For each repo within the project fetch the commits
			for _, repo := range repos.Repos {
				commits, err := azureWrapper.GetCommits(*AzureURL, org, project, repo.Name, *AzureToken)
				if err != nil {
					return totalCommits, views, viewsUsers, err
				}
				totalCommits = append(totalCommits, commits)
				uniqueContributors := getUniqueContributorsAzure(commits)
				views = append(
					views,
					RepositoryView{
						Name:               buildCountPath(org, project, repo.Name),
						UniqueContributors: uint64(len(uniqueContributors)),
					},
				)
				for name := range uniqueContributors {
					viewsUsers = append(
						viewsUsers,
						UserView{
							Name:                       buildCountPath(org, project, repo.Name),
							UniqueContributorsUsername: name,
						},
					)
				}
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromAzureOrg(azureWrapper wrappers.AzureWrapper) ([]wrappers.AzureRootCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.AzureRootCommit
	var views []RepositoryView
	var viewsUsers []UserView
	// Fetch all the projects within the organization
	for _, org := range AzureOrgs {
		projects, err := azureWrapper.GetProjects(*AzureURL, org, *AzureToken)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}
		for _, project := range projects.Projects {
			// Fetch all the repos within the project
			repos, err := azureWrapper.GetRepositories(*AzureURL, org, project.Name, *AzureToken)
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}
			// For each repo within the project fetch the commits
			for _, repo := range repos.Repos {
				commits, err := azureWrapper.GetCommits(*AzureURL, org, project.Name, repo.Name, *AzureToken)
				if err != nil {
					return totalCommits, views, viewsUsers, err
				}
				totalCommits = append(totalCommits, commits)
				uniqueContributors := getUniqueContributorsAzure(commits)

				views = append(
					views,
					RepositoryView{
						Name:               buildCountPath(org, project.Name, repo.Name),
						UniqueContributors: uint64(len(uniqueContributors)),
					},
				)
				for name := range uniqueContributors {
					viewsUsers = append(
						viewsUsers,
						UserView{
							Name:                       buildCountPath(org, project.Name, repo.Name),
							UniqueContributorsUsername: name,
						},
					)
				}
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func getUniqueContributorsAzure(commits wrappers.AzureRootCommit) map[string]bool {
	var contributors = map[string]bool{}
	for _, commit := range commits.Commits {
		name := commit.Author.Name
		if !contributors[name] && !azureIsNotBot(commit) {
			contributors[name] = true
		}
	}
	return contributors
}

func azureIsNotBot(commit wrappers.AzureCommit) bool {
	return strings.Contains(commit.Author.Name, azureBot)
}

func buildCountPath(org, project, repo string) string {
	return org + "/" + project + "/" + repo
}
