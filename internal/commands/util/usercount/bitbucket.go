package usercount

import (
	"log"
	"regexp"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	BitBucketCommand             = "bitbucket"
	BitBucketShort               = "The bitbucket command presents the unique contributors for the provided Bitbucket organizations, projects and repositories"
	workspaceFlag                = "workspaces"
	workspaceFlagUsage           = "List of workspaces to scan for contributors"
	urlBitbucket                 = "url-bitbucket"
	bitbucketAPIURL              = "https://api.bitbucket.org/2.0/"
	usernameFlagUsage            = "Username for Bitbucket authentication"
	passwordFlagUsage            = "App password for Bitbucket authentication.Requires read on “Workspace membership“ and “Repositories“ permissions"
	missingWorkspace             = "Provide at least one workspace"
	bitbucketManyWorkspaceOnRepo = "You must provide a single workspace for repo counting"
	bitBucketBot                 = "[bot]"
)

var (
	BitBucketWorkspaces []string
	BitBucketRepos      []string
	BitBucketURL        *string
	BitBucketPassword   *string
	BitBucketUsername   *string
)

func newUserCountBitBucketCommand(bitBucketWrapper wrappers.BitBucketWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     BitBucketCommand,
		Short:   BitBucketShort,
		PreRunE: preRunBitBucketUserCount,
		RunE:    createRunBitBucketUserCountFunc(bitBucketWrapper),
	}

	BitBucketURL = userCountCmd.Flags().String(urlBitbucket, bitbucketAPIURL, params.URLFlagUsage)
	userCountCmd.PersistentFlags().StringSliceVar(&BitBucketWorkspaces, workspaceFlag, []string{}, workspaceFlagUsage)
	userCountCmd.Flags().StringSliceVar(&BitBucketRepos, ReposFlag, []string{}, reposFlagUsage)
	BitBucketUsername = userCountCmd.Flags().String(params.UsernameFlag, "", usernameFlagUsage)
	BitBucketPassword = userCountCmd.Flags().String(params.PasswordFlag, "", passwordFlagUsage)

	return userCountCmd
}

func preRunBitBucketUserCount(*cobra.Command, []string) error {
	if len(BitBucketWorkspaces) == 0 {
		return errors.New(missingWorkspace)
	}
	if len(BitBucketRepos) > 0 && len(BitBucketWorkspaces) == 0 {
		return errors.New(missingWorkspace)
	}
	if len(BitBucketRepos) > 0 && len(BitBucketWorkspaces) > 1 {
		return errors.New(bitbucketManyWorkspaceOnRepo)
	}
	return nil
}

func createRunBitBucketUserCountFunc(bitBucketWrapper wrappers.BitBucketWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var totalCommits []wrappers.BitBucketRootCommit
		var err error
		var totalContrib uint64 = 0
		var views []RepositoryView
		var viewsUsers []UserView

		// In case there is the repos flag
		if len(BitBucketRepos) > 0 {
			totalCommits, views, viewsUsers, err = collectFromBitBucketRepos(bitBucketWrapper)
		} else {
			totalCommits, views, viewsUsers, err = collectFromBitBucketWorkspace(bitBucketWrapper)
		}
		if err != nil {
			return err
		}
		for _, commits := range totalCommits {
			totalContrib += uint64(len(getUniqueContributorsBitbucket(commits)))
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
		if viper.GetBool(params.DebugFlag) || viper.GetString(params.LogFileFlag) != "" || viper.GetString(params.LogFileConsoleFlag) != "" {
			err = printer.Print(cmd.OutOrStdout(), viewsUsers, format)
		}

		log.Println(params.BotCount)

		return err
	}
}

func collectFromBitBucketRepos(bitBucketWrapper wrappers.BitBucketWrapper) ([]wrappers.BitBucketRootCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.BitBucketRootCommit
	var views []RepositoryView
	var viewsUsers []UserView
	for _, workspace := range BitBucketWorkspaces {
		// Get the workspace uuid
		workspaceUUID, err := bitBucketWrapper.GetworkspaceUUID(*BitBucketURL, workspace, *BitBucketUsername, *BitBucketPassword)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}
		for _, repo := range BitBucketRepos {
			// Get the repo uuid
			repoObject, err := bitBucketWrapper.GetRepoUUID(*BitBucketURL, workspace, repo, *BitBucketUsername, *BitBucketPassword)
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}
			commits, err := bitBucketWrapper.GetCommits(*BitBucketURL, workspaceUUID.UUID, repoObject.UUID, *BitBucketUsername, *BitBucketPassword)
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}
			totalCommits = append(totalCommits, commits)
			uniqueContributors := getUniqueContributorsBitbucket(commits)

			// Case there is no organization, project, commits or repos inside the workspace
			if uint64(len(uniqueContributors)) > 0 {
				views = append(
					views,
					RepositoryView{
						Name:               buildBitBucketCountPath(workspace, repo),
						UniqueContributors: uint64(len(uniqueContributors)),
					},
				)

				for name := range uniqueContributors {
					viewsUsers = append(
						viewsUsers,
						UserView{
							Name:                       buildBitBucketCountPath(workspace, repo),
							UniqueContributorsUsername: cleanUsername(name),
							UniqueContributorsEmail:    cleanEmail(name),
						},
					)
				}
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func collectFromBitBucketWorkspace(bitBucketWrapper wrappers.BitBucketWrapper) ([]wrappers.BitBucketRootCommit, []RepositoryView, []UserView, error) {
	var totalCommits []wrappers.BitBucketRootCommit
	var views []RepositoryView
	var viewsUsers []UserView
	for _, workspace := range BitBucketWorkspaces {
		// Get the workspace uuid
		workspaceUUID, err := bitBucketWrapper.GetworkspaceUUID(*BitBucketURL, workspace, *BitBucketUsername, *BitBucketPassword)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}
		// Get repos from workspace
		var reposList wrappers.BitBucketRootRepoList
		reposList, err = bitBucketWrapper.GetRepositories(*BitBucketURL, workspace, *BitBucketUsername, *BitBucketPassword)
		if err != nil {
			return totalCommits, views, viewsUsers, err
		}
		for _, repo := range reposList.Values {
			// Get commits for a specific repo
			commits, err := bitBucketWrapper.GetCommits(*BitBucketURL, workspaceUUID.UUID, repo.UUID, *BitBucketUsername, *BitBucketPassword)
			if err != nil {
				return totalCommits, views, viewsUsers, err
			}
			totalCommits = append(totalCommits, commits)
			uniqueContributors := getUniqueContributorsBitbucket(commits)

			// Case there is no organization, project, commits or repos inside the workspace
			if uint64(len(uniqueContributors)) > 0 {
				views = append(
					views,
					RepositoryView{
						Name:               repo.Name,
						UniqueContributors: uint64(len(uniqueContributors)),
					},
				)

				for name := range uniqueContributors {
					viewsUsers = append(
						viewsUsers,
						UserView{
							Name:                       repo.Name,
							UniqueContributorsUsername: cleanUsername(name),
							UniqueContributorsEmail:    cleanEmail(name),
						},
					)
				}
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func getUniqueContributorsBitbucket(commits wrappers.BitBucketRootCommit) map[string]bool {
	var contributors = map[string]bool{}
	for _, commit := range commits.Commits {
		name := commit.Author.Name
		if !contributors[name] && !bitBucketIsNotBot(commit) {
			contributors[name] = true
		}
	}
	return contributors
}

func bitBucketIsNotBot(commit wrappers.BitBucketCommit) bool {
	return strings.Contains(commit.Author.Name, bitBucketBot)
}

func buildBitBucketCountPath(workspace, repo string) string {
	return workspace + "/" + repo
}

func cleanUsername(username string) string {
	var re = regexp.MustCompile(`<(\w+|.|@)+>`)
	cleaned := re.ReplaceAllString(username, "")
	return cleaned
}

func cleanEmail(email string) string {
	var re = regexp.MustCompile(`<(\w+|.|@)+>`)
	cleanedSplit := re.FindAllString(email, 1)
	cleaned := ""
	if len(cleanedSplit) > 0 {
		cleaned = cleanedSplit[0]
		cleaned = strings.Replace(cleaned, "<", "", 1)
		cleaned = strings.Replace(cleaned, ">", "", 1)
	}
	return cleaned
}
