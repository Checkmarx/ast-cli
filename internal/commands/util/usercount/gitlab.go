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

type gitLabRepositoryView struct {
	Name               string `json:"name"`
	UniqueContributors uint64 `json:"unique_contributors"`
}
type gitLabUserView struct {
	Name                       string `json:"name"`
	UniqueContributorsUsername string `json:"unique_contributors_username"`
}

const (
	GitLabCommand                  = "gitlab"
	gitLabShort                    = "The gitlab command presents the unique contributors for the provided GitLab repositories or groups."
	gitLabProjectsFlag             = "projects"
	gitLabProjectsFlagUsage        = "List of projects(repos) to scan for contributors"
	gitLabGroupsFlag               = "groups"
	gitLabGroupsFlagUsage          = "List of groups (organizations)  to scan for contributors"
	gitLabAPIURL                   = "https://gitlab.com"
	gitLabsinceParam               = "since"
	gitLabMissingArgs              = "Please provide at least one project or group name."
	gitLabMissingGroup             = "A group is required for your projects"
	gitLabTooManyGroupsandProjects = "Projects and Groups both cannot be provided at the same time."
	gitLabBotType                  = "Bot"
)

var (
	gitLabProjects, gitLabGroups []string
)

func newUserCountGitLabCommand(gitLabWrapper wrappers.GitLabWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     GitLabCommand,
		Short:   gitLabShort,
		PreRunE: preRunGitLabUserCount,
		RunE:    createRunGitLabUserCountFunc(gitLabWrapper),
	}

	userCountCmd.Flags().StringSliceVar(&gitLabProjects, gitLabProjectsFlag, []string{}, gitLabProjectsFlagUsage)
	userCountCmd.Flags().StringSliceVar(&gitLabGroups, gitLabGroupsFlag, []string{}, gitLabGroupsFlagUsage)
	userCountCmd.Flags().String(params.URLFlag, gitLabAPIURL, params.URLFlagUsage)
	userCountCmd.Flags().String(params.SCMTokenFlag, "", params.GitLabTokenUsage)
	_ = viper.BindPFlag(params.URLFlag, userCountCmd.Flags().Lookup(params.URLFlag))

	return userCountCmd
}

func preRunGitLabUserCount(*cobra.Command, []string) error {
	// COmmenting to allow running the command only with the token so as to get User's projects
	// if len(gitLabProjects) == 0 && len(gitLabGroups) == 0 {
	// 	return errors.New(gitLabMissingArgs)
	// }

	// if len(gitLabProjects) > 0 && len(gitLabGroups) == 0 {
	// 	return errors.New(gitLabMissingOrg)
	// }

	if len(gitLabProjects) > 0 && len(gitLabGroups) > 0 {
		return errors.New(gitLabTooManyGroupsandProjects)
	}

	return nil
}

func createRunGitLabUserCountFunc(gitLabWrapper wrappers.GitLabWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		var totalCommits []wrappers.GitLabCommit
		var views []RepositoryView
		var viewsUsers []UserView
		//var totalContrib uint64 = 0

		_ = viper.BindPFlag(params.SCMTokenFlag, cmd.Flags().Lookup(params.SCMTokenFlag))

		if len(gitLabProjects) > 0 {
			log.Println("Collecting from GitLab projects only...")
			totalCommits, views, viewsUsers, err = collectFromGitLabProjects(gitLabWrapper)
		} else if len(gitLabGroups) > 0 {
			log.Println("Collecting from GitLab groups only...")
			totalCommits, views, viewsUsers, err = collectFromGitLabGroups(gitLabWrapper)
		} else {
			log.Println("Collecting from User'projects only...")
			totalCommits, views, viewsUsers, err = collectFromUser(gitLabWrapper)
		}

		if err != nil {
			return err
		}

		// for _, commits := range totalCommits {
		// 	totalContrib += uint64(len(getGitLabUniqueContributors(commits)))
		// }

		// views = append(
		// 	views,
		// 	RepositoryView{
		// 		Name:               TotalContributorsName,
		// 		UniqueContributors: totalContrib,
		// 	},
		// )

		uniqueContributorsMap := getGitLabUniqueContributors(totalCommits)

		views = append(
			views,
			RepositoryView{
				Name:               TotalContributorsName,
				UniqueContributors: uint64(len(uniqueContributorsMap)),
			},
		)

		err = printer.Print(cmd.OutOrStdout(), views, format)

		//Only print user count information if in debug mode
		if viper.GetBool(params.DebugFlag) {
			err = printer.Print(cmd.OutOrStdout(), viewsUsers, format)
		}

		log.Println("Note: dependabot is not counted but other bots might be considered users.")

		return err
	}
}

func collectFromGitLabProjects(gitLabWrapper wrappers.GitLabWrapper) ([]wrappers.GitLabCommit, []RepositoryView, []UserView, error) {

	var totalCommits []wrappers.GitLabCommit
	var views []RepositoryView
	var viewsUsers []UserView

	for _, gitLabProjectName := range gitLabProjects {

		commits, err := gitLabWrapper.GetCommits(gitLabProjectName, map[string]string{sinceParam: ninetyDaysDate})
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		totalCommits = append(totalCommits, commits...)

		uniqueContributorsMap := getGitLabUniqueContributors(commits)

		views = append(views,
			RepositoryView{
				Name:               gitLabProjectName,
				UniqueContributors: uint64(len(uniqueContributorsMap)),
			},
		)
		for name := range uniqueContributorsMap {
			viewsUsers = append(viewsUsers,
				UserView{
					Name:                       gitLabProjectName,
					UniqueContributorsUsername: name,
				},
			)
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromGitLabGroups(gitLabWrapper wrappers.GitLabWrapper) ([]wrappers.GitLabCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.GitLabCommit
	var views []RepositoryView
	var viewsUsers []UserView

	for _, gitLabGroupName := range gitLabGroups {
		gitLabGroupsFound, err := gitLabWrapper.GetGitLabGroups(gitLabGroupName)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		for _, gitLabGroup := range gitLabGroupsFound {

			gitLabProjects, err := gitLabWrapper.GetGitLabProjects(gitLabGroup, map[string]string{})

			if err != nil {
				return totalCommits, views, viewsUsers, err
			}

			for _, gitLabProject := range gitLabProjects {

				commits, err := gitLabWrapper.GetCommits(gitLabProject.PathWithNameSpace, map[string]string{sinceParam: ninetyDaysDate})
				if err != nil {
					return totalCommits, views, viewsUsers, err
				}

				totalCommits = append(totalCommits, commits...)

				uniqueContributorsMap := getGitLabUniqueContributors(commits)

				views = append(
					views,
					RepositoryView{
						Name:               gitLabProject.PathWithNameSpace,
						UniqueContributors: uint64(len(uniqueContributorsMap)),
					},
				)
				for name := range uniqueContributorsMap {
					viewsUsers = append(
						viewsUsers,
						UserView{
							Name:                       gitLabProject.PathWithNameSpace,
							UniqueContributorsUsername: name,
						},
					)
				}

			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromUser(gitLabWrapper wrappers.GitLabWrapper) ([]wrappers.GitLabCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.GitLabCommit
	var views []RepositoryView
	var viewsUsers []UserView

	gitLabProjects, err := gitLabWrapper.GetGitLabProjectsForUser()
	if err != nil {
		return totalCommits, views, viewsUsers, err
	}

	for _, gitLabProjectName := range gitLabProjects {

		commits, err := gitLabWrapper.GetCommits(gitLabProjectName.PathWithNameSpace, map[string]string{sinceParam: ninetyDaysDate})
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}

		totalCommits = append(totalCommits, commits...)

		uniqueContributorsMap := getGitLabUniqueContributors(commits)

		views = append(views,
			RepositoryView{
				Name:               gitLabProjectName.PathWithNameSpace,
				UniqueContributors: uint64(len(uniqueContributorsMap)),
			},
		)
		for name := range uniqueContributorsMap {
			viewsUsers = append(viewsUsers,
				UserView{
					Name:                       gitLabProjectName.PathWithNameSpace,
					UniqueContributorsUsername: name,
				},
			)
		}
	}

	return totalCommits, views, viewsUsers, nil
}

func getGitLabUniqueContributors(commits []wrappers.GitLabCommit) map[string]bool {
	var contributors = map[string]bool{}

	for _, commit := range commits {
		name := commit.Name
		if !contributors[name] && !isNotGitLabBot(commit) {
			contributors[name] = true
		}
	}
	return contributors
}

func isNotGitLabBot(commit wrappers.GitLabCommit) bool {
	return strings.Contains(commit.Name, "bot")
}
