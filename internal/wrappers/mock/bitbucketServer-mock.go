package mock

import (
	"fmt"
	"log"

	"github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
	"github.com/pkg/errors"
)

type MockWrapperBitbucketServer struct {
	CorruptedRepos []string
}

type MockRepositoryView struct {
	Name               string `json:"name"`
	UniqueContributors uint64 `json:"unique_contributors"`
}

type MockUserView struct {
	Name                       string `json:"name"`
	UniqueContributorsUsername string `json:"unique_contributors_username"`
}

func (m MockWrapperBitbucketServer) GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) ([]bitbucketserver.BitbucketServerCommit, error) {

	for _, corruptedRepo := range m.CorruptedRepos {
		if repoSlug == corruptedRepo {
			return nil, errors.New(fmt.Sprintf("repository %s is corrupted", repoSlug))
		}
	}

	return []bitbucketserver.BitbucketServerCommit{
		{
			Author:          bitbucketserver.BitbucketServerAuthor{Name: "Mock Author", Email: "mock-author@example.com"},
			AuthorTimestamp: 1625078400000,
		},
	}, nil
}

func (m MockWrapperBitbucketServer) GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) ([]bitbucketserver.BitbucketServerRepo, error) {

	return []bitbucketserver.BitbucketServerRepo{
		{Slug: "repo-1", Name: "Repository 1"},
		{Slug: "repo-2", Name: "Repository 2"},
		{Slug: "repo-3", Name: "Repository 3"},
	}, nil
}

func (m MockWrapperBitbucketServer) GetProjects(bitBucketURL, bitBucketPassword string) ([]string, error) {
	// Return mock projects
	return []string{"project-1", "project-2"}, nil
}

func (m MockWrapperBitbucketServer) SearchRepos(
	project string,
	repos []string,
	bitBucketToken string,
) ([]MockRepositoryView, []MockUserView, error) {
	var views []MockRepositoryView
	var viewsUsers []MockUserView

	for _, repo := range repos {

		_, err := m.GetCommits("mock-url", project, repo, bitBucketToken)
		if err != nil {
			log.Printf("Skipping repository %s/%s: Repository is corrupted (error: %v)", project, repo, err)
			continue
		}

		log.Printf("Processed repository %s/%s", project, repo)

		uniqueContributors := map[string]string{
			"mock-email@example.com": "Mock Author",
		}
		views = append(
			views,
			MockRepositoryView{
				Name:               fmt.Sprintf("%s/%s", project, repo),
				UniqueContributors: uint64(len(uniqueContributors)),
			},
		)
		for email, name := range uniqueContributors {
			viewsUsers = append(
				viewsUsers,
				MockUserView{
					Name:                       fmt.Sprintf("%s/%s", project, repo),
					UniqueContributorsUsername: fmt.Sprintf("%s - %s", name, email),
				},
			)
		}
	}
	return views, viewsUsers, nil
}
