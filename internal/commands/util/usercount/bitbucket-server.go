package usercount

import (
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	bitBucketServerProjects []string
	bitbucketServerRepos    []string
	bitbucketServerUrl      *string
	bitbucketServerToken    *string
)

func newUserCountBitBucketServerCommand(bitBucketServerWrapper wrappers.BitBucketServerWrapper) *cobra.Command {
	userCountCmd := &cobra.Command{
		Use:     "bitbucket-server",
		Short:   "short",
		PreRunE: preRunBitBucketServerUserCount,
		RunE:    createRunBitBucketServerUserCountFunc(bitBucketServerWrapper),
	}

	bitbucketServerUrl = userCountCmd.Flags().String("base-url", "", "base-url")
	userCountCmd.Flags().StringSliceVar(
		&bitBucketServerProjects,
		"projects",
		[]string{},
		"projects",
	)
	userCountCmd.Flags().StringSliceVar(&bitbucketServerRepos, "repos", []string{}, "repos")
	bitbucketServerToken = userCountCmd.Flags().String("token", "", "token")

	_ = userCountCmd.MarkFlagRequired("base-url")
	_ = userCountCmd.MarkFlagRequired("token")

	return userCountCmd
}

func preRunBitBucketServerUserCount(*cobra.Command, []string) error {
	if len(bitbucketServerRepos) > 0 && len(bitBucketServerProjects) == 0 {
		return errors.New("Provide a project to get specific repos")
	}
	if len(bitbucketServerRepos) > 0 && len(bitBucketServerProjects) > 1 {
		return errors.New("Provide a single project to get specific repos")
	}
	return nil
}

func createRunBitBucketServerUserCountFunc(bitBucketServerWrapper wrappers.BitBucketServerWrapper) func(
	cmd *cobra.Command,
	args []string,
) error {
	return func(cmd *cobra.Command, args []string) error {
		var views []RepositoryView

		if len(bitBucketServerProjects) == 0 {
			var err error
			bitBucketServerProjects, err = bitBucketServerWrapper.GetProjects(
				*bitbucketServerUrl,
				*bitbucketServerToken,
			)
			if err != nil {
				return err
			}
		}

		var totalCommits []wrappers.BitBucketServerCommit
		for _, project := range bitBucketServerProjects {
			bbRepos, err := bitBucketServerWrapper.GetRepositories(
				*bitbucketServerUrl,
				project,
				*bitbucketServerToken,
			)
			if err != nil {
				return err
			}
			for _, repo := range bbRepos {
				commits, err := bitBucketServerWrapper.GetCommits(
					*bitbucketServerUrl,
					project,
					repo.Slug,
					*bitbucketServerToken,
				)
				if err != nil {
					return err
				}
				totalCommits = append(totalCommits, commits...)
			}
		}

		for _, commit := range totalCommits {
			logger.Printf("%v", commit)
		}

		views = append(
			views,
			RepositoryView{
				Name:               TotalContributorsName,
				UniqueContributors: uint64(len(getUniqueContributorsBitBucketServer(totalCommits))),
			},
		)

		_ = printer.Print(cmd.OutOrStdout(), views, format)

		/*list1, err := bitBucketServerWrapper.GetProjects(
			"http://ec2-44-195-19-186.compute-1.amazonaws.com",
			"Njc1Nzg1Mjg4NjI0Or3gCVm/jZLiwosd0NFwecNXX2Gi",
		)

		logger.Printf("%v", list1)

		list2, _ := bitBucketServerWrapper.GetRepositories(
			"http://ec2-44-195-19-186.compute-1.amazonaws.com",
			"DIOG",
			"Njc1Nzg1Mjg4NjI0Or3gCVm/jZLiwosd0NFwecNXX2Gi",
		)

		logger.Printf("%v", list2)

		list3, err := bitBucketServerWrapper.GetCommits(
			"http://ec2-44-195-19-186.compute-1.amazonaws.com",
			"DIOG",
			"ast-github-tester",
			"Njc1Nzg1Mjg4NjI0Or3gCVm/jZLiwosd0NFwecNXX2Gi",
		)

		logger.Printf("%v", list3)*/

		return nil
	}
}

func getUniqueContributorsBitBucketServer(commits []wrappers.BitBucketServerCommit) map[string]string {
	var contributors = map[string]string{}
	for _, commit := range commits {
		name := commit.Author.Name
		email := strings.ToLower(commit.Author.Email)
		if _, ok := contributors[email]; !ok {
			logger.Print(email)
			contributors[email] = name
		}
	}
	return contributors
}
