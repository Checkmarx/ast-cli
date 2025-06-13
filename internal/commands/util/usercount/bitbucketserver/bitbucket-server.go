//
// Tests were not added to bitbucket server contributor count as there is no testing instance of bitbucket server available
//

package bitbucketserver

import (
	"fmt"
	"log"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type repositoryView struct {
	Name               string `json:"name"`
	UniqueContributors uint64 `json:"unique_contributors"`
}
type userView struct {
	Name                       string `json:"name"`
	UniqueContributorsUsername string `json:"unique_contributors_username"`
}

const (
	bitBucketServerCommandName       = "bitbucket-server"
	bitBucketServerCommandShort      = "The BitBucket Server command presents the unique contributors for the provided Bitbucket Server projects and repositories"
	bitBucketServerFlagURL           = "server-url"
	bitBucketServerFlagURLUsage      = "BitBucket Server instance URL"
	bitBucketServerFlagProjects      = "projects"
	bitBucketServerFlagProjectsUsage = "Projects to search for contributors"
	bitBucketServerFlagRepos         = "repos"
	bitBucketServerFlagReposUsage    = "Repositories to search for contributors"
	bitBucketServerFlagToken         = "token"
	bitBucketServerFlagTokenUsage    = "Authentication token. Will search public projects if not provided"
	bitBucketServerBot               = "[bot]"
	bitBucketReposProjectError       = "Provide a single project to get repos"
	totalContributorsName            = "Total unique contributors"
)

var (
	bitBucketServerProjects []string
	bitbucketServerRepos    []string
	bitbucketServerURL      *string
	bitbucketServerToken    *string
	format                  string
)

func NewUserCountBitBucketServerCommand(bitBucketServerWrapper bitbucketserver.Wrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     bitBucketServerCommandName,
		Short:   bitBucketServerCommandShort,
		PreRunE: preRunBitBucketServerUserCount,
		RunE:    createRunBitBucketServerUserCountFunc(bitBucketServerWrapper),
	}

	bitbucketServerURL = userCountCmd.Flags().String(bitBucketServerFlagURL, "", bitBucketServerFlagURLUsage)
	userCountCmd.Flags().StringSliceVar(
		&bitBucketServerProjects,
		bitBucketServerFlagProjects,
		[]string{},
		bitBucketServerFlagProjectsUsage,
	)
	userCountCmd.Flags().StringSliceVar(
		&bitbucketServerRepos,
		bitBucketServerFlagRepos,
		[]string{},
		bitBucketServerFlagReposUsage,
	)
	bitbucketServerToken = userCountCmd.Flags().String(bitBucketServerFlagToken, "", bitBucketServerFlagTokenUsage)

	_ = userCountCmd.MarkFlagRequired(bitBucketServerFlagURL)

	userCountCmd.Flags().StringVar(
		&format,
		params.FormatFlag,
		printer.FormatTable,
		fmt.Sprintf(
			params.FormatFlagUsageFormat,
			[]string{printer.FormatTable, printer.FormatJSON, printer.FormatList},
		),
	)

	return userCountCmd
}

func preRunBitBucketServerUserCount(*cobra.Command, []string) error {
	if len(bitbucketServerRepos) > 0 && len(bitBucketServerProjects) != 1 {
		return errors.New(bitBucketReposProjectError)
	}

	if !strings.HasSuffix(*bitbucketServerURL, "/") {
		*bitbucketServerURL += "/"
	}

	return nil
}

func createRunBitBucketServerUserCountFunc(bitBucketServerWrapper bitbucketserver.Wrapper) func(
	cmd *cobra.Command,
	args []string,
) error {
	return func(cmd *cobra.Command, args []string) error {
		views, viewsUsers, err := searchBitBucketServer(bitBucketServerWrapper)
		if err != nil {
			return err
		}

		err = printer.Print(cmd.OutOrStdout(), views, format)
		if err != nil {
			return err
		}

		// Only print user count information if in debug mode
		if viper.GetBool(params.DebugFlag) {
			err = printer.Print(cmd.OutOrStdout(), viewsUsers, format)
		}

		log.Println(params.BotCount)

		return err
	}
}

func searchBitBucketServer(bitBucketServerWrapper bitbucketserver.Wrapper) (
	[]repositoryView,
	[]userView,
	error,
) {
	if len(bitBucketServerProjects) == 0 {
		var err error
		bitBucketServerProjects, err = bitBucketServerWrapper.GetProjects(
			*bitbucketServerURL,
			*bitbucketServerToken,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	views, viewsUsers, err := searchProjects(
		bitBucketServerWrapper,
	)
	if err != nil {
		return nil, nil, err
	}

	return views, viewsUsers, nil
}

func searchProjects(
	bitBucketServerWrapper bitbucketserver.Wrapper,
) ([]repositoryView, []userView, error) {
	var views []repositoryView
	var viewsUsers []userView
	var totalCommits []bitbucketserver.Commit
	for _, project := range bitBucketServerProjects {
		err := getAllRepos(bitBucketServerWrapper, project)
		if err != nil {
			return nil, nil, err
		}

		totalCommits, views, viewsUsers, err = searchRepos(
			bitBucketServerWrapper,
			project,
			views,
			viewsUsers,
			totalCommits,
		)
		if err != nil {
			return nil, nil, err
		}

		// if user provided repos, it's only for a single project, so we can wipe the flag value
		bitbucketServerRepos = []string{}
	}

	views = append(
		views,
		repositoryView{
			Name:               totalContributorsName,
			UniqueContributors: uint64(len(getUniqueContributorsBitBucketServer(totalCommits))),
		},
	)
	return views, viewsUsers, nil
}

func searchRepos(
	bitBucketServerWrapper bitbucketserver.Wrapper,
	project string,
	views []repositoryView,
	viewsUsers []userView,
	totalCommits []bitbucketserver.Commit,
) ([]bitbucketserver.Commit, []repositoryView, []userView, error) {
	for _, repo := range bitbucketServerRepos {
		commits, err := bitBucketServerWrapper.GetCommits(
			*bitbucketServerURL,
			project,
			repo,
			*bitbucketServerToken,
		)
		if err != nil {
			log.Printf("Skipping repository %s/%s: Repository is corrupted (error: %v)", project, repo, err)
			continue
		}
		totalCommits = append(totalCommits, commits...)

		uniqueContributors := getUniqueContributorsBitBucketServer(commits)

		if len(uniqueContributors) > 0 {
			views = append(
				views,
				repositoryView{
					Name:               buildCountPathBitBucketServer(project, repo),
					UniqueContributors: uint64(len(uniqueContributors)),
				},
			)
			for email, name := range uniqueContributors {
				viewsUsers = append(
					viewsUsers,
					userView{
						Name:                       buildCountPathBitBucketServer(project, repo),
						UniqueContributorsUsername: fmt.Sprintf("%s - %s", name, email),
					},
				)
			}
		}
	}
	return totalCommits, views, viewsUsers, nil
}

func getAllRepos(bitBucketServerWrapper bitbucketserver.Wrapper, project string) error {
	if len(bitbucketServerRepos) == 0 {
		bbRepos, err := bitBucketServerWrapper.GetRepositories(
			*bitbucketServerURL,
			project,
			*bitbucketServerToken,
		)
		if err != nil {
			return err
		}
		for _, repo := range bbRepos {
			bitbucketServerRepos = append(bitbucketServerRepos, repo.Slug)
		}
	}
	return nil
}

func getUniqueContributorsBitBucketServer(commits []bitbucketserver.Commit) map[string]string {
	var contributors = map[string]string{}
	for _, commit := range commits {
		name := commit.Author.Name
		email := strings.ToLower(commit.Author.Email)
		if _, ok := contributors[email]; !ok && !bitBucketServerIsNotBot(commit) {
			contributors[email] = name
		}
	}
	return contributors
}

func bitBucketServerIsNotBot(commit bitbucketserver.Commit) bool {
	return strings.Contains(commit.Author.Name, bitBucketServerBot)
}

func buildCountPathBitBucketServer(project, repo string) string {
	return project + "/" + repo
}
