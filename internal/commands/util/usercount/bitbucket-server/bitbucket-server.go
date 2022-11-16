package bitbucket_server

import (
	"fmt"
	"log"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucket-server"
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
	bitBucketServerCommandShort      = "The bitbucket-server command presents the unique contributors for the provided Bitbucket Server projects and repositories."
	bitBucketServerFlagUrl           = "server-url"
	bitBucketServerFlagUrlUsage      = "BitBucket server instance URL"
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
	bitbucketServerUrl      *string
	bitbucketServerToken    *string
	format                  string
)

func NewUserCountBitBucketServerCommand(bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     bitBucketServerCommandName,
		Short:   bitBucketServerCommandShort,
		PreRunE: preRunBitBucketServerUserCount,
		RunE:    createRunBitBucketServerUserCountFunc(bitBucketServerWrapper),
	}

	bitbucketServerUrl = userCountCmd.Flags().String(bitBucketServerFlagUrl, "", bitBucketServerFlagUrlUsage)
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

	_ = userCountCmd.MarkFlagRequired(bitBucketServerFlagUrl)

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

	if !strings.HasSuffix(*bitbucketServerUrl, "/") {
		*bitbucketServerUrl += "/"
	}

	return nil
}

func createRunBitBucketServerUserCountFunc(bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper) func(
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

func searchBitBucketServer(bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper) (
	[]repositoryView,
	[]userView,
	error,
) {

	if len(bitBucketServerProjects) == 0 {
		var err error
		bitBucketServerProjects, err = bitBucketServerWrapper.GetProjects(
			*bitbucketServerUrl,
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
	bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper,
) ([]repositoryView, []userView, error) {

	var views []repositoryView
	var viewsUsers []userView
	var totalCommits []bitbucket_server.BitBucketServerCommit
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
	bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper,
	project string,
	views []repositoryView,
	viewsUsers []userView,
	totalCommits []bitbucket_server.BitBucketServerCommit,
) ([]bitbucket_server.BitBucketServerCommit, []repositoryView, []userView, error) {

	for _, repo := range bitbucketServerRepos {
		commits, err := bitBucketServerWrapper.GetCommits(
			*bitbucketServerUrl,
			project,
			repo,
			*bitbucketServerToken,
		)
		if err != nil {
			return nil, nil, nil, err
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

func getAllRepos(bitBucketServerWrapper bitbucket_server.BitBucketServerWrapper, project string) error {
	if len(bitbucketServerRepos) == 0 {
		bbRepos, err := bitBucketServerWrapper.GetRepositories(
			*bitbucketServerUrl,
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

func getUniqueContributorsBitBucketServer(commits []bitbucket_server.BitBucketServerCommit) map[string]string {
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

func bitBucketServerIsNotBot(commit bitbucket_server.BitBucketServerCommit) bool {
	return strings.Contains(commit.Author.Name, bitBucketServerBot)
}

func buildCountPathBitBucketServer(project string, repo string) string {
	return project + "/" + repo
}
